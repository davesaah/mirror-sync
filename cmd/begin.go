/*
Copyright © 2026 David Saah davesaah@gmail.com
*/
package cmd

import (
	"github.com/davesaah/mirror-sync/core"
	"github.com/spf13/cobra"
)

// beginCmd represents the begin command
var beginCmd = &cobra.Command{
	Use:     "begin",
	Short:   "Begin mirror syncing",
	Long:    "Start to sync the defined local gitea repository to github, codeberg and gitlab",
	PostRun: func(cmd *cobra.Command, args []string) {},
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName, _ := cmd.Flags().GetString("repo")
		localOwner, _ := cmd.Flags().GetString("owner")
		isPrivate, _ := cmd.Flags().GetBool("private")

		visibility := "public"
		if isPrivate {
			visibility = "private"
		}

		return core.Run(repoName, localOwner, visibility)
	},
}

func init() {
	rootCmd.AddCommand(beginCmd)

	// Flags and configuration settings.
	beginCmd.Flags().StringP("repo", "n", "", "Name of repository")
	beginCmd.Flags().StringP("owner", "u", "", "Local owner (user/organization)")
	beginCmd.Flags().BoolP("private", "p", false, "Set visibility of the repo to private")
}
