package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
)

func ValidateIndent(indent string) (int, error) {
	i, err := strconv.Atoi(indent)
	if err != nil {
		log.Printf("Unable to parse indent: %s", err.Error())
		return 0, fmt.Errorf("Unable to parse indent: %s", indent)
	}
	if i < 0 {
		return 0, fmt.Errorf("Indent cannot be negative: %s", indent)
	}
	return i, nil
}

func ValidateAltText(title, altText string) error {
	titleArr := strings.Split(strings.ToLower(title), " ")
	altTextArr := strings.Split(strings.ToLower(altText), " ")

	if len(altTextArr) < len(titleArr) {
		return fmt.Errorf("Alt text must contain at least as many words as the title")
	}
	matches := 0
	sd := metrics.NewSorensenDice()
	sd.CaseSensitive = false
	for _, altWord := range altTextArr {
		for _, titleWord := range titleArr {
			if strutil.Similarity(altWord, titleWord, sd) > 0.5 {
				matches++
			}
		}
	}

	if matches < 1 {
		return fmt.Errorf("Alt text doesn't seem to be relevant to the title")
	}

	return nil
}

func ValidateImage(url string) (int, int, error) {
	resp, err := http.Get(url)

	if err != nil {
		return 0, 0, fmt.Errorf("Unable to fetch image: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, 0, fmt.Errorf("Unable to fetch image: %s", url)
	}

	im, format, err := image.DecodeConfig(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to decode image: %s", err.Error())
	}

	if !contains([]string{"jpeg", "png", "gif"}, format) {
		return 0, 0, fmt.Errorf("Unsupported image format: %s", format)
	}

	if im.Width == 0 || im.Height == 0 {
		return 0, 0, fmt.Errorf("Image is empty")
	}

	return im.Width, im.Height, nil
}

func contains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
