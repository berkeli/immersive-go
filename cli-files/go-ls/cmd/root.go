// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Mode bool

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
	RunE:  GolsCmdFunc,
}

func init() {
	golsCmd.PersistentFlags().BoolVarP(&Mode, "mode", "m", false, "Selects the mode of operation")
}

func GolsCmdFunc(cmd *cobra.Command, args []string) error {

	var path string
	var err error

	if len(args) > 0 {
		path = args[0]
	} else {
		path, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	argStat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !argStat.IsDir() {
		fmt.Println(path)
		return nil
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for i, file := range files {
		out := file.Name()

		if Mode && i < len(files)-1 {
			out += ", "
		} else {
			out += "\n"
		}

		fmt.Printf(out)
	}

	return nil
}
