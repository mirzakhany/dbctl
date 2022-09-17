package cmd

import (
	"fmt"
	"io"

	"github.com/mirzakhany/dbctl/internal/redis"

	"github.com/spf13/cobra"
)

// GetRedisCmd represents the pg command
func GetRedisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"rs"},
		Use:     "redis",
		Short:   "Run a redis instance",
		RunE:    runRedis,
	}

	cmd.Flags().Uint32P("port", "p", 16379, "Redis default port")
	cmd.Flags().Int("db", 0, "Redis db index")
	cmd.Flags().StringP("user", "u", "", "Database username")
	cmd.Flags().String("pass", "", "Database password")
	cmd.Flags().StringP("version", "v", "", "Database version, default 7.0.4 for docker engine")

	return cmd
}

func runRedis(cmd *cobra.Command, args []string) error {
	port, err := cmd.Flags().GetUint32("port")
	if err != nil {
		return fmt.Errorf("invalid port args, %w", err)
	}

	dbIndex, err := cmd.Flags().GetInt("db")
	if err != nil {
		return fmt.Errorf("invalid db index args, %w", err)
	}

	detach, err := cmd.Flags().GetBool("detach")
	if err != nil {
		return fmt.Errorf("invalid detach args, %w", err)
	}

	pass, err := cmd.Flags().GetString("pass")
	if err != nil {
		return fmt.Errorf("invalid pass args, %w", err)
	}

	user, err := cmd.Flags().GetString("user")
	if err != nil {
		return fmt.Errorf("invalid user args, %w", err)
	}

	redisVersion, err := cmd.Flags().GetString("version")
	if err != nil {
		return fmt.Errorf("invalid version args, %w", err)
	}

	db, err := redis.New(
		redis.WithHost(user, pass, dbIndex, port),
		redis.WithVersion(redisVersion),
		redis.WithLogger(io.Discard),
		redis.WithDetach(detach),
	)
	if err != nil {
		return err
	}

	if err := db.Start(); err != nil {
		return err
	}

	return nil
}
