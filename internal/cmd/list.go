package cmd

import (
	"os"

	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
	"github.com/mirzakhany/dbctl/internal/table"
	"github.com/mirzakhany/dbctl/internal/utils"
	"github.com/spf13/cobra"
)

// GetListCmd represents the list command
func GetListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Aliases: []string{"ls"},
		Use:     "list",
		Short:   "list the running databases managed by dbctl",
		RunE:    runList,
	}
	return cmd
}

func runList(_ *cobra.Command, _ []string) error {
	ctx := utils.ContextWithOsSignal()
	containers, err := container.List(ctx, nil)
	if err != nil {
		return err
	}

	t := table.New(os.Stdout)
	t.AddRow("ID", "Name", "Type")
	for _, c := range containers {
		t.AddRow(c.ID[:12], c.Name, c.Labels[database.LabelType])
	}

	t.Print()
	return nil
}
