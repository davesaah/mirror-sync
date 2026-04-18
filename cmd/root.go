/*
Copyright © 2026 David Saah davesaah@gmail.com
*/
// Package cmd defines a command line interface for mirror-sync
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "mirror-sync",
	Short:   "Create mirrors for your local git repos",
	Version: "1.0",
	Long:    "Sync your local gitea repos to github, codeberg and gitlab",
	Run:     func(cmd *cobra.Command, args []string) {},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// setting up config
	userHomeDir, _ := os.UserHomeDir()
	cfgPath := fmt.Sprintf("%s/.config/mirror-sync.json", userHomeDir)

	// make sure file exists
	_, err := os.ReadFile(cfgPath)
	if err != nil {
		os.WriteFile(cfgPath, []byte("{}"), 0664)
	}

	// load config
	viper.SetConfigFile(cfgPath)
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}

	rootCmd.Flags().BoolP("help", "h", false, "Show help")
	rootCmd.Flags().BoolP("version", "v", false, "Show version")
}
