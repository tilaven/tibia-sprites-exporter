package app

import (
	"os"
	"path/filepath"
	"testing"
)

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
