package testing

import (
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
		Run: func(cobraCmd *cobra.Command, args []string) {
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

			for _, cmdParts := range cmdList {
				m := []string{"start", "-d"}
				m = append(m, cmdParts...)
				rootCmd.SetArgs(m)
				rootCmd.Execute()
			}
		},
	}

	cmd.Flags().SetInterspersed(false)
	return cmd
}
