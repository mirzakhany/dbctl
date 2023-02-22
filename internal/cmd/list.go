package cmd

import (
	"github.com/fatih/color"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
	"github.com/mirzakhany/dbctl/internal/utils"
	"github.com/rodaine/table"
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

func runList(cmd *cobra.Command, args []string) error {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("ID", "Name", "Type")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	ctx := utils.ContextWithOsSignal()
	containers, err := container.List(ctx, nil)
	if err != nil {
		return err
	}

	for _, c := range containers {
		tbl.AddRow(c.ID[:12], c.Name, c.Labels[database.LabelType])
	}

	tbl.Print()
	return nil
}
