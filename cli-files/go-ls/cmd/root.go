// cmd/root.go
package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	if err := golsCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var golsCmd = &cobra.Command{
	Use:   "go-ls",
	Short: "go-ls is an alternative to ls, allows you to list files in a given directory",
	Long:  "go-ls is an alternative to ls, allows you to list files in a given directory. By default it lists files in the current working directory. You can also specify a directory to list files in.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}

		files, err := ioutil.ReadDir(path)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			fmt.Println(file.Name())
		}

		return nil
	},
}
