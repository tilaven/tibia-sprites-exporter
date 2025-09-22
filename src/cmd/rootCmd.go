package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/simivar/tibia-sprites-exporter/src/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	CatalogContentJsonPath             string
	CatalogContentJsonPathWithFilename string
	OutputPath                         string

	cfgFile           string
	debugMode         bool
	humanReadableLogs bool
)

var rootCmd = &cobra.Command{
	Short: "Tibia Sprites Exporter is set of tools for exporting Tibia sprites from the client",
	Long: `Tibia Sprites Exporter is set of tools for exporting Tibia sprites from the client.
			It is small, fast and cross-platform.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show help by default when no subcommand is provided
		return cmd.Help()
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initDebugMode)
	cobra.OnInitialize(initHumanOutput)
	cobra.OnInitialize(initPathsFromViper)
	cobra.OnInitialize(initCatalogContentJsonPathWithFilename)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tse.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().BoolVar(&humanReadableLogs, "human", false, "enable human readable mode")
	rootCmd.PersistentFlags().StringVarP(&CatalogContentJsonPath, "catalog", "c", defaultCatalogContentPath(), "path to the catalog.json file")
	rootCmd.PersistentFlags().StringVarP(&OutputPath, "output", "o", defaultOutputPath(), "path where to save the extracted sprites")

	// Bind persistent flags to Viper keys
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("human", rootCmd.PersistentFlags().Lookup("human"))
	_ = viper.BindPFlag("catalog", rootCmd.PersistentFlags().Lookup("catalog"))
	_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".tse" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".tse")
	}

	viper.SetEnvPrefix("TSE")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err == nil {
		log.Info().Msgf("Using config file: %s", viper.ConfigFileUsed())
	}
}

func initDebugMode() {
	if viper.GetBool("debug") || debugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func initHumanOutput() {
	if viper.GetBool("human") || humanReadableLogs {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

func initPathsFromViper() {
	// Sync our derived variables from Viper so config/env are respected
	if v := viper.GetString("catalog"); v != "" {
		CatalogContentJsonPath = app.ExpandPath(v)
	}
	if v := viper.GetString("output"); v != "" {
		OutputPath = app.ExpandPath(v)
	}
}

func initCatalogContentJsonPathWithFilename() {
	CatalogContentJsonPathWithFilename = filepath.Join(CatalogContentJsonPath, "catalog-content.json")
	if _, err := os.Stat(CatalogContentJsonPathWithFilename); os.IsNotExist(err) {
		log.Fatal().Msgf("catalog-content.json not found in path: %s", CatalogContentJsonPath)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func defaultCatalogContentPath() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS default path
		return app.ExpandPath(
			"~/Library/Application Support/CipSoft GmbH/Tibia/packages/Tibia.app/Contents/Resources/assets",
		)
	case "windows":
		// Windows default path (example)
		return app.ExpandPath(
			"~/AppData/Local/Tibia/packages/Tibia/assets",
		)
	case "linux":
		// Linux default path
		return app.ExpandPath(
			"~/.local/share/CipSoft GmbH/Tibia/packages/Tibia/assets",
		)
	default:
		panic(fmt.Sprintf("unsupported OS: %s", runtime.GOOS))
	}
}

func defaultOutputPath() string {
	return app.ExpandPath(
		"./output/extracted",
	)
}
