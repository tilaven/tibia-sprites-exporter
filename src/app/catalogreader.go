package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type CatalogElem struct {
	Type          string `json:"type"`
	File          string `json:"file"`
	SpriteType    int    `json:"spritetype"`
	FirstSpriteId int    `json:"firstspriteid"`
	LastSpriteId  int    `json:"lastspriteid"`
	Area          int    `json:"area"`
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
