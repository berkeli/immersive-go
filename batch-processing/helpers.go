package main

import (
	"bytes"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
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
func DownloadFileFromUrl(URL string) (io.Reader, string, [16]byte, error) {
	response, err := http.Get(URL)
	hash := [16]byte{}
	if err != nil {
		return nil, "", hash, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", hash, fmt.Errorf(CouldNotFetchImage, response.StatusCode)
	}

	var buf bytes.Buffer

	tee := io.TeeReader(response.Body, &buf)

	// this reads only part of the file for format etc
	_, format, err := image.DecodeConfig(tee)

	if err != nil {
		return nil, "", hash, err
	}
	// copy rest of the file
	io.Copy(&buf, tee)

	hash = md5.Sum(buf.Bytes())

	SupportedImageTypes := []string{"jpeg", "png", "gif"}

	if !contains(SupportedImageTypes, format) {
		return nil, "", hash, fmt.Errorf("unsupported image type, only the following are supported: %s", SupportedImageTypes)
	}

	return &buf, format, hash, nil
}

/**
* Download the image from the URL with exponential backoff
* @param: {string} URL - the URL of the image to download
* @return: {io.Reader} body - the body of the image
* @return: {string} ext - the file extension (format) of the image
* @return: {string} hash - the md5 hash of the image
* @return: {error} err - any error that occurred
 */
func DownloadWithBackoff(url string, maxRetries uint64) (io.Reader, string, string, error) {
	var body io.Reader
	var format string
	var hash [16]byte
	var err error

	operation := func() error {
		body, format, hash, err = DownloadFileFromUrl(url)

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
		return nil, "", "", err
	}

	return body, format, hex.EncodeToString(hash[:]), nil
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

func UploadToS3WithBackoff(file *os.File, key string, a *AWSConfig, maxRetries uint64) error {
	operation := func() error {
		_, err := a.PutObject(&s3.PutObjectInput{
			Bucket: &a.s3bucket,
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
