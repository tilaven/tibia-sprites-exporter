package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func captureLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()

	buf := &bytes.Buffer{}
	origLogger := log.Logger
	log.Logger = zerolog.New(buf).With().Timestamp().Logger()

	return buf, func() {
		log.Logger = origLogger
	}
}

func writeTestPNG(t *testing.T, path string, img image.Image) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", path, err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create %s: %v", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Encode PNG %s: %v", path, err)
	}
}

func TestGetTotalToSplitCountsOnlySpritePNGs(t *testing.T) {
	dir := t.TempDir()

	names := []string{
		"Sprites-1-2.png",
		"Sprites-10-11.png",
		"ignore.txt",
	}

	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	if got, want := getTotalToSplit(entries), 2; got != want {
		t.Fatalf("getTotalToSplit = %d, want %d", got, want)
	}
}

func TestGetAppearancesFileNameFromCatalogContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.json")
	contents := `[
                {"type":"sprite","file":"sprites-1.png"},
                {"type":"appearances","file":"appearances.dat"},
                {"type":"effect","file":"effects.dat"}
        ]`

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got := GetAppearancesFileNameFromCatalogContent(path)
	if got != "appearances.dat" {
		t.Fatalf("GetAppearancesFileNameFromCatalogContent = %q, want %q", got, "appearances.dat")
	}
}

func TestGetAppearancesFileNameFromCatalogContentPanicsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.json")
	contents := `[
                {"type":"sprite","file":"sprites-1.png"}
        ]`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when appearances entry missing")
		}
	}()

	_ = GetAppearancesFileNameFromCatalogContent(path)
}

func TestSplitSpritesLogsErrorWhenDirectoryMissing(t *testing.T) {
	buf, restore := captureLogs(t)
	defer restore()

	extracted := filepath.Join(t.TempDir(), "missing")
	SplitSprites(extracted, t.TempDir())

	out := buf.String()
	if !strings.Contains(out, "Failed to read directory. Did you run the extract command?") {
		t.Fatalf("log output %q missing read-dir error", out)
	}
	if !strings.Contains(out, fmt.Sprintf("\"extractedDir\":\"%s\"", extracted)) {
		t.Fatalf("log output %q missing extractedDir", out)
	}
}

func TestSplitSpritesWarnsWhenNoSprites(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	buf, restore := captureLogs(t)
	defer restore()

	SplitSprites(dir, t.TempDir())

	out := buf.String()
	if !strings.Contains(out, "No sprites found to split. Did you run the extract command?") {
		t.Fatalf("log output %q missing no-sprites warning", out)
	}
}

func TestSplitSpritesProcessesSpriteSheets(t *testing.T) {
	extracted := t.TempDir()
	split := t.TempDir()

	img := image.NewRGBA(image.Rect(0, 0, 128, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 128; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 0xFF, A: 0xFF})
		}
	}

	writeTestPNG(t, filepath.Join(extracted, "Sprites-100-101.png"), img)
	if err := os.Mkdir(filepath.Join(extracted, "sub"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	_, restore := captureLogs(t)
	defer restore()

	SplitSprites(extracted, split)

	for id := 100; id <= 101; id++ {
		path := filepath.Join(split, fmt.Sprintf("%d.png", id))
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("Open %s: %v", path, err)
		}
		decoded, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatalf("Decode %s: %v", path, err)
		}
		bounds := decoded.Bounds()
		if bounds.Dx() != 64 || bounds.Dy() != 64 {
			t.Fatalf("sprite %d bounds = %v, want 64x64", id, bounds)
		}
	}
}

func TestSplitSpritesLogsDecodeErrors(t *testing.T) {
	extracted := t.TempDir()
	split := t.TempDir()

	if err := os.WriteFile(filepath.Join(extracted, "Sprites-200-201.png"), []byte("nope"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	buf, restore := captureLogs(t)
	defer restore()

	SplitSprites(extracted, split)

	out := buf.String()
	if !strings.Contains(out, "failed to decode PNG") {
		t.Fatalf("log output %q missing decode error", out)
	}
}

func TestSplitSpritesLogsInvalidNumericPart(t *testing.T) {
	extracted := t.TempDir()
	split := t.TempDir()

	name := fmt.Sprintf("Sprites-%s-1.png", strings.Repeat("9", 40))
	if err := os.WriteFile(filepath.Join(extracted, name), []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	buf, restore := captureLogs(t)
	defer restore()

	SplitSprites(extracted, split)

	out := buf.String()
	if !strings.Contains(out, "invalid numeric part in filename") {
		t.Fatalf("log output %q missing invalid numeric warning", out)
	}
}

func TestSplitSpritesLogsOpenErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions unreliable on Windows")
	}

	extracted := t.TempDir()
	split := t.TempDir()

	missing := filepath.Join(extracted, "missing.png")
	link := filepath.Join(extracted, "Sprites-5-6.png")
	if err := os.Symlink(missing, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	buf, restore := captureLogs(t)
	defer restore()

	SplitSprites(extracted, split)

	out := buf.String()
	if !strings.Contains(out, "failed to open") {
		t.Fatalf("log output %q missing open error", out)
	}
}
