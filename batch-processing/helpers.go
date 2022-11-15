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
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"errors"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cenkalti/backoff/v4"
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
		return nil, "", backoff.Permanent(fmt.Errorf("unsupported image type, only the following are supported: %s", SupportedImageTypes))
	}

	return &buf, format, nil
}

/**
* Download the image from the URL with exponential backoff
* @param: {string} URL - the URL of the image to download
* @return: {io.Reader} body - the body of the image
* @return: {string} ext - the file extension (format) of the image
* @return: {error} err - any error that occurred
 */
func DownloadWithBackoff(url string, maxRetries uint64) (io.Reader, string, error) {
	var body io.Reader
	var format string
	var err error

	operation := func() error {
		body, format, err = DownloadFileFromUrl(url)

		if err != nil {
			return err
		}

		return nil
	}

	notify := func(err error, t time.Duration) {
		log.Printf("Error downloading file from %s: %s. Retrying in %s)\n", url, err, t)
	}

	b := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries)

	err = backoff.RetryNotify(operation, b, notify)

	if err != nil {
		return nil, "", err
	}

	return body, format, nil
}

/**
* Extract the filename from the URL
* @param: {string} url - the URL of the image
* @return: {string} filename - the filename of the image
 */
func extractFilename(url string, id int64) string {
	urlArr := strings.Split(url, "/")

	fileName := strings.Split(urlArr[len(urlArr)-1], "?")[0]

	i := strings.LastIndex(fileName, ".")
	if i != -1 {
		fileName = fileName[:i]
	}

	fileName = fmt.Sprintf("%s-%d", fileName, id)

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

func UploadToS3WithBackoff(file *os.File, key string, aws *AWSConfig, maxRetries uint64) error {
	operation := func() error {
		_, err := aws.PutObject(&s3.PutObjectInput{
			Bucket: &aws.s3bucket,
			Key:    &key,
			Body:   file,
		})
		if err != nil {
			return err
		}

		return nil
	}

	notify := func(err error, t time.Duration) {
		log.Printf("Error uploading file to S3: %s. Retrying in %s)\n", err, t)
	}

	b := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries)

	err := backoff.RetryNotify(operation, b, notify)

	return err
}
