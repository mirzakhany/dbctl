package cmd

import (
	"fmt"
	"io"

	"github.com/mirzakhany/dbctl/internal/pg"
	"github.com/spf13/cobra"
)

// GetPgCmd represents the pg command
func GetPgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"pg"},
		Use:     "postgres",
		Short:   "Run a postgres instance",
		RunE:    runPostgres,
	}

	cmd.Flags().Uint32P("port", "p", 15432, "postgres default port")
	cmd.Flags().StringP("user", "u", "postgres", "Database username")
	cmd.Flags().String("pass", "postgres", "Database password")
	cmd.Flags().StringP("name", "n", "postgres", "Database name")
	cmd.Flags().StringP("version", "v", "", "Database version, default for native 14.3.0 and 14.3.2 for docker engine")
	cmd.Flags().StringP("migrations", "m", "", "Path to migration files, will be applied if provided")
	cmd.Flags().StringP("fixtures", "f", "", "Path to fixture files, its can be a file or directory.files in directory will be sorted by name before applying.")

	return cmd
}

func runPostgres(cmd *cobra.Command, args []string) error {
	port, err := cmd.Flags().GetUint32("port")
	if err != nil {
		return fmt.Errorf("invalid port args, %w", err)
	}

	detach, err := cmd.Flags().GetBool("detach")
	if err != nil {
		return fmt.Errorf("invalid detach args, %w", err)
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

	fixturesPath, err := cmd.Flags().GetString("fixtures")
	if err != nil {
		return fmt.Errorf("invalid fixtures args, %w", err)
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
		pg.WithFixtures(fixturesPath),
		pg.WithDetach(detach),
	)
	if err != nil {
		return err
	}

	if err := db.Start(contextWithOsSignal()); err != nil {
		return err
	}

	return nil
}
