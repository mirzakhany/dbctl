/*
Copyright © 2022 Mohsen Mirzakhani <mohsen.mkh88@gmail.com>
*/
package main

import (
	"fmt"
	"os"

	"github.com/mirzakhany/dbctl/internal/cmd"
	"github.com/mirzakhany/dbctl/internal/cmd/start"
)

// version will be populated by the build script with the sha of the last git commit.
var version = "snapshot"

func main() {
	root := cmd.GetRootCmd(version)
	root.SetVersionTemplate(fmt.Sprintf("dbctl version %s\n", version))

	root.AddCommand(start.GetStartCmd())
	root.AddCommand(cmd.GetStopCmd())
	root.AddCommand(cmd.GetListCmd())
	root.AddCommand(cmd.GetSelfUpdateCmd(version))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
