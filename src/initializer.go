package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	CatalogContentJsonPath     string
	CatalogContentJsonFullPath string
	OutputPath                 string
	SplitSprites               bool
	flagJsonPath               *string
	flagOutputDir              *string
	flagHumanOutput            *bool
	flagDebugMode              *bool
	flagSplitSprites           *bool
)

func initExporter() {
	initFlags()
	initDebugMode()
	initHumanOutput()

	log.Info().Msg("Tibia Sprites Exporter initializing")

	initCatalogContentPath()
	validateCatalogContentPath()
	initOutputDir()
	validateOutputPath()
	initSplitOption()

	log.Info().Msg("Initialized")
	log.Debug().Msgf("catalog content path: %s", CatalogContentJsonPath)
	log.Debug().Msgf("output path: %s", OutputPath)
	log.Debug().Msgf("split sprites: %v", SplitSprites)
}

func initFlags() {
	// Custom help/usage with ASCII art
	flag.Usage = func() {
		ascii := `
 ______________       ___   ___
/_  __/ __/ __/ _  __/ _ \ <  /
 / / / _/_\ \  | |/ / // / / / 
/_/ /___/___/  |___/\___(_)_/  `
		fmt.Fprintln(os.Stderr, ascii)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Tibia Sprites Exporter - extract Tibia client sprite sheets into PNG files")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  tibia-sprites-exporter [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -jsonPath string   Path to the catalog-content.json file OR its containing directory")
		fmt.Fprintln(os.Stderr, "  -output string     Output directory (defaults to <executable_dir>/output)")
		fmt.Fprintln(os.Stderr, "  -human             Pretty-print logs for humans")
		fmt.Fprintln(os.Stderr, "  -debug             Enable debug logs")
		fmt.Fprintln(os.Stderr, "  -split             Split each 384x384 sheet into per-sprite PNGs (32x32 or 64x64)")
		fmt.Fprintln(os.Stderr, "  -run               Run the exporter (by default we dry-run)")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Environment variables:")
		fmt.Fprintln(os.Stderr, "  TES_JSON_PATH      Same as -jsonPath")
		fmt.Fprintln(os.Stderr, "  TES_OUTPUT_DIR     Same as -output")
		fmt.Fprintln(os.Stderr, "  TES_SPLIT or TES_SPLIT_SPRITES  Enable sprite splitting like -split")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  tibia-sprites-exporter -human")
		fmt.Fprintln(os.Stderr, "  tibia-sprites-exporter -jsonPath \"/path/to/Tibia/assets\" -output \"/tmp/exports\"")
		fmt.Fprintln(os.Stderr, "  tibia-sprites-exporter -split -human")
	}

	flag.Bool("run", false, "Run the exporter (by default we dry-run)")
	flagJsonPath = flag.String("jsonPath", "", "Path to catalog-content.json file")
	flagOutputDir = flag.String("output", "", "Where to output exported sprite files (defaults to pwd + output)")
	flagHumanOutput = flag.Bool("human", false, "Whether pretty print the logs")
	flagDebugMode = flag.Bool("debug", false, "Whether enable debug logs")
	flagSplitSprites = flag.Bool("split", false, "Split each 384x384 sheet into individual sprite PNGs named by sprite ID")

	// If no arguments are provided, show help and exit instead of running straight away
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	flag.Parse()
}

func initHumanOutput() {
	if flagHumanOutput != nil && *flagHumanOutput {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

func initDebugMode() {
	if *flagDebugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func validateCatalogContentPath() {
	path := CatalogContentJsonPath

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Err(err).Msgf("[error] path does not exist: %s", path)
		}
		log.Err(err).Msgf("[error] failed to stat path %s: %v", path, err)
	}

	if info.IsDir() {
		// If it's a directory, append the file name
		path = filepath.Join(path, "catalog-content.json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Err(err).Msgf("catalog-content.json not found in directory: %s", CatalogContentJsonPath)
		}

		CatalogContentJsonFullPath = path
	}

	log.Debug().Msgf("catalog content path: %s", CatalogContentJsonFullPath)
}

func initCatalogContentPath() {
	if isEnvExist("TES_JSON_PATH") {
		CatalogContentJsonPath = sanitizeCatalogContentPath(os.Getenv("TES_JSON_PATH"))
		return
	}

	if flagJsonPath != nil && *flagJsonPath != "" {
		CatalogContentJsonPath = sanitizeCatalogContentPath(*flagJsonPath)
		return
	}

	// todo add switch based on the OS
	CatalogContentJsonPath = expandPath(
		"~/Library/Application Support/CipSoft GmbH/Tibia/packages/Tibia.app/Contents/Resources/assets",
	)
}

func validateOutputPath() {
	if _, err := os.Stat(OutputPath); os.IsNotExist(err) {
		if err := os.MkdirAll(OutputPath, 0o755); err != nil {
			log.Err(err).Msgf("failed to create output path %s: %v", OutputPath, err)
		}
		log.Info().Msgf("created missing output path: %s", OutputPath)
	}
}

func expandPath(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func initOutputDir() {
	if isEnvExist("TES_OUTPUT_DIR") {
		OutputPath = os.Getenv("TES_OUTPUT_DIR")
		return
	}

	if flagOutputDir != nil && *flagOutputDir != "" {
		OutputPath = *flagJsonPath
		return
	}

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	osSeparator := string(os.PathSeparator)
	OutputPath = fmt.Sprint(filepath.Dir(ex), osSeparator, "output")
}

func sanitizeCatalogContentPath(path string) string {
	path = filepath.Clean(path)
	if strings.HasSuffix(path, "catalog-content.json") {
		return filepath.Dir(path)
	}
	return path
}

func initSplitOption() {
	// Environment variables take precedence if present
	if isEnvExist("TES_SPLIT") || isEnvExist("TES_SPLIT_SPRITES") {
		SplitSprites = true
		return
	}
	// Fallback to CLI flag
	if flagSplitSprites != nil && *flagSplitSprites {
		SplitSprites = true
	}
}
