package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

var (
	AppearancesFileName string
)

type CatalogElem struct {
	Type          string `json:"type"`
	File          string `json:"file"`
	SpriteType    int    `json:"spritetype"`
	FirstSpriteId int    `json:"firstspriteid"`
	LastSpriteId  int    `json:"lastspriteid"`
	Area          int    `json:"area"`
}

func ReadCatalogContent(in string) {
	elems, errs := StreamCatalogContent(in)

	for {
		select {
		case e, ok := <-elems:
			if !ok {
				elems = nil
			} else {
				// Decide what to do per element type here:
				switch e.Type {
				case "sprite":
					log.Debug().Msgf("sprite range %d..%d file=%s", e.FirstSpriteId, e.LastSpriteId, e.File)
					err := convertAsset(
						CatalogContentJsonPath,
						OutputPath,
						e.File,
						e.FirstSpriteId,
						e.LastSpriteId,
					)
					if err != nil {
						log.Err(err).Msg("failed to convert asset")
					}
				case "appearances":
					AppearancesFileName = e.File // if you still want this side effect
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
}

// StreamCatalogContent opens the JSON and streams elems as they are decoded.
// It does NOT call convertAsset or mutate globals. Errors are sent on errs.
func StreamCatalogContent(path string) (<-chan CatalogElem, <-chan error) {
	out := make(chan CatalogElem)
	errs := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errs)

		r, err := os.Open(path)
		if err != nil {
			errs <- err
			return
		}
		defer r.Close()

		dec := json.NewDecoder(bufio.NewReaderSize(r, 1<<20)) // 1 MB buffer

		// Expect top-level '['
		tok, err := dec.Token()
		if err != nil {
			errs <- err
			return
		}
		if d, ok := tok.(json.Delim); !ok || d != '[' {
			errs <- fmt.Errorf("expected top-level JSON array")
			return
		}

		var elem CatalogElem
		for dec.More() {
			elem = CatalogElem{} // reset
			if err := dec.Decode(&elem); err != nil {
				errs <- err
				return
			}
			out <- elem
		}

		// Consume closing ']'
		if _, err := dec.Token(); err != nil {
			errs <- err
			return
		}
	}()

	return out, errs
}
