package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/ulikunitz/xz/lzma"
	"golang.org/x/image/bmp"
)

// convertAsset:
//  1. open "<assetsPath>/<compressedFilename>"
//  2. skip CIP header (leading 0x00s, 4-byte constant, 7-bit length)
//  3. repair LZMA "alone" header (props + unknown size) and decode
//  4. decode BMP
//  5. write PNG as "Sprites-<firstID>-<lastID>.png" into outputPath
func convertAsset(assetsPath, outputPath, compressedFilename string, firstID, lastID int) error {
	inPath := filepath.Join(assetsPath, compressedFilename)

	f, err := os.Open(inPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Str("file", compressedFilename).Msg("skipping: file does not exist")
			return nil
		}
		return fmt.Errorf("open %q: %w", inPath, err)
	}
	defer f.Close()

	log.Debug().
		Str("input", compressedFilename).
		Str("output", fmt.Sprintf("Sprites-%d-%d.png", firstID, lastID)).
		Msg("converting")

	br := bufio.NewReaderSize(f, 1<<20) // 1MB buffer for fewer syscalls

	// 1) Skip CIP header
	if err := skipCIPHeader(br); err != nil {
		return fmt.Errorf("skip CIP header: %w", err)
	}

	// 2) Build an LZMA reader from the remaining stream (repair header)
	lzReader, err := newLZMAReader(br)
	if err != nil {
		return fmt.Errorf("lzma reader: %w", err)
	}

	// 3) LZMA→BMP bytes
	var bmpBuf bytes.Buffer
	if _, err := io.Copy(&bmpBuf, lzReader); err != nil {
		return fmt.Errorf("lzma decode: %w", err)
	}

	// 4) BMP→image.Image
	img, err := bmp.Decode(bytes.NewReader(bmpBuf.Bytes()))
	if err != nil {
		return fmt.Errorf("bmp decode: %w", err)
	}

	// 5) Write PNG
	outName := fmt.Sprintf("Sprites-%d-%d.png", firstID, lastID)
	outPath := filepath.Join(outputPath, outName)
	if err := writePNG(outPath, img); err != nil {
		return fmt.Errorf("write png %q: %w", outPath, err)
	}

	// Optionally split into individual sprites named by ID
	if SplitSprites {
		if err := splitSpriteSheet(img, firstID, lastID, outputPath); err != nil {
			return fmt.Errorf("split sprites: %w", err)
		}
	}

	return nil
}

// skipCIPHeader consumes:
//   - all leading 0x00 bytes
//   - then 4 bytes (constant marker)
//   - then a 7-bit length (continue while MSB=1)
func skipCIPHeader(r *bufio.Reader) error {
	// Skip leading zeros; consume first non-zero byte.
	for {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		if b != 0x00 {
			break // consumed first non-zero byte (part of constant)
		}
	}
	// Skip remaining 4 bytes of the constant marker.
	if _, err := io.CopyN(io.Discard, r, 4); err != nil {
		return err
	}
	// Skip 7-bit length (bytes with MSB=1 mean "more").
	for {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		if (b & 0x80) == 0 {
			break
		}
	}
	return nil
}

// newLZMAReader reads the 5-byte props + bogus 8-byte size,
// replaces size with 0xFF..FF (unknown), and returns a decoder for the rest.
func newLZMAReader(r *bufio.Reader) (io.Reader, error) {
	props := make([]byte, 5)
	if _, err := io.ReadFull(r, props); err != nil {
		return nil, fmt.Errorf("read props: %w", err)
	}
	// Discard bogus size (CIP writes compressed size).
	if _, err := io.CopyN(io.Discard, r, 8); err != nil {
		return nil, fmt.Errorf("discard bogus size: %w", err)
	}

	// Build corrected "LZMA alone" header: props + unknown size (all 0xFF).
	var header bytes.Buffer
	header.Write(props)
	header.Write(bytes.Repeat([]byte{0xFF}, 8))

	stream := io.MultiReader(&header, r)
	rd, err := lzma.NewReader(stream)
	if err != nil {
		return nil, err
	}
	return rd, nil
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	return png.Encode(out, img)
}

// splitSpriteSheet slices a 384x384 sheet into 32x32 or 64x64 tiles in
// row-major order and writes each tile as a PNG named by its sprite ID.
func splitSpriteSheet(img image.Image, firstID, lastID int, outputDir string) error {
	count := lastID - firstID + 1
	if count <= 0 {
		return nil
	}

	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width != 384 || height != 384 {
		// Proceed anyway, but log that size is unexpected
		log.Debug().Int("w", width).Int("h", height).Msg("unexpected sheet size; proceeding to split")
	}

	tile := 32
	if count <= 36 {
		tile = 64
	}

	cols := width / tile
	rows := height / tile
	maxTiles := cols * rows
	if count > maxTiles {
		log.Warn().Int("count", count).Int("capacity", maxTiles).Msg("sprite count exceeds sheet capacity; truncating")
		count = maxTiles
	}

	id := firstID
	idx := 0
	for r := 0; r < rows && idx < count; r++ {
		for c := 0; c < cols && idx < count; c++ {
			// Source rect in the sheet
			sr := image.Rect(b.Min.X+c*tile, b.Min.Y+r*tile, b.Min.X+(c+1)*tile, b.Min.Y+(r+1)*tile)
			// Copy into a new RGBA tile
			dst := image.NewRGBA(image.Rect(0, 0, tile, tile))
			draw.Draw(dst, dst.Bounds(), img, sr.Min, draw.Src)

			outPath := filepath.Join(outputDir, "split", fmt.Sprintf("%d.png", id))
			if err := writePNG(outPath, dst); err != nil {
				return fmt.Errorf("write sprite %d: %w", id, err)
			}

			id++
			idx++
		}
	}
	return nil
}
