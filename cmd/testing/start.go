package testing

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

//
// dbctl testing -- \
//			pg -p 5435 -m ./migrations -f ./fixtures - \
//   		rs -p 7654
//

// GetStartTestingCmd represents the start testing command
func GetStartTestingCmd(rootCmd *cobra.Command) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "testing -- pg [options] - rs [options]",
		Short: "Start dbctl server for unit testing",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			var cmdParts []string
			var cmdList [][]string
			for _, arg := range args {
				if arg == "-" {
					if len(cmdParts) > 0 {
						cmdList = append(cmdList, cmdParts)
						cmdParts = []string{}
					}
				} else {
					cmdParts = append(cmdParts, arg)
				}
			}
			cmdList = append(cmdList, cmdParts)

			// run db commands. a database that fails to start has to stop the whole
			// command: carrying on would leave the api server pointing at nothing, or
			// worse, at an instance left behind by an earlier run.
			for _, cmdParts := range cmdList {
				m := []string{"start", "-d"}
				m = append(m, cmdParts...)
				rootCmd.SetArgs(m)
				if err := rootCmd.Execute(); err != nil {
					return fmt.Errorf("starting %s failed: %w", strings.Join(cmdParts, " "), err)
				}
			}

			// run api server
			rootCmd.SetArgs([]string{"api-server", "-t"})
			if err := rootCmd.Execute(); err != nil {
				return fmt.Errorf("starting api server failed: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().SetInterspersed(false)
	return cmd
}
