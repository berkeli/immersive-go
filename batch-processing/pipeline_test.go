package main

import (
	"encoding/csv"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	type Test struct {
		CSVContent     string
		Size           int
		ExpectedOutput []*Out
	}

	csvErr := &csv.ParseError{
		StartLine: 1,
		Line:      1,
		Column:    1,
		Err:       errors.New("wrong number of fields"),
	}

	tests := map[string]Test{
		"valid CSV": {
			CSVContent: `https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png`,
			Size:       1,
			ExpectedOutput: []*Out{
				{
					Url: "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png",
				},
			},
		},
		"CSV with extra column": {
			CSVContent: `https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png,extra`,
			Size:       1,
			ExpectedOutput: []*Out{
				{
					Err: csvErr,
				},
			},
		},
		"must still return following rows after error": {
			CSVContent: `https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png,extra
https://placekitten.com/408/287`,
			Size: 2,
			ExpectedOutput: []*Out{
				{
					Err: csvErr,
				},
				{
					Url: "https://placekitten.com/408/287",
					Err: nil,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r := csv.NewReader(strings.NewReader(test.CSVContent))

			got := make(chan *Out, test.Size)

			Read(r, got)

			var gotArr []*Out

			for v := range got {
				gotArr = append(gotArr, v)
			}

			require.ElementsMatch(t, test.ExpectedOutput, gotArr)

		})

	}
}

func TestDownload(t *testing.T) {
	type Test struct {
		In             []*Out
		ExpectedOutput []*Out
	}

	tests := map[string]Test{
		"valid URL": {
			In: []*Out{
				{
					Url: "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png",
				},
			},
			ExpectedOutput: []*Out{
				{
					Url:   "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png",
					Input: "tmp/outputs/googlelogo_color_272x92dp",
				},
			},
		},
		"invalid URL": {
			In: []*Out{
				{
					Url: "https://www.google.com/",
				},
			},
			ExpectedOutput: []*Out{
				{
					Url: "https://www.google.com/",
					Err: errors.New("image: unknown format"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			in := make(chan *Out, len(test.In))
			out := make(chan *Out, len(test.In))

			for _, v := range test.In {
				in <- v
			}
			close(in)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			Download(in, out, wg)
			close(out)

			var gotArr []*Out

			for v := range out {
				gotArr = append(gotArr, v)

			}

			require.Equal(t, len(gotArr), len(test.ExpectedOutput))

			for _, outRow := range gotArr {
				log.Println(outRow)

				for _, inRow := range test.ExpectedOutput {
					if outRow.Url == inRow.Url {
						require.True(t, strings.HasPrefix(outRow.Input, inRow.Input), "%s must have a prefix of %s", outRow.Input, inRow.Input)
						require.True(t, strings.HasPrefix(outRow.Output, inRow.Output))

						if inRow.Err != nil {
							require.Equal(t, outRow.Err, inRow.Err)
							require.Equal(t, outRow.Input, "")
							require.Equal(t, outRow.Output, "")
						}

						if outRow.Err == nil {
							VerifyFile(t, outRow.Input)
						}
					}
				}
			}
		})
	}
}

func VerifyFile(t *testing.T, path string) {
	t.Helper()
	f, err := os.Stat(path)
	require.NoError(t, err)

	require.True(t, f.Size() > 0)
}
