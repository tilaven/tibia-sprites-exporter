package main

import (
	"log"
)

func main() {
	log.Printf("[info] Tibia Sprites Exporter starting")

	initExporter()
	readCatalogContent(CatalogContentJsonFullPath)
}
