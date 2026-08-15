package cmd

import (
	"fmt"
	"os"

	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/logger"
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
		TraverseChildren: true,
		// a database that failed to start is not a usage problem, printing the
		// whole help text on top of the error only hides it.
		SilenceUsage: true,
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			listen, err := c.Flags().GetString("listen")
			if err != nil {
				return fmt.Errorf("invalid listen args, %w", err)
			}

			if listen != "" {
				if err := os.Setenv(container.EnvListen, listen); err != nil {
					return err
				}
			}

			// databases dbctl starts have well known credentials, publishing them
			// beyond this machine hands them to anyone who can reach it.
			if container.IsPublic() {
				logger.Info(fmt.Sprintf(
					"WARNING: publishing ports on %s, the databases and the api server are reachable from your network with their default credentials",
					container.ListenAddress()))
			}

			return nil
		},
	}

	cmd.PersistentFlags().String("label", "", "Label to add to the running container or api-server")
	cmd.PersistentFlags().String("listen", "",
		"Address published ports are bound to, defaults to "+container.LoopbackAddress+" ($"+container.EnvListen+")")

	return cmd
}
