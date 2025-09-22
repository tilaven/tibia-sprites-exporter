package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/cobra"
)

var (
	SplitOutputPath string
)

func init() {
	rootCmd.AddCommand(splitCmd)

	splitCmd.Flags().StringVar(&SplitOutputPath, "splitOutput", defaultSplitOutputPath(), "split sprites output path")
}

var splitCmd = &cobra.Command{
	Use:   "split",
	Short: "Splits extracted sprites into separate files",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().
			Str("output", OutputPath).
			Str("splitOutput", SplitOutputPath).
			Msg("Tibia Sprites Split running")

		app.SplitSprites(OutputPath, SplitOutputPath)

		log.Info().Msg("Tibia Sprites Split finished")
	},
}

func defaultSplitOutputPath() string {
	return app.ExpandPath(
		"./output/split",
	)
}
