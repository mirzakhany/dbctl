package testing

import (
	"fmt"

	"github.com/mirzakhany/dbctl/internal/apiserver"
	"github.com/mirzakhany/dbctl/internal/utils"
	"github.com/spf13/cobra"
)

// GetTestingApiServerCmd represents the testing api server command
func GetTestingApiServerCmd() *cobra.Command {
	c := &cobra.Command{
		Aliases: []string{"api-server"},
		Use:     "api-server",
		Short:   "api server is a http testing server to manage databases",
		RunE:    runTestingApiServer,
	}

	c.Flags().StringP("port", "p", apiserver.DefaultPort, "testing server default port")
	return c
}

func runTestingApiServer(cmd *cobra.Command, args []string) error {
	port, err := cmd.Flags().GetString("port")
	if err != nil {
		return fmt.Errorf("invalid port args, %w", err)
	}

	server := apiserver.NewServer(port)

	return server.Start(utils.ContextWithOsSignal())
}
