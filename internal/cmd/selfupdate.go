package cmd

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

const repo = "mirzakhany/dbctl"

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
	latest, found, err := selfupdate.DetectLatest(repo)
	if err != nil {
		log.Println("Error occurred while detecting version:", err)
		return err
	}

	vr := strings.Split(version, "-")[0]
	v := semver.MustParse(vr[1:])
	if !found || latest.Version.LTE(v) {
		log.Println("Current version is the latest")
		return err
	}

	log.Print("Do you want to update to", latest.Version, "? (y/n): ")
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || (input != "y\n" && input != "n\n") {
		log.Println("Invalid input")
		return err
	}
	if input == "n\n" {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		log.Println("Could not locate executable path")
		return err
	}
	if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
		log.Println("Error occurred while updating binary:", err)
		return err
	}
	log.Println("Successfully updated to version", latest.Version)
	return nil
}
