/*
Copyright © 2022 Mohsen Mirzakhani <mohsen.mkh88@gmail.com>
*/
package main

import (
	"fmt"
	"os"

	"github.com/mirzakhany/dbctl/internal/cmd"
)

// version will be populated by the build script with the sha of the last git commit.
var version = "snapshot"

func main() {
	root := cmd.GetRootCmd(version)
	root.SetVersionTemplate(fmt.Sprintf("dbctl version %s\n", version))

	root.AddCommand(cmd.GetStartCmd())
	root.AddCommand(cmd.GetUpCmd())
	root.AddCommand(cmd.GetSelfUpdateCmd(version))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
