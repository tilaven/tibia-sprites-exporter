package app

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadVarintDecodesValues(t *testing.T) {
	var buf [10]byte
	n := binary.PutUvarint(buf[:], 300)

	got, next, ok := readVarint(buf[:n], 0)
	if !ok {
		t.Fatalf("readVarint reported failure")
	}
	if got != 300 {
		t.Fatalf("readVarint decoded %d, want 300", got)
	}
	if next != n {
		t.Fatalf("readVarint next index = %d, want %d", next, n)
	}
}

func TestReadVarintFailsOnTruncatedInput(t *testing.T) {
	buf := []byte{0x80}

	if _, _, ok := readVarint(buf, 0); ok {
		t.Fatalf("readVarint succeeded on truncated input")
	}
}

func TestScanSpriteInfosExtractsGroups(t *testing.T) {
	block1 := buildSpriteInfoBlock(32, 32, 1, 1, 100, 101)
	block2 := buildSpriteInfoBlock(64, 64, 2, 1)
	buf := append(block1, block2...)
	buf = append(buf, 0x00) // ensure loop can advance past last candidate

	groups := scanSpriteInfos(buf)
	if len(groups) != 2 {
		t.Fatalf("scanSpriteInfos returned %d groups, want 2", len(groups))
	}
	if got := groups[0].SpriteIDs; len(got) != 2 || got[0] != 100 || got[1] != 101 {
		t.Fatalf("first group IDs = %v, want [100 101]", got)
	}
	if got := groups[1].SpriteIDs; len(got) != 0 {
		t.Fatalf("second group should have zero IDs, got %v", got)
	}
}

func TestComposeGroupImageStitchesTilesHorizontally(t *testing.T) {
	dir := t.TempDir()
	ids := []int{10, 11, 12}
	colors := []color.NRGBA{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
	}

	const size = 32
	for i, id := range ids {
		writeSolidTile(t, dir, id, colors[i], size)
	}

	img, err := composeGroupImage(dir, spriteInfo{SpriteIDs: ids})
	if err != nil {
		t.Fatalf("composeGroupImage error: %v", err)
	}

	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		t.Fatalf("composeGroupImage returned %T, want *image.NRGBA", img)
	}

	bounds := nrgba.Bounds()
	if bounds.Dx() != size*len(ids) || bounds.Dy() != size {
		t.Fatalf("composed image bounds = %v, want width %d height %d", bounds, size*len(ids), size)
	}

	for i := range ids {
		px := nrgba.NRGBAAt(i*size+1, 1)
		if px != colors[i] {
			t.Fatalf("tile %d pixel = %#v, want %#v", i, px, colors[i])
		}
	}
}

func TestComposeGroupImageReturnsErrorWhenTilesMissing(t *testing.T) {
	dir := t.TempDir()

	_, err := composeGroupImage(dir, spriteInfo{SpriteIDs: []int{42}})
	if err == nil {
		t.Fatalf("composeGroupImage expected error when tiles missing")
	}
	if !strings.Contains(err.Error(), "no tiles found") {
		t.Fatalf("composeGroupImage error %q, want substring 'no tiles found'", err)
	}
}

func buildSpriteInfoBlock(w, h, layers, pw int, ids ...int) []byte {
	block := []byte{0x08}
	block = append(block, encodeVarint(w)...)
	block = append(block, 0x10)
	block = append(block, encodeVarint(h)...)
	block = append(block, 0x18)
	block = append(block, encodeVarint(layers)...)
	block = append(block, 0x20)
	block = append(block, encodeVarint(pw)...)
	for _, id := range ids {
		block = append(block, 0x28)
		block = append(block, encodeVarint(id)...)
	}
	return block
}

func encodeVarint(v int) []byte {
	var buf [10]byte
	n := binary.PutUvarint(buf[:], uint64(v))
	return buf[:n]
}

func writeSolidTile(t *testing.T, dir string, id int, c color.NRGBA, size int) {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetNRGBA(x, y, c)
		}
	}

	path := filepath.Join(dir, fmt.Sprintf("%d.png", id))
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create %s: %v", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("png.Encode %s: %v", path, err)
	}
}
