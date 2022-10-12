package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"errors"
)

const (
	InvalidCSVFormat   = "CSV file must have a header row with 'url' as the first column"
	CouldNotFetchImage = "Received status %d when trying to download image"
)

func ReadCSV(inputFilepath *string) *csv.Reader {
	f, err := os.Open(*inputFilepath)
	if err != nil {
		log.Fatalf("error reading input file: %v", err)
	}

	csvReader := csv.NewReader(f)
	csvReader.FieldsPerRecord = 1

	header, err := csvReader.Read()

	if err != nil {
		log.Fatalf("Error reading CSV: %v", err)
	}

	if strings.ToLower(header[0]) != "url" {
		log.Fatalf(InvalidCSVFormat)
	}

	return csvReader
}

func downloadFile(URL, filePath string) error {
	response, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf(CouldNotFetchImage, response.StatusCode))
	}
	//Create a empty file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	//Write the bytes to the file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func ConvertFile(url string, c Converter, r chan *Output, wg *sync.WaitGroup) {
	defer wg.Done()
	fileExt := "jpg"
	fileName := extractFilename(url)
	inputPath := fmt.Sprintf("/outputs/%s.%s", fileName, fileExt)
	outputPath := fmt.Sprintf("/outputs/%s-%s.%s", fileName, "converted", fileExt)

	fmt.Println("Downloading image from URL: ", url)
	err := downloadFile(url, inputPath)

	if err != nil {
		r <- &Output{url: url, err: err}
	}

	// Do the conversion!
	err = c.Grayscale(inputPath, outputPath)

	if err != nil {
		r <- &Output{url: url, input: inputPath, output: outputPath, err: err}
		return
	}

	r <- &Output{url: url, input: inputPath, output: outputPath}
}

func extractFilename(url string) string {
	urlArr := strings.Split(url, "/")

	fileName := strings.Split(urlArr[len(urlArr)-1], "?")[0]

	fileName = strings.ReplaceAll(fileName, ".jpg", "")

	fileName = fmt.Sprintf("%s-%d", fileName, time.Now().Unix())

	return fileName
}

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
