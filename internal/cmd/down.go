package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/spf13/cobra"
)

// GetDownCmd represents the down command
func GetDownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down {rs pg}",
		Short: "stop one or more detached databases",
		RunE:  runDown,
	}
	return cmd
}

func runDown(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("invalid args, can be postgres(pg) and/or redis(rs)")
	}

	ctx := contextWithOsSignal()
	containers, err := container.List(ctx)
	if err != nil {
		return err
	}

	toRemove := make(map[string]struct{})
	if contain(args, "postgres", "pg") {
		toRemove["pg"] = struct{}{}
	}

	if contain(args, "redis", "rs") {
		toRemove["rs"] = struct{}{}
	}

	for _, c := range containers {
		t := strings.Split(c.Name, "_")[1]
		if _, ok := toRemove[t]; ok {
			if err := container.Remove(ctx, c); err != nil {
				return fmt.Errorf("stop database %s failed", c.Name)
			}
		}
	}

	return nil
}
