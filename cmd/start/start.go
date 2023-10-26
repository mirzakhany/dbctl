package start

import (
	"github.com/spf13/cobra"
)

// GetStartCmd represents the start command
func GetStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a database instance",
	}

	cmd.PersistentFlags().BoolP("detach", "d", false, "Detached mode: Run database in the background")
	cmd.PersistentFlags().Bool("ui", false, "Run ui component if available for chosen database")

	cmd.AddCommand(GetPgCmd())
	cmd.AddCommand(GetRedisCmd())
	return cmd
}
