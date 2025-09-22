package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(extractCmd)
}

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extracts sprites from the Tibia client",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("Tibia Sprites extract running")

		app.ConvertAssetsFromCatalogContent(CatalogContentJsonPath, CatalogContentJsonPathWithFilename, OutputPath)

		log.Info().Msg("Tibia Sprites extract finished")
	},
}
