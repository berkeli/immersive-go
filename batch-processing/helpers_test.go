package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDownloadFileFromUrl(t *testing.T) {
	type Test struct {
		Url            string
		ExpectedErr    error
		ExpectedFormat string
	}
	tests := map[string]Test{
		"valid PNG": {
			Url:            "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png",
			ExpectedErr:    nil,
			ExpectedFormat: "png",
		},
		"valid JPEG": {
			Url:            "https://placekitten.com/408/287",
			ExpectedErr:    nil,
			ExpectedFormat: "jpeg",
		},
		"valid GIF": {
			Url:            "https://via.placeholder.com/350x150.gif",
			ExpectedErr:    nil,
			ExpectedFormat: "gif",
		},
		"invalid URL": {
			Url:         "https://via.placeholder.com/",
			ExpectedErr: errors.New(fmt.Sprintf("Received status 403 when trying to download image")),
		},
		"Not and image": {
			Url:         "https://google.com/",
			ExpectedErr: errors.New(fmt.Sprintf("image: unknown format")),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, format, err := DownloadFileFromUrl(test.Url)

			require.Equal(t, test.ExpectedErr, err)

			if test.ExpectedErr != nil {
				require.Equal(t, format, test.ExpectedFormat)
			}

		})
	}
}

func TestExtractFileName(t *testing.T) {
	type Test struct {
		Url      string
		Expected string
	}
	tests := map[string]Test{
		"valid URL": {
			Url:      "https://via.placeholder.com/350x150.gif",
			Expected: "350x150",
		},
		"invalid URL": {
			Url:      "https://via.placeholder.com/",
			Expected: "",
		},
		"with query params": {
			Url:      "https://via.placeholder.com/350x150.gif?test=1",
			Expected: "350x150",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			name := extractFilename(test.Url)

			require.Equal(t, test.Expected, strings.Split(name, "-")[0])
		})
	}
}

func TestContains(t *testing.T) {
	type Test struct {
		Arr      []string
		Val      string
		Expected bool
	}
	tests := map[string]Test{
		"valid": {
			Arr:      []string{"jpeg", "png", "gif"},
			Expected: true,
		},
		"invalid": {
			Arr:      []string{"jpeg", "png"},
			Expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.Expected, contains(test.Arr, "gif"))
		})
	}
}

func SetupSuite(t *testing.T, contents [][]string) (string, func()) {
	// Create a temporary file
	tempDir, err := os.MkdirTemp("/", "tests")
	require.NoError(t, err)

	tmpFileName := fmt.Sprintf("test-%d.csv", time.Now().UnixNano())

	file, err := os.Create(path.Join(tempDir, tmpFileName))
	require.NoError(t, err)

	csvwriter := csv.NewWriter(file)

	for _, row := range contents {
		err = csvwriter.Write(row)
		require.NoError(t, err)
	}

	csvwriter.Flush()

	file.Close()

	return path.Join(tempDir, tmpFileName), func() {
		// teardown
		err := os.RemoveAll(tempDir)
		require.NoError(t, err)
	}
}

func TestOpenCSVFile(t *testing.T) {

	type Test struct {
		Contents    [][]string
		ExpectedErr error
	}
	tests := map[string]Test{
		"valid": {
			Contents: [][]string{
				{"url"},
				{"https://via.placeholder.com/350x150.gif"},
				{"https://via.placeholder.com/350x150.gif"},
			},
		},
		"Empty CSV": {
			Contents:    [][]string{},
			ExpectedErr: errors.New(EmptyCSV),
		},
		"Invalid header": {
			Contents: [][]string{
				{"not a valid header"},
				{"https://via.placeholder.com/350x150.gif"},
			},
			ExpectedErr: errors.New(InvalidCSVFormat),
		},
		"More than 1 columns": {
			Contents: [][]string{
				{"url", "test"},
				{"https://via.placeholder.com/350x150.gif", "asd"},
			},
			ExpectedErr: errors.New("Error reading CSV: record on line 1: wrong number of fields"),
		},
		"Should only check header row, others will be checked per row basis": {
			Contents: [][]string{
				{"url"},
				{"https://via.placeholder.com/350x150.gif", "asd"},
			},
			ExpectedErr: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tmpFile, teardown := SetupSuite(t, test.Contents)
			defer teardown()

			_, err := OpenCSVFile(tmpFile)

			require.Equal(t, test.ExpectedErr, err)
		})
	}
}
