package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmd1    = &cobra.Command{}
	cmd2    = &cobra.Command{}
	rootCmd = &cobra.Command{}
)

func main() {
	var d string
	var a string

	//
	cmd1 = &cobra.Command{
		Use: "comm1",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("1111")
			fmt.Println("dval is", d)
		},
	}
	cmd1.Flags().StringVarP(&d, "domain", "d", "", "xxxxxxxxxxxxxxx")

	//
	cmd2 = &cobra.Command{
		Use: "comm2",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("2222222")
			fmt.Println("aval is", a)
		},
	}
	cmd2.Flags().StringVarP(&a, "dirscan", "a", "", "xxxxxxxxxxxxxxx")

	//
	rootCmd = &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			for _, arg := range args {
				switch arg {
				case "comm1":
					fmt.Println("chose root and comm1")
					cmd1.DisableFlagParsing = true
					cmd1.Execute()
				case "comm2":
					fmt.Println("chose root and comm2")

					cmd2.Execute()
				}
			}
		},
	}

	for _, c := range []*cobra.Command{cmd1, cmd2} {
		rootCmd.Flags().AddFlagSet(c.Flags())
		c.DisableFlagParsing = true
	}
	rootCmd.Execute()
}
