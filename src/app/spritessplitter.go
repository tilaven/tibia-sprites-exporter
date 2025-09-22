package app

import (
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/rs/zerolog/log"
)

var spriteFilePattern = regexp.MustCompile(`^Sprites-(\d+)-(\d+)\.png$`)

func SplitSprites(extractedDir, splitOutputDir string) {
	entries, err := os.ReadDir(extractedDir)
	if err != nil {
		log.Panic().Err(err).Msgf("failed to read dir=%s", extractedDir)
		return
	}

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
			continue
		}

		path := filepath.Join(extractedDir, e.Name())
		f, err := os.Open(path)
		if err != nil {
			log.Error().Str("file", path).Err(err).Msg("failed to open")
			continue
		}
		img, err := png.Decode(f)
		_ = f.Close()
		if err != nil {
			log.Error().Str("file", path).Err(err).Msg("failed to decode PNG")
			continue
		}

		log.Debug().Msgf("processing %s (first=%d, second=%d)", e.Name(), first, second)
		err = SplitSpriteSheet(img, first, second, splitOutputDir)
		if err != nil {
			log.Error().Err(err).Msg("failed to split")
			continue
		}
	}
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
