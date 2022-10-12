package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"errors"
)

const (
	InvalidCSVFormat   = "CSV file must have a header row with 'url' as the first column"
	CouldNotFetchImage = "Received status %d when trying to download image"
)

var (
	SupportedImageTypes = []string{
		"image/jpeg",
		"image/png",
	}
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
	mimeType := response.Header.Get("Content-Type")

	if !isSupportedImageType(mimeType) {
		return errors.New(fmt.Sprintf("Unsupported image type: %s, only the following are supported: %s", mimeType, SupportedImageTypes))
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

func extractFilename(url string) string {
	urlArr := strings.Split(url, "/")

	fileName := strings.Split(urlArr[len(urlArr)-1], "?")[0]

	fileName = strings.ReplaceAll(fileName, ".jpg", "")

	fileName = fmt.Sprintf("%s-%d", fileName, time.Now().Unix())

	return fileName
}

func isSupportedImageType(mimeType string) bool {
	for _, t := range SupportedImageTypes {
		if t == mimeType {
			return true
		}
	}

	return false
}
