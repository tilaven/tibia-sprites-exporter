# Tibia Sprites Exporter

```asciiart
 ______________       ___   ___
/_  __/ __/ __/ _  __/ _ \ <  /
 / / / _/_\ \  | |/ / // / / / 
/_/ /___/___/  |___/\___(_)_/  
```

A small, fast, and cross-platform CLI utility to extract Tibia client sprite sheets into PNG files and optionally split 
them into per-sprite PNGs named by their sprite ID.

# What this tool does:
- Reads Tibia's catalog-content.json (the asset catalog used by the client).
- For each entry of type "sprite", opens the referenced compressed asset file from the same assets directory.
- Strips the CIP header, repairs the LZMA "alone" header, and decompresses to a BMP.
- Converts the BMP to PNG and writes it as Sprites-<firstID>-<lastID>.png to the output directory.
- Optional: splits each sheet into individual 32x32 or 64x64 tiles written to output/split/<spriteId>.png.

# Status and compatibility note
- Tested with the newest version of sprites as of 18.09 2 am CET.
- Automated verification in CI pipelines will come.

# Getting started
- Prebuilt binaries: When a tag starting with v is pushed, GitHub Actions builds artifacts for Windows, Linux, and macOS (see Releases/Actions artifacts in this repository).
- Build from source:
  1) Prerequisites: Go (see go.mod for the required version) and a C toolchain is NOT required (CGO is disabled).
  2) Clone the repo and build:
     `go build -trimpath -ldflags "-s -w" -o tibia-sprites-exporter ./src`

# Usage
Basic example (pretty logs):
```shell
tibia-sprites-exporter -human
```

Point to a specific assets folder (the one containing catalog-content.json):
```shell
tibia-sprites-exporter -jsonPath "/path/to/Tibia/assets"
```

Choose a custom output directory:
```shell
tibia-sprites-exporter -output "/tmp/exports"
```

Split each sheet into per-sprite PNGs named by ID:
```shell
tibia-sprites-exporter -split
```

Enable debug logs:
```shell
tibia-sprites-exporter -debug -human
```

Flags
- `-jsonPath` string    Path to the catalog-content.json file OR its containing directory.
- `-output` string      Output directory (defaults to <executable_dir>/output).
- `-human`              Pretty-print logs for humans.
- `-debug`              Enable debug logs.
- `-split`              Split each 384x384 sheet into per-sprite PNGs (32x32 or 64x64 tiles depending on sheet content).

Environment variables (override flags)
- `TES_JSON_PATH`       Same as -jsonPath.
- `TES_OUTPUT_DIR`      Same as -output.
- `TES_SPLIT` or `TES_SPLIT_SPRITES`

Defaults
- If no `-jsonPath` (or `TES_JSON_PATH`) is provided the tool will look for catalog-content.json in the default path
- Output defaults to a folder named output next to the executable

How it works under the hood
- Streaming JSON parser reads `catalog-content.json` for entries with `"type": "sprite"`.
- Each referenced file is read from the assets directory, CIP header is skipped, and an LZMA reader is constructed with a corrected header.
- The decompressed BMP is converted to PNG via golang.org/x/image/bmp and written to disk.
- If -split (or TES_SPLIT/TES_SPLIT_SPRITES) is enabled, the 384x384 sheet is sliced row-major into 32x32 tiles (or 64x64 for small sets) and saved as output/split/<spriteId>.png.

Notes and tips
- If you pass a directory to -jsonPath (recommended), the tool will automatically look for catalog-content.json inside it.
- If a referenced asset file is missing, the tool logs it at debug level and continues.
- On first run, the output directory is created automatically if it doesn't exist.

Roadmap
- CI pipeline verification of outputs against known-good references.
- Cross-platform discovery of default Tibia assets locations (Windows/Linux).

Acknowledgments
- Tibia is a trademark of CipSoft GmbH. This tool is a community utility and is not affiliated with or endorsed by CipSoft.
