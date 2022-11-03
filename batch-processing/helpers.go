package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"errors"
)

const (
	InvalidCSVFormat   = "csv file must have a header row with 'url' as the first column"
	CouldNotFetchImage = "received status %d when trying to download image"
	EmptyCSV           = "provided CSV file appears to be empty"
)

/**
* Download the image from the URL
* @param: {string} URL - the URL of the image to download
* @return: {io.Reader} body - the body of the image
* @return: {string} ext - the file extension (format) of the image
* @return: {error} err - any error that occurred
 */
func DownloadFileFromUrl(URL string) (io.Reader, string, error) {
	response, err := http.Get(URL)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf(CouldNotFetchImage, response.StatusCode)
	}

	var buf bytes.Buffer

	tee := io.TeeReader(response.Body, &buf)

	_, format, err := image.Decode(tee)

	if err != nil {
		return nil, "", err
	}

	SupportedImageTypes := []string{"jpeg", "png", "gif"}

	if !contains(SupportedImageTypes, format) {
		return nil, "", fmt.Errorf("unsupported image type, only the following are supported: %s", SupportedImageTypes)
	}

	return &buf, format, nil
}

/**
* Extract the filename from the URL
* @param: {string} url - the URL of the image
* @return: {string} filename - the filename of the image
 */
func extractFilename(url string) string {
	urlArr := strings.Split(url, "/")

	fileName := strings.Split(urlArr[len(urlArr)-1], "?")[0]

	i := strings.LastIndex(fileName, ".")
	if i != -1 {
		fileName = fileName[:i]
	}

	fileName = fmt.Sprintf("%s-%d", fileName, time.Now().Unix())

	return fileName
}

/**
* Check if the provided image type is supported
* @param: {[]string} SupportedImageTypes - the supported image types
* @param: {string} mimeType - the image type to check
* @return: {bool} - whether or not the image type is supported
 */
func contains(SupportedImageTypes []string, mimeType string) bool {
	for _, t := range SupportedImageTypes {
		if t == mimeType {
			return true
		}
	}

	return false
}

/**
* Function opens the csv file, validates headers and returns the reader.
* @param: {string} filename - the path of the CSV file
* @return: {csv.Reader} csvReader - the CSV reader
* @return: {error} err - any error that occurred
 */
func OpenCSVFile(filename string) (*csv.Reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = 1

	header, err := csvReader.Read()

	if err == io.EOF {
		return nil, errors.New(EmptyCSV)
	}

	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %v", err)
	}

	if strings.ToLower(header[0]) != "url" {
		return nil, errors.New(InvalidCSVFormat)
	}

	return csvReader, nil
}
