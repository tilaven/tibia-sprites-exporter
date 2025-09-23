package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/viper"
)

func TestGroupCommandExpandsPathsAndLogs(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)
	buf := captureLogs(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	catalogRel := "~/catalog"
	splitRel := "~/split"
	groupedRel := "~/grouped"

	catalogDir := filepath.Join(homeDir, "catalog")
	splitDir := filepath.Join(homeDir, "split")

	if err := ensureDir(catalogDir); err != nil {
		t.Fatalf("ensureDir catalog: %v", err)
	}
	if err := ensureDir(splitDir); err != nil {
		t.Fatalf("ensureDir split: %v", err)
	}

	catalogContent := []byte(`[{"type":"appearances","file":"appearances.dat"}]`)
	if err := os.WriteFile(filepath.Join(catalogDir, "catalog-content.json"), catalogContent, 0o644); err != nil {
		t.Fatalf("write catalog-content.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "appearances.dat"), nil, 0o644); err != nil {
		t.Fatalf("write appearances.dat: %v", err)
	}

	viper.Set("catalog", catalogRel)
	viper.Set("splitOutput", splitRel)
	viper.Set("groupedOutput", groupedRel)

	groupCmd.Run(groupCmd, nil)

	groupedDir := app.ExpandPath(groupedRel)
	info, err := os.Stat(groupedDir)
	if err != nil {
		t.Fatalf("stat grouped output: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("grouped output is not a directory: %v", info.Mode())
	}

	logs := buf.String()
	if !strings.Contains(logs, "Tibia Sprites group running") {
		t.Fatalf("expected start log, got %q", logs)
	}
	if !strings.Contains(logs, "Appearances file name: appearances.dat") {
		t.Fatalf("expected appearances log, got %q", logs)
	}
	if !strings.Contains(logs, "Tibia Sprites group finished") {
		t.Fatalf("expected finish log, got %q", logs)
	}
}

func TestGroupedOutputFlagUpdatesGlobalAndViper(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	override := "~/custom-grouped"

	if err := viper.BindPFlag("groupedOutput", groupCmd.Flags().Lookup("groupedOutput")); err != nil {
		t.Fatalf("bind groupedOutput flag: %v", err)
	}

	if err := groupCmd.Flags().Set("groupedOutput", override); err != nil {
		t.Fatalf("set groupedOutput flag: %v", err)
	}
	t.Cleanup(func() {
		_ = groupCmd.Flags().Set("groupedOutput", defaultGroupedOutputPath())
	})

	if GroupedOutputPath != override {
		t.Fatalf("GroupedOutputPath = %q, want %q", GroupedOutputPath, override)
	}
	if got := viper.GetString("groupedOutput"); got != override {
		t.Fatalf("viper groupedOutput = %q, want %q", got, override)
	}
}

func TestGroupSplitOutputFlagUpdatesGlobalAndViper(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	override := "~/custom-split"

	if err := viper.BindPFlag("splitOutput", groupCmd.Flags().Lookup("splitOutput")); err != nil {
		t.Fatalf("bind splitOutput flag: %v", err)
	}

	if err := groupCmd.Flags().Set("splitOutput", override); err != nil {
		t.Fatalf("set splitOutput flag: %v", err)
	}
	t.Cleanup(func() {
		_ = groupCmd.Flags().Set("splitOutput", defaultSplitOutputPath())
	})

	if SplitOutputPath != override {
		t.Fatalf("SplitOutputPath = %q, want %q", SplitOutputPath, override)
	}
	if got := viper.GetString("splitOutput"); got != override {
		t.Fatalf("viper splitOutput = %q, want %q", got, override)
	}
}
