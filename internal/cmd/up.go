package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// GetUpCmd represents the up command
func GetUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "started one or more database type in detached mode",
		RunE:  runUp,
	}
	return cmd
}

func runUp(cmd *cobra.Command, args []string) error {
	p := GetStartCmd()

	if contain(args, "postgres", "pg") {
		p.SetArgs([]string{"pg", "-d"})
		if err := p.Execute(); err != nil {
			return err
		}
	}

	if contain(args, "redis", "rs") {
		p.SetArgs([]string{"rs", "-d"})
		return p.Execute()
	}

	return errors.New("invalid args, can be postgres(pg) and/or redis(rs)")
}
