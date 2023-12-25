package describe

import (
	"fmt"
	"os"

	"github.com/mirzakhany/dbctl/internal/container"
	pg "github.com/mirzakhany/dbctl/internal/database/postgres"
	"github.com/mirzakhany/dbctl/internal/utils"
	"github.com/spf13/cobra"
)

// GetDescribePgCmd represents the describe pg command
func GetDescribePgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"pg"},
		Use:     "postgres",
		Short:   "Describe a postgres database instance",
		RunE:    describePostgres,
	}

	cmd.Flags().StringP("version", "v", "", "Database version, default 14.3.2")
	cmd.Flags().StringP("migrations", "m", "", "Path to migration files")
	cmd.Flags().StringP("schema", "s", "public", "Schema name to describe, default public")
	cmd.Flags().StringP("table", "t", "*", "Table name to describe, default * (all tables)")

	return cmd
}

func describePostgres(cmd *cobra.Command, _ []string) error {
	version, err := cmd.Flags().GetString("version")
	if err != nil {
		return fmt.Errorf("invalid version args, %w", err)
	}

	migrations, err := cmd.Flags().GetString("migrations")
	if err != nil {
		return fmt.Errorf("invalid migrations args, %w", err)
	}

	schema, err := cmd.Flags().GetString("schema")
	if err != nil {
		return fmt.Errorf("invalid schema args, %w", err)
	}

	table, err := cmd.Flags().GetString("table")
	if err != nil {
		return fmt.Errorf("invalid table args, %w", err)
	}

	ctx := utils.ContextWithOsSignal()
	// get a free port
	portNumber := utils.GetAvailablePort()
	db, err := pg.New(
		pg.WithHost("postgres", "postgres", "postgres", uint32(portNumber)),
		pg.WithVersion(version),
		pg.WithLogger(os.Stdout),
		pg.WithMigrations(migrations),
	)
	if err != nil {
		return err
	}

	if err := db.Start(ctx, true); err != nil {
		return err
	}

	url := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	psqlCommand := []string{"sh", "-c", fmt.Sprintf("psql %q -c \"\\d %s.%s\"", url, schema, table)}
	result, err := container.RunExec(ctx, db.ContainerID(), psqlCommand)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s", result)

	return db.Stop(ctx)
}
