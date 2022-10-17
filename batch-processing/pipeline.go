package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func Read(inputFilepath *string, out chan *Out) {
	// Read the input CSV file
	// For each line, send the URL to the download channel
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

	for {
		row, err := csvReader.Read()
		if err != nil {
			out <- &Out{Err: err}
			continue
		}
		out <- &Out{Url: row[0]}
	}
}

func Download(in <-chan *Out, out chan *Out) {
	// Download the image from the URL
	// Send the image to the convert channel
	for row := range in {
		if row.Err != nil {
			out <- row
			continue
		}
		body, err := DownloadFileFromUrl(row.Url)

		if err != nil {
			row.Err = err
			out <- row
			continue
		}

		//Create a empty file
		fileExt := "jpg"
		fileName := extractFilename(row.Url)
		inputPath := fmt.Sprintf("/outputs/%s.%s", fileName, fileExt)

		file, err := os.Create(inputPath)
		if err != nil {
			out <- &Out{
				Url: row.Url,
				Err: err,
			}
			continue
		}
		defer file.Close()

		//Write the bytes to the file
		_, err = io.Copy(file, body)
		if err != nil {
			out <- &Out{
				Url: row.Url,
				Err: err,
			}
			continue
		}

		out <- &Out{
			Url:   row.Url,
			Input: inputPath,
		}
	}
}
