package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/service/s3"
)

func ResultTOCSV(rows <-chan *Output, outputFilepath, failedOutputFilepath string) {

	f, err := os.Create(outputFilepath)
	if err != nil {
		log.Fatalf("error creating output file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"url", "input", "output", "s3url"})
	var failedW *csv.Writer
	defer w.Flush()

	if failedOutputFilepath != "" {
		failedF, err := os.Create(failedOutputFilepath)
		if err != nil {
			log.Fatalf("error creating failed output file: %v", err)
		}
		defer failedF.Close()

		failedW = csv.NewWriter(failedF)
		failedW.Write([]string{"url", "input", "output", "s3url", "error"})
		defer failedW.Flush()
	}

	for row := range rows {
		if row.err != nil && failedOutputFilepath != "" {
			failedW.Write([]string{row.url, row.input, row.output, row.s3url, row.err.Error()})
			continue
		}

		w.Write([]string{row.url, row.input, row.output, row.s3url})
	}
}

func ProcessRow(url string, c Converter, r chan *Output, wg *sync.WaitGroup, s3Client *s3.S3, awsConfig *AWSConfig) {
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

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: &awsConfig.s3bucket,
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
		s3url:  fmt.Sprintf("https://%s.s3.amazonaws.com/%s", awsConfig.s3bucket, key),
	}

}