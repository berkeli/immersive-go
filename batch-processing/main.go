package main

import (
	"flag"
	"io"
	"log"
	"os"
	"sync"

	"gopkg.in/gographics/imagick.v2/imagick"
)

func main() {
	// Accept --input and --output arguments for the images
	inputFilepath := flag.String("input", "", "A path to a CSV file containing image URLs to be processed")
	outputFilepath := flag.String("output", "", "A path to where the CSV file with processed image URLs should be written")
	failedOutputFilepath := flag.String("output-failed", "", "A path to where the CSV file with failed image URLs should be written (optional)")
	flag.Parse()

	// Ensure that both flags were set
	if *inputFilepath == "" || *outputFilepath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Set up imagemagick
	imagick.Initialize()
	defer imagick.Terminate()

	// Log what we're going to do
	log.Printf("processing: %q to %q\n", *inputFilepath, *outputFilepath)

	// Build a Converter struct that will use imagick
	c := &Converter{
		cmd: imagick.ConvertImageCommand,
	}

	reader := ReadCSV(inputFilepath)
	result := make(chan *Output)

	wg := &sync.WaitGroup{}

	go ResultTOCSV(result, *outputFilepath, *failedOutputFilepath)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result <- &Output{url: row[0], err: err}
			continue
		}
		url := row[0]
		wg.Add(1)

		go ProcessRow(url, *c, result, wg)
	}
	wg.Wait()
	close(result)
	// Log what we did
	log.Printf("processed: %q to %q\n", *inputFilepath, *outputFilepath)

	if *failedOutputFilepath != "" {
		log.Printf("failed: %q\n", *failedOutputFilepath)
	}
}
