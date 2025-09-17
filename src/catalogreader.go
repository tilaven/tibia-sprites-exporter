package main

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/rs/zerolog/log"
)

func readCatalogContent(in string) {
	var r *os.File
	var err error
	r, err = os.Open(in)
	if err != nil {
		log.Err(err).Msgf("failed to open input: %v", err)
	}
	defer r.Close()

	dec := json.NewDecoder(bufio.NewReaderSize(r, 1<<20)) // 1 MB buffer
	// Expect a top-level array
	tok, err := dec.Token()
	if err != nil {
		log.Err(err).Msgf("failed reading first token: %v", err)
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		log.Err(err).Msgf("expected top-level JSON array")
	}

	// Define a minimal struct so we only decode what we need.
	var elem struct {
		Type          string `json:"type"`
		File          string `json:"file"`
		SpriteType    int    `json:"spritetype"`
		FirstSpriteId int    `json:"firstspriteid"`
		LastSpriteId  int    `json:"lastspriteid"`
		Area          int    `json:"area"`
	}

	for dec.More() {
		// Zero the struct each iteration to avoid accidental reuse
		elem = struct {
			Type          string `json:"type"`
			File          string `json:"file"`
			SpriteType    int    `json:"spritetype"`
			FirstSpriteId int    `json:"firstspriteid"`
			LastSpriteId  int    `json:"lastspriteid"`
			Area          int    `json:"area"`
		}{}

		if err := dec.Decode(&elem); err != nil {
			log.Err(err).Msgf("decode error: %v", err)
		}
		if elem.Type == "sprite" && elem.File != "" {
			err := convertAsset(
				CatalogContentJsonPath,
				OutputPath,
				elem.File,
				elem.FirstSpriteId,
				elem.LastSpriteId,
			)
			if err != nil {
				log.Err(err).Msgf("convertAsset error: %v", err)
			}

			log.Debug().Msgf("Parsed element: %+v", elem)
		} else {
			log.Debug().Msgf("Skipping element: %+v", elem)
		}

	}

	// Consume the closing ']'
	if tok, err = dec.Token(); err != nil {
		log.Err(err).Msgf("failed reading closing token: %v", err)
	}
}
