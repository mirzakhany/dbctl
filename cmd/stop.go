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
		Use:   "stop {rs pg id all <label>}",
		Short: "stop one or more detached databases",
		Long: `using this command you can stop one or more detached databases by their type, id or label
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

		if err := removeByInfo(ctx, items); err != nil {
			return err
		}
	}

	if utils.Contain(args, "rs", "redis") {
		items, err := redis.Instances(ctx)
		if err != nil {
			return err
		}

		if err := removeByInfo(ctx, items); err != nil {
			return err
		}
	}

	// it could be the case that user sent instance id instead of type
	// so we try to remove it
	// TODO check if database is in detached mode and warn user
	if len(args) == 1 && !itsDBType(args[0]) {
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

		// check if its label then remove by label
		effectd, err := removeByLabel(ctx, args[0])
		if err != nil {
			return err
		}

		if effectd > 0 {
			return nil
		}

		// remove by id
		if err := container.TerminateByID(ctx, args[0]); err != nil {
			return err
		}
	}

	return nil
}

func removeByInfo(ctx context.Context, dbs []database.Info) error {
	for _, i := range dbs {
		if err := container.TerminateByID(ctx, i.ID); err != nil {
			return err
		}
	}
	return nil
}

func removeByLabel(ctx context.Context, label string) (int, error) {
	// remove by label
	items, err := container.List(ctx, map[string]string{container.LabelCustom: label})
	if err != nil {
		return 0, err
	}

	var effected int = 0

	for _, i := range items {
		if err := container.TerminateByID(ctx, i.ID); err != nil {
			return effected, err
		}

		effected++
	}

	return effected, nil
}

func itsDBType(a string) bool {
	return utils.OneOf(a, "pg", "postgres", "rs", "redis")
}
