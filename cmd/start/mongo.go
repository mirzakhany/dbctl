package start

import (
	"fmt"
	"io"

	"github.com/mirzakhany/dbctl/internal/database/mongodb"
	"github.com/mirzakhany/dbctl/internal/utils"
	"github.com/spf13/cobra"
)

// GetMongoDBCmd represents the mongodb command
func GetMongoDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"mongo", "mdb"},
		Use:     "mongodb",
		Short:   "Run a MongoDB instance",
		RunE:    runMongoDB,
	}

	cmd.Flags().Uint32P("port", "p", mongodb.DefaultPort, "MongoDB default port")
	cmd.Flags().StringP("user", "u", mongodb.DefaultUser, "Database username")
	cmd.Flags().String("pass", mongodb.DefaultPass, "Database password")
	cmd.Flags().StringP("name", "n", mongodb.DefaultName, "Database name")
	cmd.Flags().StringP("version", "v", "", "Database version, default 7.0")
	cmd.Flags().StringP("migrations", "m", "", "Path to migration files, will be applied if provided")
	cmd.Flags().StringP("fixtures", "f", "", "Path to fixture files, it can be a file or directory. Files in directory will be sorted by name before applying.")

	return cmd
}

func runMongoDB(cmd *cobra.Command, _ []string) error {
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

	mongoVersion, err := cmd.Flags().GetString("version")
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

	db, err := mongodb.New(
		mongodb.WithHost(user, pass, name, port),
		mongodb.WithVersion(mongoVersion),
		mongodb.WithLogger(io.Discard),
		mongodb.WithMigrations(migrationsPath),
		mongodb.WithFixtures(fixturesPath),
		mongodb.WithUI(withUI),
		mongodb.WithLabel(label),
	)
	if err != nil {
		return err
	}

	return db.Start(utils.ContextWithOsSignal(), detach)
}
