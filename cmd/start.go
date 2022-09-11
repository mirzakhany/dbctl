package cmd

import (
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a database instance",
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.PersistentFlags().Uint32P("port", "p", 15432, "default port")
	startCmd.PersistentFlags().BoolP("detach", "d", false, "Detached mode: Run database in the background")
	startCmd.PersistentFlags().Bool("use-docker", true, "Use Docker to run databases, default is true")
}
