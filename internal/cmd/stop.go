package cmd

import (
	"context"
	"errors"

	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
	pg "github.com/mirzakhany/dbctl/internal/database/postgres"
	"github.com/mirzakhany/dbctl/internal/database/redis"
	"github.com/mirzakhany/dbctl/internal/utils"
	"github.com/spf13/cobra"
)

// GetStopCmd represents the stop command
func GetStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop {rs pg}",
		Short: "stop one or more detached databases",
		RunE:  runStop,
	}
	return cmd
}

func runStop(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("invalid args, can be postgres(pg) and/or redis(rs)")
	}

	ctx := utils.ContextWithOsSignal()

	if utils.Contain(args, "pg", "postgres") {
		items, err := pg.Instances(ctx)
		if err != nil {
			return err
		}

		if err := remove(ctx, items); err != nil {
			return err
		}
	}

	if utils.Contain(args, "rs", "redis") {
		items, err := redis.Instances(ctx)
		if err != nil {
			return err
		}

		if err := remove(ctx, items); err != nil {
			return err
		}
	}

	return nil
}

func remove(ctx context.Context, dbs []database.Info) error {
	for _, i := range dbs {
		if err := container.TerminateByID(ctx, i.ID); err != nil {
			return err
		}
	}
	return nil
}
