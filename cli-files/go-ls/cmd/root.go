// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

const (
	ErrInvalidPath           = "Invalid path provided"
	ErrNotADirectory         = "Provided path is not a directory"
	ErrCouldNotReadDirectory = "Could not read directory"
)

func Execute() {
	cobra.CheckErr(golsCmd.Execute())
}

var golsCmd = &cobra.Command{
	Use:   "go-ls",
	Short: "go-ls is an alternative to ls, allows you to list files in a given directory",
	Long:  "go-ls is an alternative to ls, allows you to list files in a given directory. By default it lists files in the current working directory. You can also specify a directory to list files in.",
	RunE:  GolsCmdFunc,
}

func init() {
	GoLsCmdFlags(golsCmd)
}

func GolsCmdFunc(cmd *cobra.Command, args []string) error {
	var (
		ErrCollection *multierror.Error
		Delimeter     string   = "\n"
		paths         []string = args
	)

	Writer := cmd.OutOrStdout()

	m, err := cmd.Flags().GetBool("m")
	if err != nil {
		return err
	}

	if m {
		Delimeter = ", "
	}

	if len(paths) == 0 {
		path, err := os.Getwd()
		if err != nil {
			return err
		}
		paths = append(paths, path)
	}

	for i, path := range paths {
		out, err := ListFolderContents(path, Delimeter)

		if err != nil {
			ErrCollection = multierror.Append(ErrCollection, err)
			continue
		}

		endOfLine := ""
		if i < len(args)-1 && len(args) > 1 {
			endOfLine = "\n"
		}

		// Display the name of the folder if there are more than 1 paths passed in
		if len(paths) > 1 {
			fmt.Fprintln(Writer, path+":")
		}

		fmt.Fprintln(Writer, out+endOfLine)
	}

	return ErrCollection.ErrorOrNil()
}

func ListFolderContents(path, delimeter string) (string, error) {
	file, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", ErrInvalidPath, err)
	}

	if !file.IsDir() {
		return path, nil
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", ErrCouldNotReadDirectory, err)
	}
	out := ""
	for i, file := range files {
		out += file.Name()

		if i < len(files)-1 {
			out += delimeter
		}
	}

	return out, nil
}

func GoLsCmdFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("m", "m", false, "Use this flag if you would like the output to be comma separated instead of new lines.")
}
