package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// GetStopCmd represents the stop command
func GetStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a database instance",
		RunE:  runStop,
	}
	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	p := GetDownCmd()

	if contain(args, "postgres", "pg") {
		p.SetArgs([]string{"pg"})
		if err := p.Execute(); err != nil {
			return err
		}
	}

	if contain(args, "redis", "rs") {
		p.SetArgs([]string{"rs"})
		return p.Execute()
	}

	return errors.New("invalid args, can be postgres(pg) and/or redis(rs)")
}
