/*
Copyright © 2026 David Saah davesaah@gmail.com
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use: "config",
	PostRunE: func(cmd *cobra.Command, args []string) error {
		return viper.WriteConfig()
	},
	Short: "Configure options for mirror sync",
	Long: `Configure options for mirror sync. For each provider, ensure your
access token has the following permissions.

github:
  + admin:repo_hook
  + delete_repo
  + repo
  + workflow

codeberg:
  + write:organization
  + write:repository
  + write:user

gitlab:
  + API
  + READ REPOSITORY
  + READ API
  + WRITE REPOSITORY

local gitea:
  + write:organization
  + write:repository

[INFO] Config file is found at $HOME/.config/mirror-sync.json
`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().String("github-token", "", "Add github access token")
	configCmd.Flags().String("gitlab-token", "", "Add gitlab access token")
	configCmd.Flags().String("codeberg-token", "", "Add codeberg access token")
	configCmd.Flags().String("localhost-token", "", "Add local gitea server access token")
	configCmd.Flags().String("local-url", "", "Set local gitea server url (e.g. https://gitea.com)")
	configCmd.Flags().String("external-user", "", "Set username on cloud providers")
	viper.BindPFlags(configCmd.Flags())
}
