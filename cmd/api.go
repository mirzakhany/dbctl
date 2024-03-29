package cmd

import (
	"fmt"
	"time"

	"github.com/mirzakhany/dbctl/internal/apiserver"
	"github.com/mirzakhany/dbctl/internal/utils"
	"github.com/spf13/cobra"
)

// GetTestingAPIServerCmd represents the testing api server command
func GetTestingAPIServerCmd() *cobra.Command {
	c := &cobra.Command{
		Aliases: []string{"api-server"},
		Use:     "api-server",
		Short:   "api server is a http testing server to manage databases",
		RunE:    runTestingAPIServer,
	}

	c.Flags().StringP("port", "p", apiserver.DefaultPort, "testing server default port")
	c.Flags().BoolP("testing", "t", false, "run in testing mode with containerized server")
	return c
}

func runTestingAPIServer(cmd *cobra.Command, args []string) error {
	port, err := cmd.Flags().GetString("port")
	if err != nil {
		return fmt.Errorf("invalid port args, %w", err)
	}

	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return fmt.Errorf("invalid label args, %w", err)
	}

	testing, err := cmd.Flags().GetBool("testing")
	if err != nil {
		return fmt.Errorf("invalid testing args, %w", err)
	}

	if testing {
		return apiserver.RunAPIServerContainer(utils.ContextWithOsSignal(), port, label, 20*time.Second)
	}

	server := apiserver.NewServer(port)
	return server.Start(utils.ContextWithOsSignal())
}
