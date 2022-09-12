package cmd

import (
	"github.com/spf13/cobra"
)

// GetRootCmd represents the base command when called without any subcommands
func GetRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dbctl",
		Version: version,
		Short:   "Your swish knife of testing databases",
		Long: `Dbctl is a command line tools, providing simple 
command to run and manage databases for tests proposes`,
	}

	return cmd
}
