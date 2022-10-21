package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
)

func Read(csvReader *csv.Reader, out chan *Out) {
	// Read the input CSV file
	// For each line, send the URL to the download channel
	defer close(out)

	csvReader.FieldsPerRecord = 1

	for {
		row, err := csvReader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			out <- &Out{Err: err}
			continue
		}
		out <- &Out{Url: row[0]}
	}
}

func Download(in <-chan *Out, out chan *Out, wg *sync.WaitGroup) {
	// Download the image from the URL
	// Send the image to the convert channel
	defer wg.Done()
	for {
		row, ok := <-in
		if !ok {
			break
		}
		if row.Err != nil {
			out <- row
			continue
		}
		body, ext, err := DownloadFileFromUrl(row.Url)
		if err != nil {
			row.Err = err
			out <- row
			continue
		}

		//Create an empty file
		fileName := extractFilename(row.Url)
		inputPath := fmt.Sprintf("/outputs/%s.%s", fileName, ext)
		outputPath := fmt.Sprintf("/outputs/%s-converted.%s", fileName, ext)

		file, err := os.Create(inputPath)
		if err != nil {
			out <- &Out{
				Url: row.Url,
				Err: err,
			}
			continue
		}
		defer file.Close()

		//Write bytes to the file
		_, err = io.Copy(file, body)
		if err != nil {
			out <- &Out{
				Url: row.Url,
				Err: err,
			}
			continue
		}

		out <- &Out{
			Url:    row.Url,
			Input:  inputPath,
			Output: outputPath,
		}
	}
}

func Convert(in <-chan *Out, out chan *Out, wg *sync.WaitGroup, c *Converter) {
	// Convert the image
	// Send the image to the upload channel
	defer wg.Done()
	for {
		row, ok := <-in
		if !ok {
			break
		}
		if row.Err != nil {
			out <- &Out{
				Url:    row.Url,
				Err:    row.Err,
				Input:  row.Input,
				Output: "",
			}
			continue
		}
		err := c.Grayscale(row.Input, row.Output)
		if err != nil {
			out <- &Out{
				Url:    row.Url,
				Err:    err,
				Input:  row.Input,
				Output: "",
			}
			continue
		}
		out <- row
	}
}

func Upload(in <-chan *Out, out chan *Out, wg *sync.WaitGroup, a *AWSConfig) {
	// Upload the image to S3
	// Send the image to the result channel
	defer wg.Done()
	for {
		row, ok := <-in
		if !ok {
			break
		}

		if row.Err != nil {
			out <- row
			continue
		}

		file, err := os.Open(row.Output)

		if err != nil {
			row.Err = err
			out <- row
			continue
		}

		// Upload to S3
		key := strings.Replace(row.Output, "/outputs/", "", 1)

		_, err = a.s3.PutObject(&s3.PutObjectInput{
			Bucket: &a.s3bucket,
			Key:    &key,
			Body:   file,
		})

		if err != nil {
			row.Err = err
			out <- row
			continue
		}

		row.S3url = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", a.s3bucket, key)

		out <- row
	}
}

func ResultToCSV(rows <-chan *Out, c *Config, done chan bool) {

	outputHeader := []string{"url", "input", "output", "s3url"}
	failedOutputHeader := []string{"url", "input", "output", "s3url", "error"}

	f, err := os.Create(c.OutputFilepath)
	if err != nil {
		log.Fatalf("error creating output file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write(outputHeader)
	var failedW *csv.Writer

	// Create csv for failed output if param is set
	if c.FailedOutputFilepath != "" {
		failedF, err := os.Create(c.FailedOutputFilepath)
		if err != nil {
			log.Fatalf("error creating failed output file: %v", err)
		}
		defer failedF.Close()

		failedW = csv.NewWriter(failedF)
		failedW.Write(failedOutputHeader)
	}

	for {
		row, ok := <-rows

		if !ok {
			break
		}

		if row.Err != nil && c.FailedOutputFilepath != "" {
			failedW.Write([]string{row.Url, row.Input, row.Output, row.S3url, row.Err.Error()})
			continue
		}

		w.Write([]string{row.Url, row.Input, row.Output, row.S3url})
	}
	w.Flush()
	failedW.Flush()
	done <- true
}

func Do(config *Config) error {

	start := time.Now()
	// Create the channels
	done := make(chan bool)
	readOut := make(chan *Out)
	downloadOut := make(chan *Out)
	convertOut := make(chan *Out)
	uploadOut := make(chan *Out)

	// Start CSV writer
	go ResultToCSV(uploadOut, config, done)

	// Start the uploaders
	uploadWg := &sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		uploadWg.Add(1)
		go Upload(convertOut, uploadOut, uploadWg, config.Aws)
	}

	//start converter workers
	convertWg := &sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		convertWg.Add(1)
		go Convert(downloadOut, convertOut, convertWg, config.Converter)
	}

	//start download workers
	downWg := &sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		downWg.Add(1)
		go Download(readOut, downloadOut, downWg)
	}

	// Start the reader
	csvReader, err := OpenCSVFile(config.InputFilepath)

	if err != nil {
		return err
	}
	go Read(csvReader, readOut)

	downWg.Wait()
	log.Println("Finished downloading images")
	close(downloadOut)

	convertWg.Wait()
	log.Println("Finished converting files")
	close(convertOut)

	uploadWg.Wait()
	log.Println("Finished uploading files")
	close(uploadOut)

	elapsed := time.Since(start)

	log.Printf("Finished in %s", elapsed)

	<-done

	return nil
}
