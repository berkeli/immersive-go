package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
	"gopkg.in/gographics/imagick.v2/imagick"
)

func TestPipeline_Read(t *testing.T) {
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

			p := NewPipeline(&Config{})
			p.channels[READ] = make(chan *Out, test.Size)
			p.Read(r)

			var gotArr []*Out

			for v := range p.channels[READ] {
				gotArr = append(gotArr, v)
			}

			require.ElementsMatch(t, test.ExpectedOutput, gotArr)

		})

	}
}

func TestPipeline_Download(t *testing.T) {
	fs := http.FileServer(http.Dir("/inputs/test_assets"))

	srv := httptest.NewServer(fs)

	type Test struct {
		In             []*Out
		ExpectedOutput []*Out
	}

	tests := map[string]Test{
		"valid PNG": {
			In: []*Out{
				{
					Url: srv.URL + "/test.png",
				},
			},
			ExpectedOutput: []*Out{
				{
					Url:    srv.URL + "/test.png",
					Input:  "/outputs/test-1.png",
					Output: "/outputs/test-1-converted.png",
				},
			},
		},
		"valid JPG": {
			In: []*Out{
				{
					Url: srv.URL + "/test.jpg",
				},
			},
			ExpectedOutput: []*Out{
				{
					Url:    srv.URL + "/test.jpg",
					Input:  "/outputs/test-1.jpeg",
					Output: "/outputs/test-1-converted.jpeg",
				},
			},
		},
		"valid GIF": {
			In: []*Out{
				{
					Url: srv.URL + "/test.gif",
				},
			},
			ExpectedOutput: []*Out{
				{
					Url:    srv.URL + "/test.gif",
					Input:  "/outputs/test-1.gif",
					Output: "/outputs/test-1-converted.gif",
				},
			},
		},
		"Not an image #1": {
			In: []*Out{
				{
					Url: srv.URL + "/test.txt",
				},
			},
			ExpectedOutput: []*Out{
				{
					Url: srv.URL + "/test.txt",
					Err: errors.New("image: unknown format"),
				},
			},
		},
		"Not an image #2 (Remote URL)": {
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
		"invalid url": {
			In: []*Out{
				{
					Url: "http://notavalidurl:8080/test.png",
				},
			},
			ExpectedOutput: []*Out{
				{
					Url: "http://notavalidurl:8080/test.png",
					Err: errors.New("no such host"),
				},
			},
		},
		"erroneous row": {
			In: []*Out{
				{
					Url: "http://notavalidurl:8080/test.png",
					Err: errors.New("wrong number of fields"),
				},
			},
			ExpectedOutput: []*Out{
				{
					Url: "http://notavalidurl:8080/test.png",
					Err: errors.New("wrong number of fields"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			p := NewPipeline(&Config{})

			p.uuidGen = func() int64 {
				return 1
			}

			p.channels[READ] = make(chan *Out, len(test.In))
			p.channels[DOWNLOAD] = make(chan *Out, len(test.In))

			for _, v := range test.In {
				p.channels[READ] <- v
			}
			close(p.channels[READ])
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go p.Download(wg)
			wg.Wait()
			close(p.channels[DOWNLOAD])

			var gotArr []*Out

			for v := range p.channels[DOWNLOAD] {
				gotArr = append(gotArr, v)
			}

			require.Equal(t, len(gotArr), len(test.ExpectedOutput))

			for _, downloadRow := range gotArr {
				for _, expRow := range test.ExpectedOutput {
					if downloadRow.Url == expRow.Url {
						require.Equal(t, expRow.Input, downloadRow.Input)
						require.Equal(t, expRow.Output, downloadRow.Output)

						if expRow.Err != nil {
							require.ErrorContains(t, downloadRow.Err, expRow.Err.Error())
							require.Equal(t, downloadRow.Input, "")
							require.Equal(t, downloadRow.Output, "")
						}

						if downloadRow.Err == nil {
							verifyFile(t, downloadRow.Input)
						}
					}
				}
			}

			t.Cleanup(func() {
				for _, outRow := range gotArr {
					if outRow.Err == nil {
						os.Remove(outRow.Input)
					}
				}
			})
		})
	}
}

func TestPipeline_Convert(t *testing.T) {
	// setup mock Converter
	setupMockConverter := func(t *testing.T, calls *[][]string) *Converter {
		// setup mock converter
		c := &Converter{
			cmd: func(a []string) (*imagick.ImageCommandResult, error) {
				*calls = append(*calls, a)
				return &imagick.ImageCommandResult{
					Info: nil,
					Meta: "",
				}, nil
			},
		}
		return c
	}

	tests := map[string]struct {
		In            []*Out
		ExpectedCalls [][]string
		ExpectedOut   []*Out
	}{
		"valid image": {
			In: []*Out{
				{
					Input:  "/outputs/test.jpg",
					Output: "/outputs/test-converted.jpg",
				},
			},
			ExpectedCalls: [][]string{
				{"convert", "/outputs/test.jpg", "-set", "colorspace", "Gray", "/outputs/test-converted.jpg"},
			},
			ExpectedOut: []*Out{
				{
					Input:  "/outputs/test.jpg",
					Output: "/outputs/test-converted.jpg",
				},
			},
		},
		"multiple images": {
			In: []*Out{
				{
					Input:  "/outputs/test1.jpg",
					Output: "/outputs/test1-converted.jpg",
				},
				{
					Input:  "/outputs/test2.jpg",
					Output: "/outputs/test2-converted.jpg",
				},
			},
			ExpectedCalls: [][]string{
				{"convert", "/outputs/test1.jpg", "-set", "colorspace", "Gray", "/outputs/test1-converted.jpg"},
				{"convert", "/outputs/test2.jpg", "-set", "colorspace", "Gray", "/outputs/test2-converted.jpg"},
			},
			ExpectedOut: []*Out{
				{
					Input:  "/outputs/test1.jpg",
					Output: "/outputs/test1-converted.jpg",
				},
				{
					Input:  "/outputs/test2.jpg",
					Output: "/outputs/test2-converted.jpg",
				},
			},
		},
		"erroneous rows should be skipped": {
			In: []*Out{
				{
					Input:  "/outputs/test1.jpg",
					Output: "/outputs/test1-converted.jpg",
				},
				{
					Input: "/outputs/test2.jpg",
					Err:   fmt.Errorf("some error"),
				},
			},
			ExpectedCalls: [][]string{
				{"convert", "/outputs/test1.jpg", "-set", "colorspace", "Gray", "/outputs/test1-converted.jpg"},
			},
			ExpectedOut: []*Out{
				{
					Input:  "/outputs/test1.jpg",
					Output: "/outputs/test1-converted.jpg",
				},
				{
					Input: "/outputs/test2.jpg",
					Err:   fmt.Errorf("some error"),
				},
			},
		},
	}

	for name, test := range tests {

		t.Run(name, func(t *testing.T) {

			got := [][]string{}
			p := NewPipeline(&Config{
				Converter: setupMockConverter(t, &got),
			})

			p.channels[DOWNLOAD] = make(chan *Out, len(test.In))
			p.channels[CONVERT] = make(chan *Out, len(test.In))

			for _, v := range test.In {
				p.channels[DOWNLOAD] <- v
			}

			close(p.channels[DOWNLOAD])

			wg := &sync.WaitGroup{}
			wg.Add(1)
			go p.Convert(wg)
			wg.Wait()
			close(p.channels[CONVERT])

			var gotArr []*Out

			for v := range p.channels[CONVERT] {
				gotArr = append(gotArr, v)
			}

			require.Equal(t, len(gotArr), len(test.In))

			require.ElementsMatch(t, test.In, gotArr)

			require.ElementsMatch(t, test.ExpectedCalls, got)

			require.ElementsMatch(t, test.ExpectedOut, gotArr)
		})
	}

	t.Run("erroneous converter test", func(t *testing.T) {
		p := NewPipeline(&Config{
			Converter: &Converter{
				cmd: func(a []string) (*imagick.ImageCommandResult, error) {
					return nil, fmt.Errorf("some error")
				},
			},
		})

		p.channels[DOWNLOAD] = make(chan *Out, 1)
		p.channels[CONVERT] = make(chan *Out, 1)

		p.channels[DOWNLOAD] <- &Out{
			Input:  "/outputs/test1.jpg",
			Output: "/outputs/test1-converted.jpg",
		}

		close(p.channels[DOWNLOAD])

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go p.Convert(wg)
		wg.Wait()
		close(p.channels[CONVERT])

		var gotArr []*Out

		for v := range p.channels[CONVERT] {
			gotArr = append(gotArr, v)
		}

		require.Equal(t, 1, len(gotArr))

		require.Equal(t, "/outputs/test1.jpg", gotArr[0].Input)
		require.Equal(t, "", gotArr[0].Output)
		require.EqualError(t, gotArr[0].Err, "some error")
	})
}

func TestPipeline_Upload(t *testing.T) {
	call := &s3.PutObjectInput{}
	mockPutObject := func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
		call = input
		return &s3.PutObjectOutput{}, nil
	}

	p := NewPipeline(&Config{
		Aws: &AWSConfig{
			s3bucket:  "some_bucket",
			PutObject: mockPutObject,
		},
	})

	tests := map[string]struct {
		In           []*Out
		ExpectedCall *s3.PutObjectInput
		Out          []*Out
	}{
		"valid file": {
			In: []*Out{
				{
					Output: "/inputs/test_assets/test.jpg",
				},
			},
			ExpectedCall: &s3.PutObjectInput{
				Bucket: aws.String("some_bucket"),
				Key:    aws.String("test.jpg"),
			},
			Out: []*Out{
				{
					Output: "/inputs/test_assets/test.jpg",
					S3url:  fmt.Sprintf("https://%s.s3.amazonaws.com/test.jpg", "some_bucket"),
					Err:    nil,
				},
			},
		},
		"erroneous file": {
			In: []*Out{
				{
					Output: "/inputs/test_assets/not-found.jpg",
				},
			},
			ExpectedCall: &s3.PutObjectInput{},
			Out: []*Out{
				{
					Output: "/inputs/test_assets/not-found.jpg",
					S3url:  "",
					Err: &fs.PathError{
						Op:   "open",
						Path: "/inputs/test_assets/not-found.jpg",
						Err:  syscall.ENOENT,
					},
				},
			},
		},
		"erroneous row": {
			In: []*Out{
				{
					Output: "/inputs/test_assets/not-found.jpg",
					Err:    fmt.Errorf("some error"),
				},
			},
			ExpectedCall: &s3.PutObjectInput{},
			Out: []*Out{
				{
					Output: "/inputs/test_assets/not-found.jpg",
					S3url:  "",
					Err:    fmt.Errorf("some error"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p.channels[CONVERT] = make(chan *Out, len(test.In))
			p.channels[UPLOAD] = make(chan *Out, len(test.In))

			for _, v := range test.In {
				p.channels[CONVERT] <- v
			}

			close(p.channels[CONVERT])

			wg := &sync.WaitGroup{}
			wg.Add(1)
			go p.Upload(wg)
			wg.Wait()
			close(p.channels[UPLOAD])

			var gotArr []*Out

			for v := range p.channels[UPLOAD] {
				gotArr = append(gotArr, v)
			}

			require.Equal(t, len(gotArr), len(test.In))

			require.ElementsMatch(t, test.Out, gotArr)

			require.Equal(t, test.ExpectedCall.Bucket, call.Bucket)
			require.Equal(t, test.ExpectedCall.Key, call.Key)
			t.Cleanup(func() {
				call = &s3.PutObjectInput{}
			})
		})
	}

	t.Run("erroneous PutObject", func(t *testing.T) {
		errPutObject := func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
			return &s3.PutObjectOutput{}, fmt.Errorf("some error")
		}

		p := NewPipeline(&Config{
			Aws: &AWSConfig{
				s3bucket:  "some_bucket",
				PutObject: errPutObject,
			},
		})

		p.channels[CONVERT] = make(chan *Out, 1)
		p.channels[UPLOAD] = make(chan *Out, 1)

		p.channels[CONVERT] <- &Out{
			Output: "/inputs/test_assets/test.jpg",
		}

		close(p.channels[CONVERT])

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go p.Upload(wg)
		wg.Wait()
		close(p.channels[UPLOAD])

		var gotArr []*Out

		for v := range p.channels[UPLOAD] {
			gotArr = append(gotArr, v)
		}

		require.Equal(t, len(gotArr), 1)

		require.Equal(t, "/inputs/test_assets/test.jpg", gotArr[0].Output)
		require.Equal(t, "", gotArr[0].S3url)
		require.Equal(t, fmt.Errorf("some error"), gotArr[0].Err)
	})

}

func TestPipeline_Write(t *testing.T) {

	setupSuite := func(t *testing.T, outputFile, failedOutputFile string) (*Pipeline, func()) {
		t.Helper()
		// Create a temporary folder
		tempDir, err := os.MkdirTemp("/outputs", "tests")
		require.NoError(t, err)

		p := NewPipeline(&Config{})

		if outputFile != "" {
			p.config.OutputFilepath = path.Join(tempDir, outputFile)
		}

		if failedOutputFile != "" {
			p.config.FailedOutputFilepath = path.Join(tempDir, failedOutputFile)
		}

		return p, func() {
			os.RemoveAll(tempDir)
		}
	}

	tests := map[string]struct {
		OutputFilepath       string
		FailedOutputFilepath string
		In                   []*Out
		ExpectedCSV          [][]string
		ExpectedFailedCSV    [][]string
	}{
		"valid output, no failed": {
			OutputFilepath: "test.csv",
			In: []*Out{
				{
					Url:    "https://some-url.com/1.jpg",
					Input:  "/outputs/1.jpg",
					Output: "/outputs/1-converted.jpg",
					S3url:  "https://some-bucket.s3.amazonaws.com/1-converted.jpg",
				},
			},
			ExpectedCSV: [][]string{
				OutputHeader,
				{"https://some-url.com/1.jpg", "/outputs/1.jpg", "/outputs/1-converted.jpg", "https://some-bucket.s3.amazonaws.com/1-converted.jpg", ""},
			},
		},
		"valid output, with failed": {
			OutputFilepath:       "test.csv",
			FailedOutputFilepath: "failed.csv",
			In: []*Out{
				{
					Url:    "https://some-url.com/1.jpg",
					Input:  "/outputs/1.jpg",
					Output: "/outputs/1-converted.jpg",
					S3url:  "https://some-bucket.s3.amazonaws.com/1-converted.jpg",
				},
				{
					Url:    "https://some-url.com/2.jpg",
					Input:  "/outputs/2.jpg",
					Output: "/outputs/2-converted.jpg",
					Err:    fmt.Errorf("some error"),
				},
			},
			ExpectedCSV: [][]string{
				OutputHeader,
				{"https://some-url.com/1.jpg", "/outputs/1.jpg", "/outputs/1-converted.jpg", "https://some-bucket.s3.amazonaws.com/1-converted.jpg", ""},
				{"https://some-url.com/2.jpg", "/outputs/2.jpg", "/outputs/2-converted.jpg", "", "some error"},
			},
			ExpectedFailedCSV: [][]string{
				FailedOutputHeader,
				{"https://some-url.com/2.jpg"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, teardown := setupSuite(t, test.OutputFilepath, test.FailedOutputFilepath)
			defer teardown()

			p.channels[UPLOAD] = make(chan *Out, len(test.In))

			for _, v := range test.In {
				p.channels[UPLOAD] <- v
			}

			close(p.channels[UPLOAD])

			done := make(chan bool)
			go p.Write(done)

			<-done

			if test.OutputFilepath != "" {
				f, err := os.Open(p.config.OutputFilepath)
				require.NoError(t, err)

				r := csv.NewReader(f)
				content, err := r.ReadAll()
				require.NoError(t, err)

				require.ElementsMatch(t, test.ExpectedCSV, content)
			}

			if test.FailedOutputFilepath != "" {
				f, err := os.Open(p.config.FailedOutputFilepath)
				require.NoError(t, err)

				r := csv.NewReader(f)
				content, err := r.ReadAll()
				require.NoError(t, err)

				require.ElementsMatch(t, test.ExpectedFailedCSV, content)
			}
		})
	}
}

func TestPipeline_Execute(t *testing.T) {

	setupSuite := func(t *testing.T, data [][]string) (*Pipeline, func(string)) {
		t.Helper()
		// Create a temporary folder
		tempDir, err := os.MkdirTemp("/outputs", "tests")
		require.NoError(t, err)

		//Create input file
		inputFilepath := path.Join(tempDir, "input.csv")
		f, err := os.Create(inputFilepath)
		require.NoError(t, err)

		w := csv.NewWriter(f)
		err = w.WriteAll(data)
		require.NoError(t, err)

		w.Flush()

		imagick.Initialize()
		defer imagick.Terminate()

		p := NewPipeline(&Config{
			Converter: &Converter{
				cmd: imagick.ConvertImageCommand,
			},
			InputFilepath: inputFilepath,
			Aws: &AWSConfig{
				region:   "us-east-1",
				s3bucket: "some-bucket",
				PutObject: func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
					return &s3.PutObjectOutput{}, nil
				},
			},
		})

		p.config.OutputFilepath = path.Join(tempDir, "test.csv")
		p.config.FailedOutputFilepath = path.Join(tempDir, "failed.csv")

		p.uuidGen = func() int64 {
			return 1
		}

		return p, func(outputFile string) {
			defer os.RemoveAll(tempDir)

			if outputFile != "" {
				f, err := os.Open(outputFile)
				require.NoError(t, err)

				r := csv.NewReader(f)
				content, err := r.ReadAll()

				require.NoError(t, err)

				for _, row := range content {
					os.Remove(row[1])
					os.Remove(row[2])
				}
			}
		}
	}

	t.Run("valid urls from local", func(t *testing.T) {

		fs := http.FileServer(http.Dir("/inputs/test_assets"))
		srv := httptest.NewServer(fs)
		defer srv.Close()

		inputCSV := [][]string{
			{"url"},
			{srv.URL + "/test.jpg"},
			{srv.URL + "/test.gif"},
			{srv.URL + "/test.png"},
			{srv.URL + "/test.txt"},
			{srv.URL + "/not-found"},
		}

		wantOutputCSV := [][]string{
			OutputHeader,
			{srv.URL + "/test.jpg", "/outputs/test-1.jpeg", "/outputs/test-1-converted.jpeg", "https://some-bucket.s3.amazonaws.com/test-1-converted.jpeg", ""},
			{srv.URL + "/test.gif", "/outputs/test-1.gif", "/outputs/test-1-converted.gif", "https://some-bucket.s3.amazonaws.com/test-1-converted.gif", ""},
			{srv.URL + "/test.png", "/outputs/test-1.png", "/outputs/test-1-converted.png", "https://some-bucket.s3.amazonaws.com/test-1-converted.png", ""},
			{srv.URL + "/test.txt", "", "", "", "image: unknown format"},
			{srv.URL + "/not-found", "", "", "", "received status 404 when trying to download image"},
		}

		wantFailedOutputCSV := [][]string{
			FailedOutputHeader,
			{srv.URL + "/test.txt"},
			{srv.URL + "/not-found"},
		}

		p, teardown := setupSuite(t, inputCSV)

		p.Execute()

		assertCSVOutputFile(t, p.config.OutputFilepath, wantOutputCSV)
		assertCSVOutputFile(t, p.config.FailedOutputFilepath, wantFailedOutputCSV)

		teardown(p.config.OutputFilepath)
	})
}

func verifyFile(t *testing.T, path string) {
	t.Helper()
	f, err := os.Stat(path)
	require.NoError(t, err)

	require.True(t, f.Size() > 0)
}

func assertCSVOutputFile(t *testing.T, path string, want [][]string) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)

	r := csv.NewReader(f)
	content, err := r.ReadAll()

	require.NoError(t, err)

	require.ElementsMatch(t, want, content)
}
