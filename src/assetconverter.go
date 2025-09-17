package main

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/ulikunitz/xz/lzma"
	"golang.org/x/image/bmp"
)

// convertAsset - builds "<assetsPath><file>.lzma"
// - skips the CIP header (leading 0x00 bytes, then 4 bytes, then 7-bit int size field)
// - reads 5-byte LZMA properties + 8-byte (bogus) size, replaces size with unknown (all 0xFF)
// - LZMA-decodes the remaining bytes
// - treats the result as BMP and writes "Sprites <firstID>-<lastID>.png" to dumpToPath.
func convertAsset(assetsPath string, outputPath string, compressedFilename string, firstID int, lastID int) error {
	filePath := filepath.Join(assetsPath, compressedFilename)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			log.Debug().Msgf("Skipping '%s', doesn't exist!\n", compressedFilename)
			return nil
		}

		log.Err(err).Msgf("stat failed for %s: %w", filePath, err)
		return err
	}

	log.Debug().Msgf("Dumping '%s' to 'Sprites %d-%d.png'", compressedFilename, firstID, lastID)

	f, err := os.Open(filePath)
	if err != nil {
		log.Err(err).Msgf("open failed for %s: %w", filePath, err)

		return err
	}
	defer f.Close()

	// 1) Skip CIP header:
	//    - variable number of 0x00
	//    - then 4 bytes constant (not validated here)
	//    - then 7-bit int (keep reading while MSB=1)
	if err := skipCIPHeader(f); err != nil {
		log.Err(err).Msgf("skip header failed for %s: %w", filePath, err)

		return err
	}

	// 2) Read LZMA "alone" header parts:
	//    - 5 bytes properties
	//    - 8 bytes size (but CIP writes compressed size; we’ll overwrite with unknown)
	prop := make([]byte, 5)
	if _, err := io.ReadFull(f, prop); err != nil {
		log.Err(err).Msgf("read lzma props: %w", err)

		return err
	}
	// Discard the next 8 bytes (bogus size from CIP)
	var bogusSize [8]byte
	if _, err := io.ReadFull(f, bogusSize[:]); err != nil {
		log.Err(err).Msgf("read bogus size: %w", err)

		return err
	}

	// Prepare a corrected LZMA-alike stream:
	// Build a header: 5-byte props + 8 bytes of 0xFF (unknown uncompressed size),
	// then the remaining compressed bytes.
	var lzHeader bytes.Buffer
	lzHeader.Write(prop)
	unknown := make([]byte, 8)
	for i := range unknown {
		unknown[i] = 0xFF
	}
	lzHeader.Write(unknown)

	// MultiReader to provide a full "LZMA alone" stream to the decoder
	lzStream := io.MultiReader(&lzHeader, f)

	// 3) Decode LZMA to BMP bytes
	lzReader, err := lzma.NewReader(lzStream)
	if err != nil {
		log.Err(err).Msgf("lzma reader: %w", err)

		return err
	}
	var bmpBuf bytes.Buffer
	if _, err := io.Copy(&bmpBuf, lzReader); err != nil {
		log.Err(err).Msgf("lzma decode copy: %w", err)

		return err
	}

	// 4) Decode BMP → image.Image
	img, err := bmp.Decode(bytes.NewReader(bmpBuf.Bytes()))
	if err != nil {
		log.Err(err).Msgf("bmp decode: %w", err)

		return err
	}

	// 5) Save as PNG
	outName := fmt.Sprintf("Sprites-%d-%d.png", firstID, lastID)
	outPath := filepath.Join(outputPath, outName)

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		log.Err(err).Msgf("ensure out dir: %w", err)

		return err
	}
	out, err := os.Create(outPath)
	if err != nil {
		log.Err(err).Msgf("create out file: %w", err)

		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err := png.Encode(out, img); err != nil {
		log.Err(err).Msgf("png encode: %w", err)

		return err
	}

	return nil
}

// replace your skipCIPHeader with this
func skipCIPHeader(r io.Reader) error {
	br := &byteReader{r: r}

	// 1) Skip leading 0x00 bytes; consume the first non-zero byte (part of 5-byte constant)
	for {
		b, err := br.ReadByte()
		if err != nil {
			return err
		}
		if b != 0x00 {
			// C#: while(ReadByte()==0){} -> we have CONSUMED this non-zero byte
			break
		}
	}

	// 2) Skip the remaining 4 bytes of the constant
	var tmp4 [4]byte
	if _, err := io.ReadFull(br, tmp4[:]); err != nil {
		return err
	}

	// 3) Skip the 7-bit length (read until a byte with MSB==0)
	for {
		b, err := br.ReadByte()
		if err != nil {
			return err
		}
		if (b & 0x80) == 0 {
			break
		}
	}
	return nil
}

// Small helper to allow peeking/unreading a single byte during header parsing.
type byteReader struct {
	r   io.Reader
	buf *byte // one-byte pushback buffer
}

func (br *byteReader) Read(p []byte) (int, error) {
	if br.buf != nil && len(p) > 0 {
		p[0] = *br.buf
		br.buf = nil
		if len(p) == 1 {
			return 1, nil
		}
		n, err := br.r.Read(p[1:])
		return n + 1, err
	}
	return br.r.Read(p)
}

func (br *byteReader) ReadByte() (byte, error) {
	var b [1]byte
	_, err := br.Read(b[:])
	return b[0], err
}

func (br *byteReader) unreadByte(b byte) {
	br.buf = &b
}
