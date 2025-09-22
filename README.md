# Tibia Sprites Exporter

A batteries-included Go CLI that turns Tibia's packed client resources into clean sprite PNGs you can catalog, diff, or feed into tooling of your own.

## Table of contents

- [Overview](#overview)
- [Key capabilities](#key-capabilities)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Command reference](#command-reference)
- [Configuration & paths](#configuration--paths)
- [How it works](#how-it-works)
- [Project status](#project-status)
- [Contributing](#contributing)
- [License](#license)

## Overview

Tibia Sprites Exporter understands the catalog that ships with the official Tibia client. It walks that catalog, decompresses the binary assets, and writes them out as standard PNG files. You can then split the big sprite sheets into per-sprite tiles and regroup them based on appearance definitions extracted from the client metadata. Every command ships with progress bars, structured logging, and sensible defaults for Windows, macOS, and Linux installs of the game client.【F:src/cmd/rootCmd.go†L27-L127】【F:src/cmd/extractCmd.go†L13-L23】【F:src/cmd/splitCmd.go†L19-L38】【F:src/cmd/groupCmd.go†L20-L38】

## Key capabilities

- **One-command extraction** – Walks `catalog-content.json`, finds every sprite entry, and exports Tibia's compressed atlases as PNG files named `Sprites-<first>-<last>.png`. Failed assets are logged and the run continues.【F:src/app/catalogreader.go†L20-L68】【F:src/app/assetconverter.go†L20-L139】
- **Sprite sheet splitting** – Detects sprite ID ranges from the exported sheets and slices them into individual 32×32 or 64×64 tiles depending on sprite count, writing each sprite to `<id>.png`. Unexpected sheet sizes are logged but still processed.【F:src/app/assetconverter.go†L211-L257】【F:src/app/spritessplitter.go†L16-L74】
- **Appearance-aware grouping** – Reads the client's `appearances.dat`, reconstructs sprite groups, and stitches the individual tiles back into composite PNGs for easier review of outfits and objects.【F:src/app/spritessplitter.go†L76-L105】【F:src/app/spritesgroupper.go†L20-L217】
- **Robust progress tracking** – Fast heuristics pre-count sprite catalog entries so the CLI can render accurate progress bars during long-running export, split, and group operations.【F:src/app/catalogreader.go†L70-L83】【F:src/app/assetconverter.go†L20-L79】【F:src/app/spritessplitter.go†L16-L74】【F:src/app/spritesgroupper.go†L40-L93】
- **Human-friendly logging** – Switch between machine-readable JSON logs and console-friendly output, and enable verbose tracing for troubleshooting the asset pipeline.【F:src/cmd/rootCmd.go†L36-L86】

## Installation

### Prerequisites

- Go 1.25 or newer.【F:go.mod†L1-L10】
- Access to a Tibia client installation (to provide `catalog-content.json`, compressed sprite archives, and `appearances.dat`).

### Install with `go install`

```bash
go install github.com/simivar/tibia-sprites-exporter/src@latest
```

The command drops a `tibia-sprites-exporter` binary into your `GOBIN` (defaults to `$GOPATH/bin`).【F:src/main.go†L1-L5】

### Build from source

```bash
git clone https://github.com/simivar/tibia-sprites-exporter.git
cd tibia-sprites-exporter
go build -o tibia-sprites-exporter ./src
```

## Quick start

1. **Export sprites**
   ```bash
   tibia-sprites-exporter extract --catalog "/path/to/Tibia/assets" --output ./output/extracted
   ```
   This reads the game's `catalog-content.json`, decompresses every sprite atlas, and writes PNG sheets into `./output/extracted` (created automatically).【F:src/cmd/extractCmd.go†L13-L23】【F:src/cmd/rootCmd.go†L36-L93】【F:src/app/assetconverter.go†L20-L139】【F:src/app/assetconverter.go†L198-L208】

2. **Split the sheets**
   ```bash
   tibia-sprites-exporter split --splitOutput ./output/split
   ```
   Each exported atlas is sliced into per-sprite PNG tiles named after their sprite ID.【F:src/cmd/splitCmd.go†L13-L38】【F:src/app/spritessplitter.go†L16-L120】【F:src/app/assetconverter.go†L211-L257】

3. **Rebuild appearance composites (optional)**
   ```bash
   tibia-sprites-exporter group --splitOutput ./output/split --groupedOutput ./output/grouped
   ```
   The command locates the latest `appearances.dat`, gathers the sprite IDs used by each appearance, and renders grouped reference PNGs.【F:src/cmd/groupCmd.go†L13-L39】【F:src/app/spritessplitter.go†L76-L105】【F:src/app/spritesgroupper.go†L20-L217】

## Command reference

| Command | Purpose | Notable flags |
|---------|---------|---------------|
| `extract` | Export sprite atlases referenced by `catalog-content.json` into PNG sheets. | `--catalog`, `-c` – path to the directory containing `catalog-content.json` (defaults per OS). `--output`, `-o` – where sheets are saved.【F:src/cmd/rootCmd.go†L36-L127】【F:src/cmd/extractCmd.go†L13-L23】 |
| `split` | Split each atlas from the extract step into single-sprite PNG files. | `--splitOutput` – destination directory for per-sprite tiles.【F:src/cmd/splitCmd.go†L13-L38】【F:src/app/spritessplitter.go†L16-L120】 |
| `group` | Compose grouped sprites using appearance metadata extracted from the client. | `--splitOutput` – directory containing per-sprite PNGs. `--groupedOutput` – destination for grouped composites.【F:src/cmd/groupCmd.go†L13-L39】【F:src/app/spritesgroupper.go†L20-L217】 |

All commands understand the shared global flags:

- `--config` – load defaults from a YAML/TOML/JSON file (defaults to `~/.tse.*`).
- `--debug` – raise log level to `DEBUG`.
- `--human` – switch from structured JSON logs to pretty console logs.
- `--catalog`, `--output` – override asset and export locations.

These options are wired through Cobra/Viper, so environment variables can override the same keys if you follow Viper's naming conventions.【F:src/cmd/rootCmd.go†L36-L99】

## Configuration & paths

- **Catalog discovery** – The CLI expands `~` in paths and, by default, points to the typical Tibia asset directories on macOS, Windows, and Linux. The tool verifies that `catalog-content.json` exists before running any work.【F:src/app/utils.go†L9-L23】【F:src/cmd/rootCmd.go†L42-L121】
- **Outputs** – Default extraction, split, and grouped outputs live under `./output/extracted`, `./output/split`, and `./output/grouped`. Directories are created on demand.【F:src/cmd/rootCmd.go†L42-L127】【F:src/cmd/splitCmd.go†L13-L38】【F:src/cmd/groupCmd.go†L13-L39】【F:src/app/assetconverter.go†L198-L208】
- **Configuration file** – Place a `.tse.yaml` (or `.tse.json`/`.tse.toml`) file in your home directory to persist defaults for the global flags. You can point to an alternate config with `--config` if desired.【F:src/cmd/rootCmd.go†L36-L71】

## How it works

1. **Streaming catalog reader** – A buffered JSON decoder streams `catalog-content.json` so large files never load fully into memory. Each catalog element is emitted for further processing.【F:src/app/catalogreader.go†L20-L68】
2. **Sprite decompression pipeline** – For each sprite entry the tool opens the compressed asset, skips the proprietary CIP header, reconstructs a proper LZMA header, decodes the BMP payload, and encodes a PNG file on disk.【F:src/app/assetconverter.go†L82-L208】
3. **Sheet splitting** – Sprite sheets are treated as grids of 32×32 tiles (switching to 64×64 for small sets) and the exporter keeps writing tiles until the declared sprite count is satisfied.【F:src/app/assetconverter.go†L211-L257】
4. **Appearance grouping** – The binary `appearances.dat` is scanned for sprite references, pulling out sprite ID sequences that represent outfits or objects. Missing sprite tiles are skipped but logged so you can inspect the source assets.【F:src/app/spritesgroupper.go†L20-L217】
5. **Resilient logging** – Missing files or decode errors are surfaced via zerolog while the CLI keeps marching forward unless a fatal precondition (like a missing catalog) is encountered.【F:src/cmd/rootCmd.go†L88-L93】【F:src/app/assetconverter.go†L20-L139】【F:src/app/spritessplitter.go†L16-L74】

## Project status

The exporter is stable for day-to-day sprite extraction. Future work includes automated regression tests and packaging improvements, but the core sprite pipeline is production-ready and already powers community tooling.【F:src/app/assetconverter.go†L20-L257】【F:src/app/spritessplitter.go†L16-L120】【F:src/app/spritesgroupper.go†L20-L217】

## Contributing

Issues and pull requests are welcome. Please format Go code with `gofmt` and include tests or sample assets when proposing changes to the decoding pipeline so regressions can be reproduced locally.【F:src/main.go†L1-L5】

## License

Tibia Sprites Exporter is distributed under the terms of the MIT License. Tibia and all related assets remain © CipSoft GmbH.
