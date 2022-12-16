/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	ErrNoPath = errors.New("Please provide a file name or path to a file")
	ErrDir    = errors.New("Please provide a file name or path to a file, cannot read a directory")
)

var rootCmd = &cobra.Command{
	Use:   "go-cat",
	Short: "go-cat allows you to see contents of a file",
	Long:  `go-cat allows you to see contents of a file, just like the cat command`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return ErrNoPath
		}

		for _, path := range args {
			err := PrintFileContents(cmd.OutOrStdout(), path)
			if err != nil {
				return err
			}
		}
		return nil

	},
}

func PrintFileContents(Writer io.Writer, path string) error {

	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return ErrDir
	}

	file, err := os.Open(path)
	_, err = io.Copy(Writer, file)

	if err != nil {
		return err
	}

	return nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
