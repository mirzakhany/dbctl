/*
Copyright © 2022 Mohsen Mirzakhani <mohsen.mkh88@gmail.com>
*/
package main

import (
	"fmt"
	"os"

	"github.com/mirzakhany/dbctl/cmd"
	"github.com/mirzakhany/dbctl/cmd/start"
	"github.com/mirzakhany/dbctl/cmd/testing"
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

	// testing is able to run multiple commands includes starting the dbctl api server
	root.AddCommand(testing.GetStartTestingCmd(root))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
