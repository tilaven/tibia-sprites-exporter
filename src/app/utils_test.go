package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPathExpandsHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	got := ExpandPath("~/catalog")
	want := filepath.Join(home, "catalog")
	if got != want {
		t.Fatalf("ExpandPath = %q, want %q", got, want)
	}
}

func TestExpandPathLeavesAbsolutePathUnchanged(t *testing.T) {
	path := "/tmp/catalog"
	if got := ExpandPath(path); got != path {
		t.Fatalf("ExpandPath(%q) = %q, want same", path, got)
	}
}

func TestSanitizeCatalogContentPathRemovesFilename(t *testing.T) {
	path := filepath.Join("/tmp", "dir", "catalog-content.json")
	got := sanitizeCatalogContentPath(path)
	want := filepath.Join("/tmp", "dir")
	if got != want {
		t.Fatalf("sanitizeCatalogContentPath(%q) = %q, want %q", path, got, want)
	}
}

func TestSanitizeCatalogContentPathLeavesDirectory(t *testing.T) {
	path := filepath.Join("/tmp", "dir")
	if got := sanitizeCatalogContentPath(path); got != path {
		t.Fatalf("sanitizeCatalogContentPath(%q) = %q, want same", path, got)
	}
}
