package app

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz/lzma"
	"golang.org/x/image/bmp"
)

func newTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: uint8((x + y) % 256), A: 255})
		}
	}
	return img
}

func encodeVarint7bit(v int) []byte {
	var out []byte
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		out = append(out, b)
		if v == 0 {
			break
		}
	}
	return out
}

func encodeLZMA(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	w, err := lzma.NewWriter(&buf)
	if err != nil {
		t.Fatalf("new lzma writer: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write lzma data: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close lzma writer: %v", err)
	}
	return buf.Bytes()
}

func makeCIPAsset(t *testing.T, lzmaStream []byte) []byte {
	t.Helper()

	header := []byte{0x00, 0x00, 0xC1, 0xA7, 0xC0, 0xDE, 0x01}
	header = append(header, encodeVarint7bit(len(lzmaStream))...)
	return append(header, lzmaStream...)
}

func makeCIPAssetFromImage(t *testing.T, img image.Image) []byte {
	t.Helper()

	var bmpBuf bytes.Buffer
	if err := bmp.Encode(&bmpBuf, img); err != nil {
		t.Fatalf("encode bmp: %v", err)
	}
	return makeCIPAsset(t, encodeLZMA(t, bmpBuf.Bytes()))
}

func makeCIPAssetFromBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	return makeCIPAsset(t, encodeLZMA(t, payload))
}

func decodePNG(t *testing.T, path string) image.Image {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open png: %v", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	return img
}

func compareImages(t *testing.T, got image.Image, want image.Image) {
	t.Helper()

	if got.Bounds() != want.Bounds() {
		t.Fatalf("image bounds mismatch: got %v want %v", got.Bounds(), want.Bounds())
	}
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			wantPixel := color.NRGBAModel.Convert(want.At(x, y)).(color.NRGBA)
			gotPixel := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA)
			if wantPixel != gotPixel {
				t.Fatalf("pixel mismatch at (%d,%d): got %#v want %#v", x, y, gotPixel, wantPixel)
			}
		}
	}
}

func writeCIPFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write asset %s: %v", name, err)
	}
}

func TestConvertAssetCreatesPNGFromCompressedBMP(t *testing.T) {
	assetsDir := t.TempDir()
	outputDir := t.TempDir()

	const (
		filename   = "sprite.bin"
		firstID    = 10
		lastID     = 12
		outputName = "Sprites-10-12.png"
	)

	srcImg := newTestImage(4, 3)
	writeCIPFile(t, assetsDir, filename, makeCIPAssetFromImage(t, srcImg))

	if err := convertAsset(assetsDir, outputDir, filename, firstID, lastID); err != nil {
		t.Fatalf("convertAsset returned error: %v", err)
	}

	got := decodePNG(t, filepath.Join(outputDir, outputName))
	compareImages(t, got, srcImg)
}

func TestConvertAssetReturnsNilForMissingFile(t *testing.T) {
	assetsDir := t.TempDir()
	outputDir := t.TempDir()

	if err := convertAsset(assetsDir, outputDir, "missing.bin", 1, 1); err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, "Sprites-1-1.png")); !os.IsNotExist(err) {
		t.Fatalf("expected no output file, stat err=%v", err)
	}
}

func TestConvertAssetReturnsErrorForInvalidBMP(t *testing.T) {
	assetsDir := t.TempDir()
	outputDir := t.TempDir()

	const filename = "corrupt.bin"
	writeCIPFile(t, assetsDir, filename, makeCIPAssetFromBytes(t, []byte("not a bmp")))

	if err := convertAsset(assetsDir, outputDir, filename, 5, 6); err == nil {
		t.Fatalf("expected error for invalid BMP data")
	}
}

func TestSkipCIPHeaderSkipsZerosConstantAndVarint(t *testing.T) {
	payload := []byte{0x00, 0x00, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE}
	payload = append(payload, 0x81, 0x01) // two-byte varint
	payload = append(payload, 0xFF)

	r := bufio.NewReader(bytes.NewReader(payload))
	if err := skipCIPHeader(r); err != nil {
		t.Fatalf("skipCIPHeader returned error: %v", err)
	}
	b, err := r.ReadByte()
	if err != nil {
		t.Fatalf("expected remaining byte: %v", err)
	}
	if b != 0xFF {
		t.Fatalf("unexpected byte after header: got 0x%X want 0xFF", b)
	}
}

func TestSkipCIPHeaderErrorsOnUnexpectedEOF(t *testing.T) {
	r := bufio.NewReader(bytes.NewReader([]byte{0x00, 0x00}))
	if err := skipCIPHeader(r); err == nil {
		t.Fatalf("expected error for truncated header")
	}
}

func TestNewLZMAReaderDecodesStream(t *testing.T) {
	stream := encodeLZMA(t, []byte("hello world"))
	r, err := newLZMAReader(bufio.NewReader(bytes.NewReader(stream)))
	if err != nil {
		t.Fatalf("newLZMAReader error: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("unexpected decompressed data: %q", string(data))
	}
}

func TestNewLZMAReaderErrorsOnShortHeader(t *testing.T) {
	if _, err := newLZMAReader(bufio.NewReader(bytes.NewReader([]byte{0x01, 0x02, 0x03}))); err == nil {
		t.Fatalf("expected error for short header")
	}
}

func TestConvertAssetsFromCatalogContent(t *testing.T) {
	assetsDir := t.TempDir()
	outputDir := t.TempDir()
	tempDir := t.TempDir()

	imgA := newTestImage(4, 4)
	imgB := newTestImage(2, 3)
	writeCIPFile(t, assetsDir, "spriteA.bin", makeCIPAssetFromImage(t, imgA))
	writeCIPFile(t, assetsDir, "spriteB.bin", makeCIPAssetFromImage(t, imgB))

	catalog := `[
                {"type":"sprite","file":"spriteA.bin","spritetype":0,"firstspriteid":1,"lastspriteid":2,"area":0},
                {"type":"effect","file":"ignore.bin","spritetype":0,"firstspriteid":3,"lastspriteid":3,"area":0},
                {"type":"sprite","file":"spriteB.bin","spritetype":0,"firstspriteid":5,"lastspriteid":5,"area":0}
        ]`
	catalogPath := filepath.Join(tempDir, "content.json")
	if err := os.WriteFile(catalogPath, []byte(catalog), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	ConvertAssetsFromCatalogContent(assetsDir, catalogPath, outputDir)

	gotA := decodePNG(t, filepath.Join(outputDir, "Sprites-1-2.png"))
	compareImages(t, gotA, imgA)

	gotB := decodePNG(t, filepath.Join(outputDir, "Sprites-5-5.png"))
	compareImages(t, gotB, imgB)
}

func readSpriteBounds(t *testing.T, dir string, id int) image.Rectangle {
	t.Helper()

	path := filepath.Join(dir, fmt.Sprintf("%d.png", id))
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open sprite %d: %v", id, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode sprite %d: %v", id, err)
	}
	return img.Bounds()
}

func TestSplitSpriteSheetUses64PixelTilesForSmallSheets(t *testing.T) {
	img := newTestImage(384, 384)
	outputDir := t.TempDir()

	const (
		firstID = 100
		lastID  = 103
	)

	if err := SplitSpriteSheet(img, firstID, lastID, outputDir); err != nil {
		t.Fatalf("SplitSpriteSheet returned error: %v", err)
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != lastID-firstID+1 {
		t.Fatalf("expected %d sprites, got %d", lastID-firstID+1, len(entries))
	}

	for id := firstID; id <= lastID; id++ {
		bounds := readSpriteBounds(t, outputDir, id)
		if got, want := bounds.Dx(), 64; got != want {
			t.Fatalf("sprite %d width = %d, want %d", id, got, want)
		}
		if got, want := bounds.Dy(), 64; got != want {
			t.Fatalf("sprite %d height = %d, want %d", id, got, want)
		}
	}
}

func TestSplitSpriteSheetUses32PixelTilesForLargeSheets(t *testing.T) {
	img := newTestImage(384, 384)
	outputDir := t.TempDir()

	const (
		firstID = 200
		lastID  = 239
	)

	if err := SplitSpriteSheet(img, firstID, lastID, outputDir); err != nil {
		t.Fatalf("SplitSpriteSheet returned error: %v", err)
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != lastID-firstID+1 {
		t.Fatalf("expected %d sprites, got %d", lastID-firstID+1, len(entries))
	}

	for id := firstID; id <= lastID; id++ {
		bounds := readSpriteBounds(t, outputDir, id)
		if got, want := bounds.Dx(), 32; got != want {
			t.Fatalf("sprite %d width = %d, want %d", id, got, want)
		}
		if got, want := bounds.Dy(), 32; got != want {
			t.Fatalf("sprite %d height = %d, want %d", id, got, want)
		}
	}
}

func TestSplitSpriteSheetTruncatesWhenCountExceedsCapacity(t *testing.T) {
	img := newTestImage(96, 32)
	outputDir := t.TempDir()

	const (
		firstID = 300
		lastID  = 360
	)

	if err := SplitSpriteSheet(img, firstID, lastID, outputDir); err != nil {
		t.Fatalf("SplitSpriteSheet returned error: %v", err)
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	const expected = 3 // 96x32 sheet with 32px tiles holds 3 sprites
	if len(entries) != expected {
		t.Fatalf("expected %d sprites, got %d", expected, len(entries))
	}

	for i := 0; i < expected; i++ {
		id := firstID + i
		bounds := readSpriteBounds(t, outputDir, id)
		if got, want := bounds.Dx(), 32; got != want {
			t.Fatalf("sprite %d width = %d, want %d", id, got, want)
		}
		if got, want := bounds.Dy(), 32; got != want {
			t.Fatalf("sprite %d height = %d, want %d", id, got, want)
		}
	}

	if _, err := os.Stat(filepath.Join(outputDir, fmt.Sprintf("%d.png", firstID+expected))); err == nil {
		t.Fatalf("unexpected sprite %d generated", firstID+expected)
	}
}
