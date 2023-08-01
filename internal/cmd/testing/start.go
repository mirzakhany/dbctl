package testing

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

//
// dbctl testing \
//			pg -p 5435 -m ./migrations -f ./fixtures \
//   		rs -p 7654
//

// GetStartTestingCmd represents the start testing command
func GetStartTestingCmd() *cobra.Command {
	redisCmd := rsCommand()
	postgresCmd := pgCommand()

	cmd := &cobra.Command{
		Use:                "testing",
		Short:              "Start dbctl server for unit testing",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				switch arg {
				case "postgres", "pg":
					postgresCmd.DisableFlagParsing = true
					if err := postgresCmd.Execute(); err != nil {
						log.Fatalln(err)
					}
				case "redis", "rs":
					return redisCmd.Execute()
				}
			}

			return nil
		},
	}

	cmd.Flags().Uint32P("port", "p", 15432, "tcp port")
	cmd.Flags().StringP("migrations", "m", "", "Path to migration files, will be applied if provided")
	cmd.Flags().StringP("fixtures", "f", "", "Path to fixture files, its can be a file or directory.files in directory will be sorted by name before applying.")

	// for _, c := range []*cobra.Command{redisCmd, postgresCmd} {
	// 	c.ResetFlags().AddFlagSet(cmd.Flags())
	// }

	return cmd
}

func pgCommand() *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"pg"},
		Use:     "postgres",
		Short:   "Run a postgres instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			migrationsPath, err := cmd.Flags().GetString("migrations")
			if err != nil {
				return fmt.Errorf("invalid migrations args, %w", err)
			}

			fixturesPath, err := cmd.Flags().GetString("fixtures")
			if err != nil {
				return fmt.Errorf("invalid fixtures args, %w", err)
			}

			port, err := cmd.Flags().GetUint32("port")
			if err != nil {
				return fmt.Errorf("invalid port args, %w", err)
			}

			fmt.Println("pg", migrationsPath, fixturesPath, port)

			return nil
		},
	}

	return cmd
}

func rsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"rs"},
		Use:     "redis",
		Short:   "Run a redis instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := cmd.Flags().GetUint32("port")
			if err != nil {
				return fmt.Errorf("invalid port args, %w", err)
			}
			fmt.Println("rs", port)
			return nil
		},
	}

	return cmd
}
