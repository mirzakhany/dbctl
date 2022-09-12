package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version will be populated by the build script with the sha of the last git commit.
var version = "snapshot"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the dbctl version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dbctl version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
