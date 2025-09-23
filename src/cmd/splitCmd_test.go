package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/viper"
)

func TestSplitCommandLogsExpandedPaths(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)
	buf := captureLogs(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	extractedRel := "~/extracted"
	splitRel := "~/split"

	extractedDir := filepath.Join(homeDir, "extracted")
	splitDir := filepath.Join(homeDir, "split")

	if err := ensureDir(extractedDir); err != nil {
		t.Fatalf("ensureDir extracted: %v", err)
	}
	if err := ensureDir(splitDir); err != nil {
		t.Fatalf("ensureDir split: %v", err)
	}

	viper.Set("output", extractedRel)
	viper.Set("splitOutput", splitRel)

	splitCmd.Run(splitCmd, nil)

	logs := buf.String()
	wantOutput := app.ExpandPath(extractedRel)
	if !strings.Contains(logs, "\"output\":\""+wantOutput+"\"") {
		t.Fatalf("expected log to contain expanded output %q, got %q", wantOutput, logs)
	}
	wantSplit := app.ExpandPath(splitRel)
	if !strings.Contains(logs, "\"splitOutput\":\""+wantSplit+"\"") {
		t.Fatalf("expected log to contain expanded split output %q, got %q", wantSplit, logs)
	}
	if !strings.Contains(logs, "Tibia Sprites Split finished") {
		t.Fatalf("expected finish log, got %q", logs)
	}
}

func TestSplitOutputFlagUpdatesGlobalAndViper(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	override := "~/custom-split"

	if err := viper.BindPFlag("splitOutput", splitCmd.Flags().Lookup("splitOutput")); err != nil {
		t.Fatalf("bind splitOutput flag: %v", err)
	}

	if err := splitCmd.Flags().Set("splitOutput", override); err != nil {
		t.Fatalf("set splitOutput flag: %v", err)
	}
	t.Cleanup(func() {
		_ = splitCmd.Flags().Set("splitOutput", defaultSplitOutputPath())
	})

	if SplitOutputPath != override {
		t.Fatalf("SplitOutputPath = %q, want %q", SplitOutputPath, override)
	}
	if got := viper.GetString("splitOutput"); got != override {
		t.Fatalf("viper splitOutput = %q, want %q", got, override)
	}
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
