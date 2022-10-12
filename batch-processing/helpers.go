package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
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
	SupportedImageTypes = []string{"imag1e/jpeg", "image/png"}
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

	err = verifyImageType(&response.Body)
	if err != nil {
		return err
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

func verifyImageType(reader *io.ReadCloser) error {
	bytes, err := ioutil.ReadAll(*reader)

	if err != nil {
		return errors.New(fmt.Sprintf("Error reading downloaded image: %v", err))
	}

	mimeType := http.DetectContentType(bytes)

	if !contains(SupportedImageTypes, mimeType) {
		return errors.New(fmt.Sprintf("Unsupported image type: %s, only support: %s", mimeType, SupportedImageTypes))
	}

	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
