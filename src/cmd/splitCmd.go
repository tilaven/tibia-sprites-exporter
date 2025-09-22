package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	SplitOutputPath string
)

func init() {
	rootCmd.AddCommand(splitCmd)

	splitCmd.Flags().StringVar(&SplitOutputPath, "splitOutput", defaultSplitOutputPath(), "split sprites output path")
	_ = viper.BindPFlag("splitOutput", splitCmd.Flags().Lookup("splitOutput"))
}

var splitCmd = &cobra.Command{
	Use:   "split",
	Short: "Splits extracted sprites into separate files",
	Run: func(cmd *cobra.Command, args []string) {
		outputDir := app.ExpandPath(viper.GetString("output"))
		splitOutputDir := app.ExpandPath(viper.GetString("splitOutput"))

		log.Info().
			Str("output", outputDir).
			Str("splitOutput", splitOutputDir).
			Msg("Tibia Sprites Split running")

		app.SplitSprites(outputDir, splitOutputDir)

		log.Info().Msg("Tibia Sprites Split finished")
	},
}

func defaultSplitOutputPath() string {
	return app.ExpandPath(
		"./output/split",
	)
}
