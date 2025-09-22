package cmd

import (
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(extractCmd)
}

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extracts sprites from the Tibia client",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("Tibia Sprites extract running")

		catalogDir := app.ExpandPath(viper.GetString("catalog"))
		outputDir := app.ExpandPath(viper.GetString("output"))
		catalogFile := filepath.Join(catalogDir, "catalog-content.json")

		// Update globals so downstream helpers/logs stay consistent
		CatalogContentJsonPath = catalogDir
		CatalogContentJsonPathWithFilename = catalogFile
		OutputPath = outputDir

		app.ConvertAssetsFromCatalogContent(catalogDir, catalogFile, outputDir)

		log.Info().Msg("Tibia Sprites extract finished")
	},
}
