package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// GetVersionCmd represents the version command
func GetVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the dbctl version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dbctl version %s\n", version)
		},
	}
}
