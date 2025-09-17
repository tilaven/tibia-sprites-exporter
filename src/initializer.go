package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	CatalogContentJsonPath     string
	CatalogContentJsonFullPath string
	OutputPath                 string
	flagJsonPath               *string
	flagOutputDir              *string
)

func initExporter() {
	initFlags()
	initCatalogContentPath()
	validateCatalogContentPath()
	initOutputDir()
	validateOutputPath()

	log.Printf("[info] catalog content path: %s", CatalogContentJsonPath)
	log.Printf("[info] output path: %s", OutputPath)
}

func initFlags() {
	flagJsonPath = flag.String("jsonPath", "", "Path to catalog-content.json file")
	flagOutputDir = flag.String("output", "", "Where to output exported sprite files (defaults to pwd + output)")

	flag.Parse()
}

func validateCatalogContentPath() {
	path := CatalogContentJsonPath

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("[error] path does not exist: %s", path)
		}
		log.Fatalf("[error] failed to stat path %s: %v", path, err)
	}

	if info.IsDir() {
		// If it's a directory, append the file name
		path = filepath.Join(path, "catalog-content.json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatalf("[error] catalog-content.json not found in directory: %s", CatalogContentJsonPath)
		}

		CatalogContentJsonFullPath = path
	}
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
			log.Fatalf("[error] failed to create output path %s: %v", OutputPath, err)
		}
		log.Printf("[info] created missing output path: %s", OutputPath)
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
