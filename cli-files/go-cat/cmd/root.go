/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"errors"
	"fmt"
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
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return ErrDir
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Fprintln(Writer, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
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
