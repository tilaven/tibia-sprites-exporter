package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/viper"
)

func TestExtractCommandUpdatesGlobalsFromExpandedPaths(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)
	buf := captureLogs(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	catalogRel := "~/catalog"
	outputRel := "~/out"

	catalogDir := filepath.Join(homeDir, "catalog")
	outputDir := filepath.Join(homeDir, "out")

	if err := ensureDir(catalogDir); err != nil {
		t.Fatalf("ensureDir catalog: %v", err)
	}
	if err := ensureDir(outputDir); err != nil {
		t.Fatalf("ensureDir output: %v", err)
	}
	catalogFile := filepath.Join(catalogDir, "catalog-content.json")
	if err := os.WriteFile(catalogFile, []byte("[]"), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	viper.Set("catalog", catalogRel)
	viper.Set("output", outputRel)

	extractCmd.Run(extractCmd, nil)

	wantCatalog := app.ExpandPath(catalogRel)
	if CatalogContentJsonPath != wantCatalog {
		t.Fatalf("CatalogContentJsonPath = %q, want %q", CatalogContentJsonPath, wantCatalog)
	}
	wantFile := filepath.Join(wantCatalog, "catalog-content.json")
	if CatalogContentJsonPathWithFilename != wantFile {
		t.Fatalf("CatalogContentJsonPathWithFilename = %q, want %q", CatalogContentJsonPathWithFilename, wantFile)
	}
	wantOutput := app.ExpandPath(outputRel)
	if OutputPath != wantOutput {
		t.Fatalf("OutputPath = %q, want %q", OutputPath, wantOutput)
	}

	logs := buf.String()
	if !strings.Contains(logs, "Tibia Sprites extract running") {
		t.Fatalf("expected start log, got %q", logs)
	}
	if !strings.Contains(logs, "Tibia Sprites extract finished") {
		t.Fatalf("expected finish log, got %q", logs)
	}
}
