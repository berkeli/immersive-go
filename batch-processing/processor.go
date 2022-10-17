package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/service/s3"
)

// This function consumes output from the channel for results
// and writes them to a CSV file

func ResultToCSV(rows <-chan *Output, outputFilepath, failedOutputFilepath string) {

	outputHeader := []string{"url", "input", "output", "s3url"}
	failedOutputHeader := []string{"url", "input", "output", "s3url", "error"}

	f, err := os.Create(outputFilepath)
	if err != nil {
		log.Fatalf("error creating output file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write(outputHeader)
	var failedW *csv.Writer
	defer w.Flush()
	defer failedW.Flush()
	// Create csv for failed output if param is set
	if failedOutputFilepath != "" {
		failedF, err := os.Create(failedOutputFilepath)
		if err != nil {
			log.Fatalf("error creating failed output file: %v", err)
		}
		defer failedF.Close()

		failedW = csv.NewWriter(failedF)
		failedW.Write(failedOutputHeader)
	}

	for row := range rows {
		if row.err != nil && failedOutputFilepath != "" {
			failedW.Write([]string{row.url, row.input, row.output, row.s3url, row.err.Error()})
			continue
		}

		w.Write([]string{row.url, row.input, row.output, row.s3url})
	}
}

// Main function that processes each row in the csv

func ProcessRow(url string, c Converter, r chan *Output, wg *sync.WaitGroup, a *AWSConfig) {
	defer wg.Done()
	fileExt := "jpg"
	fileName := extractFilename(url)
	inputPath := fmt.Sprintf("/outputs/%s.%s", fileName, fileExt)
	outputPath := fmt.Sprintf("/outputs/%s-%s.%s", fileName, "converted", fileExt)

	fmt.Println("Downloading image from URL: ", url)
	err := downloadFile(url, inputPath)

	if err != nil {
		r <- &Output{url: url, err: err}
		return
	}

	// Do the conversion!
	err = c.Grayscale(inputPath, outputPath)

	if err != nil {
		r <- &Output{url: url, input: inputPath, output: outputPath, err: err}
		return
	}

	file, err := os.Open(outputPath)

	if err != nil {
		r <- &Output{url: url, input: inputPath, output: outputPath, err: err}
		return
	}

	// Upload to S3
	key := fmt.Sprintf("%s.%s", fileName, fileExt)

	_, err = a.s3.PutObject(&s3.PutObjectInput{
		Bucket: &a.s3bucket,
		Key:    &key,
		Body:   file,
	})

	if err != nil {
		r <- &Output{url: url, input: inputPath, output: outputPath, err: err}
		return
	}

	r <- &Output{
		url:    url,
		input:  inputPath,
		output: outputPath,
		s3url:  fmt.Sprintf("https://%s.s3.amazonaws.com/%s", a.s3bucket, key),
	}

}
