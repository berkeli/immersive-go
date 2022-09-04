/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
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

		path := args[0]

		argStat, err := os.Stat(path)
		if err != nil {
			return err
		}

		if argStat.IsDir() {
			return ErrDir
		}

		data, err := os.ReadFile(args[0])

		if err != nil {
			return err
		}

		fmt.Println(string(data))

		return nil

	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}