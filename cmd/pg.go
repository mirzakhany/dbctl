package cmd

import (
	"fmt"
	"io"

	"github.com/mirzakhany/dbctl/pg"

	"github.com/spf13/cobra"
)

// pgCmd represents the pg command
var pgCmd = &cobra.Command{
	Use:   "pg",
	Short: "Run a postgres instance",
	RunE:  runPostgres,
}

func init() {
	startCmd.AddCommand(pgCmd)

	pgCmd.Flags().StringP("user", "u", "postgres", "Username, default is postgres")
	pgCmd.Flags().String("pass", "postgres", "Password, default is postgres")
	pgCmd.Flags().StringP("name", "n", "postgres", "Database name, default is postgres")
	pgCmd.Flags().StringP("version", "v", "", "Database version, default for native 14.3.0 and 14.3.2 for docker engine")
	pgCmd.Flags().StringP("migrations", "m", "", "Relative path to migration files, will be applied if provided")
}

func runPostgres(cmd *cobra.Command, args []string) error {
	port, err := cmd.Flags().GetUint32("port")
	if err != nil {
		return fmt.Errorf("invalid port args, %w", err)
	}

	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return fmt.Errorf("invalid name args, %w", err)
	}

	pass, err := cmd.Flags().GetString("pass")
	if err != nil {
		return fmt.Errorf("invalid pass args, %w", err)
	}

	user, err := cmd.Flags().GetString("user")
	if err != nil {
		return fmt.Errorf("invalid user args, %w", err)
	}

	pgVersion, err := cmd.Flags().GetString("version")
	if err != nil {
		return fmt.Errorf("invalid version args, %w", err)
	}

	migrationsPath, err := cmd.Flags().GetString("migrations")
	if err != nil {
		return fmt.Errorf("invalid migrations args, %w", err)
	}

	useDockerEngine, err := cmd.Flags().GetBool("use-docker")
	if err != nil {
		return fmt.Errorf("invalid use-docker args, %w", err)
	}

	db, err := pg.New(
		pg.WithDockerEngine(useDockerEngine),
		pg.WithHost(user, pass, name, port),
		pg.WithVersion(pgVersion),
		pg.WithLogger(io.Discard),
		pg.WithMigrations(migrationsPath),
	)
	if err != nil {
		return err
	}

	if err := db.Start(); err != nil {
		return err
	}
	return nil
}
