package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"errors"
)

const (
	InvalidCSVFormat   = "CSV file must have a header row with 'url' as the first column"
	CouldNotFetchImage = "Received status %d when trying to download image"
)

func DownloadFileFromUrl(URL string) (io.Reader, error) {
	response, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf(CouldNotFetchImage, response.StatusCode))
	}
	mimeType := response.Header.Get("Content-Type")

	SupportedImageTypes := []string{
		"image/jpeg",
		"image/png",
	}

	if !contains(SupportedImageTypes, mimeType) {
		return nil, errors.New(fmt.Sprintf("Unsupported image type: %s, only the following are supported: %s", mimeType, SupportedImageTypes))
	}

	return response.Body, nil
}

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

func contains(SupportedImageTypes []string, mimeType string) bool {
	for _, t := range SupportedImageTypes {
		if t == mimeType {
			return true
		}
	}

	return false
}
