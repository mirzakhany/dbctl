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
		Use:   "stop {rs pg id all}",
		Short: "stop one or more detached databases",
		Long: `using this command you can stop one or more detached databases by their type or id, 
		for example: dbctl stop pg rs or dbctl stop 969ec9747052`,
		RunE: runStop,
	}
	return cmd
}

func runStop(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("invalid args, can be postgres(pg) and/or redis(rs) or instance id")
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

	// it could be the case that user sent instance id instead of type
	// so we try to remove it
	// TODO check if database is in detached mode and warn user
	// TODO hot fixing a bug, need to be refactored
	if len(args) == 1 && args[0] != "pg" && args[0] != "rs" && args[0] != "postgres" && args[0] != "redis" {
		// check if its all then remove all
		if args[0] == "all" {
			containers, err := container.List(ctx, nil)
			if err != nil {
				return err
			}

			for _, c := range containers {
				if err := container.TerminateByID(ctx, c.ID); err != nil {
					return err
				}
			}
			return nil
		}

		// remove by id
		if err := container.TerminateByID(ctx, args[0]); err != nil {
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
