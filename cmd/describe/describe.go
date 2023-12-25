package describe

import "github.com/spf13/cobra"

// GetDescribeCmd represents the describe command
func GetDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe a database instance",
		Long:  `Describe a database instance in detail, including its tables, indexes, and foreign keys.`,
	}

	cmd.AddCommand(GetDescribePgCmd())
	return cmd
}
