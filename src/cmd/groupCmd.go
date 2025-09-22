package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/cobra"
)

var (
	GroupedOutputPath string
)

func init() {
	rootCmd.AddCommand(groupCmd)

	groupCmd.Flags().StringVar(&SplitOutputPath, "splitOutput", defaultSplitOutputPath(), "split sprites output path")
	groupCmd.Flags().StringVar(&GroupedOutputPath, "groupedOutput", defaultGroupedOutputPath(), "grouped sprites by appearances.json output path")
}

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Groups sprites from the Tibia client based on the appearances file",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("Tibia Sprites group running")

		appearancesFileName := app.GetAppearancesFileNameFromCatalogContent(CatalogContentJsonPathWithFilename)
		log.Info().Msgf("Appearances file name: %s", appearancesFileName)

		app.GroupSplitSprites(CatalogContentJsonPath, appearancesFileName, SplitOutputPath, GroupedOutputPath)

		log.Info().Msg("Tibia Sprites group finished")
	},
}

func defaultGroupedOutputPath() string {
	return app.ExpandPath(
		"./output/grouped",
	)
}
