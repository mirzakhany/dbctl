package start

import (
	"fmt"
	"io"

	pg "github.com/mirzakhany/dbctl/internal/database/postgres"
	"github.com/mirzakhany/dbctl/internal/utils"
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

	cmd.Flags().Uint32P("port", "p", pg.DefaultPort, "postgres default port")
	cmd.Flags().StringP("user", "u", pg.DefaultUser, "Database username")
	cmd.Flags().String("pass", pg.DefaultPass, "Database password")
	cmd.Flags().StringP("name", "n", pg.DefaultName, "Database name")
	cmd.Flags().StringP("version", "v", "", "Database version, default 14.3.2")
	cmd.Flags().StringP("migrations", "m", "", "Path to migration files, will be applied if provided")
	cmd.Flags().StringP("fixtures", "f", "", "Path to fixture files, its can be a file or directory.files in directory will be sorted by name before applying.")

	return cmd
}

func runPostgres(cmd *cobra.Command, _ []string) error {
	port, err := cmd.Flags().GetUint32("port")
	if err != nil {
		return fmt.Errorf("invalid port args, %w", err)
	}

	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return fmt.Errorf("invalid label args, %w", err)
	}

	detach, err := cmd.Flags().GetBool("detach")
	if err != nil {
		return fmt.Errorf("invalid detach args, %w", err)
	}

	withUI, err := cmd.Flags().GetBool("ui")
	if err != nil {
		return fmt.Errorf("invalid ui args, %w", err)
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

	db, err := pg.New(
		pg.WithHost(user, pass, name, port),
		pg.WithVersion(pgVersion),
		pg.WithLogger(io.Discard),
		pg.WithMigrations(migrationsPath),
		pg.WithFixtures(fixturesPath),
		pg.WithUI(withUI),
		pg.WithLabel(label),
	)
	if err != nil {
		return err
	}

	return db.Start(utils.ContextWithOsSignal(), detach)
}
