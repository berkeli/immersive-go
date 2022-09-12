// cmd/root.go
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	ErrInvalidPath           = "Invalid path provided"
	ErrNotADirectory         = "Provided path is not a directory"
	ErrCouldNotReadDirectory = "Could not read directory"
)

var (
	Delimeter string = "\n"
	flagM     bool
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
	RunE:  GolsCmdFunc,
}

func init() {
	golsCmd.Flags().BoolVarP(&flagM, "m", "m", false, "Use this flag if you would like the output to be comma separated instead of new lines.")
}

func GolsCmdFunc(cmd *cobra.Command, args []string) error {

	var (
		path string
		err  error
	)

	if flagM {
		Delimeter = ", "
	}

	switch len(args) {
	case 0:
		path, err = os.Getwd()
		if err != nil {
			return err
		}
		out, err := ListFolderContents(path)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), out)
	case 1:
		path = args[0]
		out, err := ListFolderContents(path)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), out)
	default:
		for i, path := range args {
			out, err := ListFolderContents(path)
			endOfLine := ""
			if i < len(args)-1 {
				endOfLine = "\n"
			}

			if err != nil {
				fmt.Println(path + ": " + err.Error() + endOfLine)
				continue
			}
			fmt.Fprint(cmd.OutOrStdout(), path+": \n")
			fmt.Fprintln(cmd.OutOrStdout(), out+endOfLine)
		}

	}

	if err != nil {
		return err
	}

	return nil
}

func ListFolderContents(path string) (string, error) {
	file, err := os.Stat(path)
	if err != nil {
		return "", errors.New(ErrInvalidPath)
	}

	if !file.IsDir() {
		return "", errors.New(ErrNotADirectory)
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return "", errors.New(ErrCouldNotReadDirectory)
	}
	out := ""
	for i, file := range files {
		out += file.Name()

		if i < len(files)-1 {
			out += Delimeter
		}
	}

	return out, nil
}
