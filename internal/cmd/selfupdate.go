package cmd

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/mirzakhany/dbctl/internal/selfupdate"
	"github.com/mirzakhany/dbctl/internal/utils"
	"github.com/spf13/cobra"
)

func GetSelfUpdateCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Update dbctl to its latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doSelfUpdate(version)
		},
	}
}

func doSelfUpdate(version string) error {
	ctx := utils.ContextWithOsSignal()

	updater := selfupdate.New("mirzakhany", "dbctl", "dbctl")
	latest, err := updater.LatestVersion(ctx)
	if err != nil {
		return err
	}

	var v *selfupdate.Version
	if version == "snapshot" {
		vv, err := selfupdate.ParseVersion("0.0.1")
		if err != nil {
			return err
		}
		v = vv
	} else {
		vr := strings.Split(version, "-")[0]
		vv, err := selfupdate.ParseVersion(vr[1:])
		if err != nil {
			return err
		}
		v = vv
	}

	if v == nil {
		return errors.New("parse version failed")
	}

	if latest.Greater(v) {
		log.Printf("Current version (%s) is the latest\n", v)
		return nil
	}

	log.Print("Do you want to update to ", latest, "? (y/n): ")
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || (input != "y\n" && input != "n\n") {
		log.Println("Invalid input")
		return err
	}
	if input == "n\n" {
		return nil
	}

	if err := updater.Update(ctx); err != nil {
		log.Println("Error occurred while updating binary:", err)
		return err
	}
	log.Println("Successfully updated to version", latest)
	return nil
}
