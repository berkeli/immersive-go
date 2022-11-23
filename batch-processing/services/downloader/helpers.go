package downloader

import (
	"crypto/md5"
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
	"time"

	"github.com/cenkalti/backoff"
)

/**
* Download the image from the URL
* @param: {string} URL - the URL of the image to download
* @return: {[]byte} hash - md5 sum of the image
* @return: {string} ext - the file extension (format) of the image
* @return: {error} err - any error that occurred
 */
func DownloadFileFromUrl(URL string, file *os.File) ([]byte, string, error) {
	response, err := http.Get(URL)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf(CouldNotFetchImage, response.StatusCode)
	}

	hashReader := md5.New()

	tee := io.TeeReader(response.Body, file)
	teeHash := io.TeeReader(tee, hashReader)

	_, format, err := image.Decode(teeHash)

	if err != nil {
		return nil, "", err
	}

	hash := hashReader.Sum(nil)

	SupportedImageTypes := []string{"jpeg", "png", "gif"}

	if !contains(SupportedImageTypes, format) {
		return nil, "", fmt.Errorf("unsupported image type, only the following are supported: %s", SupportedImageTypes)
	}

	return hash, format, nil
}

/**
* Download the image from the URL with exponential backoff
* @param: {string} URL - the URL of the image to download
* @return: {io.Reader} body - the body of the image
* @return: {string} ext - the file extension (format) of the image
* @return: {string} hash - the md5 hash of the image
* @return: {error} err - any error that occurred
 */
func DownloadWithBackoff(url string, maxRetries uint64, file *os.File) (string, string, error) {
	var format string
	var hash []byte
	var err error

	operation := func() error {
		hash, format, err = DownloadFileFromUrl(url, file)

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
		return "", "", err
	}

	return hex.EncodeToString(hash), format, nil
}

/**
* Helper function to check if a string is in a slice of strings
* @param: {[]string} slice - the slice of strings to search
* @param: {string}  value - the value to search for
* @return: {bool} - true if the value is in the slice, false otherwise
 */
func contains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}

	return false
}

func GetMD5Hash(text []byte) string {
	hash := md5.Sum(text)
	return hex.EncodeToString(hash[:])
}
