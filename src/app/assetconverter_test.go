package app

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
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
