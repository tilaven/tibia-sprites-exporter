package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/viper"
)

func preserveGlobals(t *testing.T) {
	t.Helper()
	origCatalog := CatalogContentJsonPath
	origCatalogWithFile := CatalogContentJsonPathWithFilename
	origOutput := OutputPath
	origCfgFile := cfgFile
	origDebug := debugMode
	origHuman := humanReadableLogs
	origSplit := SplitOutputPath
	origGrouped := GroupedOutputPath
	origLogger := log.Logger
	origLevel := zerolog.GlobalLevel()

	t.Cleanup(func() {
		CatalogContentJsonPath = origCatalog
		CatalogContentJsonPathWithFilename = origCatalogWithFile
		OutputPath = origOutput
		cfgFile = origCfgFile
		debugMode = origDebug
		humanReadableLogs = origHuman
		SplitOutputPath = origSplit
		GroupedOutputPath = origGrouped
		log.Logger = origLogger
		zerolog.SetGlobalLevel(origLevel)
	})
}

func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	log.Logger = zerolog.New(buf).With().Timestamp().Logger()
	return buf
}

func TestDefaultCatalogContentPathMatchesRuntime(t *testing.T) {
	switch runtime.GOOS {
	case "darwin":
		want := app.ExpandPath("~/Library/Application Support/CipSoft GmbH/Tibia/packages/Tibia.app/Contents/Resources/assets")
		if got := defaultCatalogContentPath(); got != want {
			t.Fatalf("defaultCatalogContentPath() = %q, want %q", got, want)
		}
	case "windows":
		want := app.ExpandPath("~/AppData/Local/Tibia/packages/Tibia/assets")
		if got := defaultCatalogContentPath(); got != want {
			t.Fatalf("defaultCatalogContentPath() = %q, want %q", got, want)
		}
	case "linux":
		want := app.ExpandPath("~/.local/share/CipSoft GmbH/Tibia/packages/Tibia/assets")
		if got := defaultCatalogContentPath(); got != want {
			t.Fatalf("defaultCatalogContentPath() = %q, want %q", got, want)
		}
	default:
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("defaultCatalogContentPath() did not panic on unsupported runtime %q", runtime.GOOS)
			}
		}()
		_ = defaultCatalogContentPath()
	}
}

func TestDefaultOutputPath(t *testing.T) {
	if got, want := defaultOutputPath(), app.ExpandPath("./output/extracted"); got != want {
		t.Fatalf("defaultOutputPath() = %q, want %q", got, want)
	}
}

func TestDefaultSplitAndGroupedPaths(t *testing.T) {
	if got, want := defaultSplitOutputPath(), app.ExpandPath("./output/split"); got != want {
		t.Fatalf("defaultSplitOutputPath() = %q, want %q", got, want)
	}
	if got, want := defaultGroupedOutputPath(), app.ExpandPath("./output/grouped"); got != want {
		t.Fatalf("defaultGroupedOutputPath() = %q, want %q", got, want)
	}
}

func TestInitPathsFromViperOverridesGlobals(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)

	CatalogContentJsonPath = "before-catalog"
	OutputPath = "before-output"

	tempDir := t.TempDir()
	cat := filepath.Join(tempDir, "catalog")
	out := filepath.Join(tempDir, "output")

	viper.Set("catalog", cat)
	viper.Set("output", out)

	initPathsFromViper()

	if CatalogContentJsonPath != app.ExpandPath(cat) {
		t.Fatalf("CatalogContentJsonPath = %q, want %q", CatalogContentJsonPath, app.ExpandPath(cat))
	}
	if OutputPath != app.ExpandPath(out) {
		t.Fatalf("OutputPath = %q, want %q", OutputPath, app.ExpandPath(out))
	}
}

func TestInitPathsFromViperKeepsExistingWhenUnset(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)

	CatalogContentJsonPath = "keep-catalog"
	OutputPath = "keep-output"

	initPathsFromViper()

	if CatalogContentJsonPath != "keep-catalog" {
		t.Fatalf("CatalogContentJsonPath changed to %q", CatalogContentJsonPath)
	}
	if OutputPath != "keep-output" {
		t.Fatalf("OutputPath changed to %q", OutputPath)
	}
}

func TestInitCatalogContentJsonPathWithFilename(t *testing.T) {
	preserveGlobals(t)

	dir := t.TempDir()
	file := filepath.Join(dir, "catalog-content.json")
	if err := os.WriteFile(file, []byte("[]"), 0o644); err != nil {
		t.Fatalf("failed writing catalog-content.json: %v", err)
	}

	CatalogContentJsonPath = dir
	CatalogContentJsonPathWithFilename = ""

	initCatalogContentJsonPathWithFilename()

	if CatalogContentJsonPathWithFilename != file {
		t.Fatalf("CatalogContentJsonPathWithFilename = %q, want %q", CatalogContentJsonPathWithFilename, file)
	}
}

func TestInitDebugModeRespectsViperAndFlag(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)

	viper.Set("debug", false)
	debugMode = false
	initDebugMode()
	if lvl := zerolog.GlobalLevel(); lvl != zerolog.InfoLevel {
		t.Fatalf("Global level = %v, want %v", lvl, zerolog.InfoLevel)
	}

	viper.Set("debug", true)
	initDebugMode()
	if lvl := zerolog.GlobalLevel(); lvl != zerolog.DebugLevel {
		t.Fatalf("Global level = %v, want %v when viper debug true", lvl, zerolog.DebugLevel)
	}

	viper.Set("debug", false)
	debugMode = true
	initDebugMode()
	if lvl := zerolog.GlobalLevel(); lvl != zerolog.DebugLevel {
		t.Fatalf("Global level = %v, want %v when flag debug true", lvl, zerolog.DebugLevel)
	}
}

func TestInitHumanOutputSwitchesLogger(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	origStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = origStderr })

	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	log.Info().Msg("json before")

	humanReadableLogs = true
	initHumanOutput()

	log.Info().Msg("human after")

	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read logs: %v", err)
	}
	logs := string(data)
	if !strings.Contains(logs, "\"message\":\"json before\"") {
		t.Fatalf("expected JSON log before switch, got %q", logs)
	}
	if !strings.Contains(logs, "human after") {
		t.Fatalf("expected human log after switch, got %q", logs)
	}
	if strings.Contains(logs, "\"message\":\"human after\"") {
		t.Fatalf("expected console output for human log, got %q", logs)
	}
}

func TestSplitCommandRun(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)
	buf := captureLogs(t)

	tempDir := t.TempDir()
	extractDir := filepath.Join(tempDir, "extract")
	splitDir := filepath.Join(tempDir, "split")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatalf("mkdir extractDir: %v", err)
	}

	viper.Set("output", extractDir)
	viper.Set("splitOutput", splitDir)

	splitCmd.Run(splitCmd, nil)

	logs := buf.String()
	if !strings.Contains(logs, "Tibia Sprites Split running") {
		t.Fatalf("expected start log, got %q", logs)
	}
	if !strings.Contains(logs, "Tibia Sprites Split finished") {
		t.Fatalf("expected finish log, got %q", logs)
	}
}

func TestExtractCommandRun(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)
	buf := captureLogs(t)

	tempDir := t.TempDir()
	catalogDir := filepath.Join(tempDir, "catalog")
	outputDir := filepath.Join(tempDir, "out")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatalf("mkdir catalogDir: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir outputDir: %v", err)
	}
	catalogFile := filepath.Join(catalogDir, "catalog-content.json")
	if err := os.WriteFile(catalogFile, []byte("[]"), 0o644); err != nil {
		t.Fatalf("write catalog-content.json: %v", err)
	}

	viper.Set("catalog", catalogDir)
	viper.Set("output", outputDir)

	extractCmd.Run(extractCmd, nil)

	if CatalogContentJsonPath != app.ExpandPath(catalogDir) {
		t.Fatalf("CatalogContentJsonPath = %q, want %q", CatalogContentJsonPath, app.ExpandPath(catalogDir))
	}
	wantFile := filepath.Join(app.ExpandPath(catalogDir), "catalog-content.json")
	if CatalogContentJsonPathWithFilename != wantFile {
		t.Fatalf("CatalogContentJsonPathWithFilename = %q, want %q", CatalogContentJsonPathWithFilename, wantFile)
	}
	if OutputPath != app.ExpandPath(outputDir) {
		t.Fatalf("OutputPath = %q, want %q", OutputPath, app.ExpandPath(outputDir))
	}

	logs := buf.String()
	if !strings.Contains(logs, "Tibia Sprites extract running") {
		t.Fatalf("expected start log, got %q", logs)
	}
	if !strings.Contains(logs, "Tibia Sprites extract finished") {
		t.Fatalf("expected finish log, got %q", logs)
	}
}

func TestGroupCommandRun(t *testing.T) {
	preserveGlobals(t)
	resetViper(t)
	buf := captureLogs(t)

	tempDir := t.TempDir()
	catalogDir := filepath.Join(tempDir, "catalog")
	splitDir := filepath.Join(tempDir, "split")
	groupedDir := filepath.Join(tempDir, "grouped")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatalf("mkdir catalogDir: %v", err)
	}
	if err := os.MkdirAll(splitDir, 0o755); err != nil {
		t.Fatalf("mkdir splitDir: %v", err)
	}
	catalogFile := filepath.Join(catalogDir, "catalog-content.json")
	if err := os.WriteFile(catalogFile, []byte(`[{"type":"appearances","file":"appearances.dat"}]`), 0o644); err != nil {
		t.Fatalf("write catalog-content.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "appearances.dat"), nil, 0o644); err != nil {
		t.Fatalf("write appearances.dat: %v", err)
	}

	viper.Set("catalog", catalogDir)
	viper.Set("splitOutput", splitDir)
	viper.Set("groupedOutput", groupedDir)

	groupCmd.Run(groupCmd, nil)

	if _, err := os.Stat(groupedDir); err != nil {
		t.Fatalf("expected grouped output directory: %v", err)
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
