package app

import (
	"os"
	"path/filepath"
	"strings"
)

func ExpandPath(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func sanitizeCatalogContentPath(path string) string {
	path = filepath.Clean(path)
	if strings.HasSuffix(path, "catalog-content.json") {
		return filepath.Dir(path)
	}
	return path
}
