package app

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
	bar "github.com/schollz/progressbar/v3"
)

type spriteInfo struct {
	SpriteIDs []int
}

func GroupSplitSprites(catalogContentJsonPath, appearancesFileName, splitSpitesDir, outputGroupedDir string) {
	datPath := filepath.Join(catalogContentJsonPath, appearancesFileName)
	if _, err := os.Stat(datPath); err != nil {
		log.Fatal().Msgf("[read] dat file not found: %v", err)
	}

	data, err := os.ReadFile(datPath)
	if err != nil {
		log.Fatal().Msgf("[read] failed to read dat file: %v", err)
	}
	log.Debug().Msgf("[read] appearances.dat bytes=%d", len(data))

	groups := scanSpriteInfos(data)
	log.Debug().Msgf("[parse] found %d candidate groups (sprite-info blocks)", len(groups))

	if err := os.MkdirAll(outputGroupedDir, 0o755); err != nil {
		log.Fatal().Msgf("[fs] failed to create outputGroupedDir=%s: %v", outputGroupedDir, err)
	}
	log.Debug().Msgf("[fs] outputGroupedDir directory ready: %s", outputGroupedDir)

	exported, skipped, failPNG := 0, 0, 0
	progress := bar.NewOptions(
		len(groups),
		bar.OptionSetDescription("Grouping sprites"),
		bar.OptionShowCount(),
		bar.OptionShowIts(),
		bar.OptionSetItsString("groups"),
		bar.OptionThrottle(100),
		bar.OptionClearOnFinish(),
	)
	for idx, g := range groups {
		if len(g.SpriteIDs) == 0 {
			skipped++
			if idx < 5 {
				log.Debug().Msgf("[skip #%d] no sprite IDs", idx)
			}
			_ = progress.Add(1)
			continue
		}

		first, last := g.SpriteIDs[0], g.SpriteIDs[len(g.SpriteIDs)-1]
		base := strconv.Itoa(first)
		if first != last {
			base = fmt.Sprintf("%d-%d", first, last)
		}
		outPNG := filepath.Join(outputGroupedDir, base+".png")

		log.Debug().Int("group", idx).Int("sprites", len(g.SpriteIDs)).Msg("compose group")

		img, err := composeGroupImage(splitSpitesDir, g)
		if err != nil {
			failPNG++
			log.Error().Msgf("[compose #%d] %v", idx, err)
			_ = progress.Add(1)
			continue
		}
		if err := writePNG(outPNG, img); err != nil {
			failPNG++
			log.Error().Msgf("[writePNG #%d] %v", idx, err)
			_ = progress.Add(1)
			continue
		}
		log.Debug().Int("group", idx).Str("outPNG", outPNG).Msg("wrote grouped PNG")
		exported++
		_ = progress.Add(1)
	}
	_ = progress.Finish()

	log.Info().
		Int("exported", exported).
		Int("skipped", skipped).
		Int("pngErrors", failPNG).
		Str("outputGroupedDir", outputGroupedDir).
		Msg("Exporting groups finished")
}

func scanSpriteInfos(buf []byte) []spriteInfo {
	out := make([]spriteInfo, 0, 1024)
	n, i, scanned := len(buf), 0, 0

	for i < n-8 {
		if !(buf[i] == 0x08 && i+6 < n && buf[i+2] == 0x10 && buf[i+4] == 0x18 && buf[i+6] == 0x20) {
			i++
			continue
		}
		w, p, ok := readVarint(buf, i+1)
		if !ok {
			i++
			continue
		}
		h, p, ok := readVarint(buf, p+1)
		if !ok {
			i++
			continue
		}
		l, p, ok := readVarint(buf, p+1)
		if !ok {
			i++
			continue
		}
		pw, p, ok := readVarint(buf, p+1)
		if !ok {
			i++
			continue
		}
		if w <= 0 || h <= 0 || l <= 0 {
			i++
			continue
		}

		ids := make([]int, 0, 64)
		k := p
		for k < n && buf[k] == 0x28 {
			v, k2, ok := readVarint(buf, k+1)
			if !ok {
				break
			}
			ids = append(ids, v)
			k = k2
			if len(ids) > 1_000_000 {
				break
			}
		}
		out = append(out, spriteInfo{SpriteIDs: ids})
		if scanned < 5 {
			log.Printf("[scan] off=%d w=%d h=%d layers=%d pw=%d ids=%d", i, w, h, l, pw, len(ids))
		}
		scanned++
		i = k
	}
	log.Printf("[scan] total=%d sprite-info blocks", len(out))
	return out
}

func readVarint(buf []byte, i int) (int, int, bool) {
	var x uint64
	var s uint
	start := i
	for {
		if i >= len(buf) {
			return 0, start, false
		}
		b := buf[i]
		if b < 0x80 {
			if s >= 64 {
				return 0, start, false
			}
			x |= uint64(b) << s
			i++
			return int(x), i, true
		}
		x |= uint64(b&0x7F) << s
		s += 7
		i++
		if s > 70 {
			return 0, start, false
		}
	}
}

func composeGroupImage(splitSpitesDir string, g spriteInfo) (image.Image, error) {
	total := len(g.SpriteIDs)
	if total == 0 {
		return nil, errors.New("no sprite ids")
	}

	tiles := make([]image.Image, total)
	var tileW, tileH int
	for idx := 0; idx < total; idx++ {
		path := filepath.Join(splitSpitesDir, strconv.Itoa(g.SpriteIDs[idx])+".png")
		img, err := loadPNG(path)
		if err != nil {
			log.Error().Str("file", path).Msg("tile error")
			continue
		}
		if tileW == 0 {
			b := img.Bounds()
			tileW, tileH = b.Dx(), b.Dy()
		}
		tiles[idx] = img
	}
	if tileW == 0 {
		return nil, errors.New("no tiles found for this group (check spritesDir)")
	}
	if tileW < 32 || tileH < 32 {
		return nil, errors.New("tile size too small")
	}

	dst := image.NewNRGBA(image.Rect(0, 0, tileW*total, tileH))
	for i := 0; i < total; i++ {
		if tiles[i] == nil {
			continue
		}
		pt := image.Pt(i*tileW, 0)
		draw.Draw(dst, image.Rectangle{Min: pt, Max: pt.Add(tiles[i].Bounds().Size())}, tiles[i], tiles[i].Bounds().Min, draw.Over)
	}
	return dst, nil
}

func loadPNG(path string) (image.Image, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err // file truly not found or perms
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err // open failure (perm/locks)
	}
	defer f.Close()

	// try generic decoder (gives you the real reason if decoding fails)
	img, format, err := image.Decode(f)
	if err != nil {
		log.Fatal().Msgf("[decode] failed: size=%dB format=%q: %v", fi.Size(), format, err)
	}
	return img, nil
}
