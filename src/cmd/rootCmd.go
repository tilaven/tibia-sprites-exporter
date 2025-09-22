package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/cobra"
)

func init() {
	// Ensure the app's standard flags exist and make Cobra parse them too.
	app.EnsureFlagsDefined()
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

var rootCmd = &cobra.Command{
	Use:   "tse",
	Short: "Tibia Sprites Exporter is set of tools for exporting Tibia sprites from the client",
	Long: `Tibia Sprites Exporter is set of tools for exporting Tibia sprites from the client.
			It is small, fast and cross-platform.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("Tibia Sprites Exporter running")
		app.InitExporter()
		app.ReadCatalogContent(app.CatalogContentJsonFullPath)
		app.GroupSplitSprites()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
