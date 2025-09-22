package app

import (
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/rs/zerolog/log"
	bar "github.com/schollz/progressbar/v3"
)

var spriteFilePattern = regexp.MustCompile(`^Sprites-(\d+)-(\d+)\.png$`)

func SplitSprites(extractedDir, splitOutputDir string) {
	entries, err := os.ReadDir(extractedDir)
	if err != nil {
		log.Err(err).
			Str("extractedDir", extractedDir).
			Msg("Failed to read directory. Did you run the extract command?")
		return
	}

	total := getTotalToSplit(entries)
	if total == 0 {
		log.Warn().
			Str("extractedDir", extractedDir).
			Msg("No sprites found to split. Did you run the extract command?")
		return
	}

	progress := bar.NewOptions(
		total,
		bar.OptionSetDescription("Splitting sprites"),
		bar.OptionShowCount(),
		bar.OptionShowIts(),
		bar.OptionSetItsString("files"),
		bar.OptionThrottle(100),
		bar.OptionClearOnFinish(),
	)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		m := spriteFilePattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}

		first, err1 := strconv.Atoi(m[1])
		second, err2 := strconv.Atoi(m[2])
		if err1 != nil || err2 != nil {
			log.Error().Str("file", e.Name()).Msg("invalid numeric part in filename")
			_ = progress.Add(1)
			continue
		}

		path := filepath.Join(extractedDir, e.Name())
		f, err := os.Open(path)
		if err != nil {
			log.Error().Str("file", path).Err(err).Msg("failed to open")
			_ = progress.Add(1)
			continue
		}
		img, err := png.Decode(f)
		_ = f.Close()
		if err != nil {
			log.Error().Str("file", path).Err(err).Msg("failed to decode PNG")
			_ = progress.Add(1)
			continue
		}

		log.Debug().Msgf("processing %s (first=%d, second=%d)", e.Name(), first, second)
		err = SplitSpriteSheet(img, first, second, splitOutputDir)
		if err != nil {
			log.Error().Err(err).Msg("failed to split")
		}
		_ = progress.Add(1)
	}
	_ = progress.Finish()
}

func GetAppearancesFileNameFromCatalogContent(in string) string {
	elems, errs := StreamCatalogContent(in)

	for {
		select {
		case e, ok := <-elems:
			if !ok {
				elems = nil
			} else {
				// Decide what to do per element type here:
				switch e.Type {
				case "appearances":
					return e.File // if you still want this side effect
				default:
					log.Debug().Msgf("skip type=%s file=%s", e.Type, e.File)
				}
			}
		case err, ok := <-errs:
			if ok && err != nil {
				log.Err(err).Msg("stream error")
			}
			errs = nil
		}
		if elems == nil && errs == nil {
			break
		}
	}

	log.Panic().Msg("no appearances file found")
	return "" // satisfy compiler
}

func getTotalToSplit(entries []os.DirEntry) int {
	total := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if spriteFilePattern.MatchString(e.Name()) {
			total++
		}
	}

	return total
}
