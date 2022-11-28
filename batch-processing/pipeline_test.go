package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"sync"
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
		ExpectedOutput []*ReadOut
		ExpectedErrOut []*ErrOut
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
			ExpectedOutput: []*ReadOut{
				{
					Url: "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png",
				},
			},
		},
		"CSV with extra column": {
			CSVContent: `https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png,extra`,
			Size:       1,
			ExpectedErrOut: []*ErrOut{
				{
					Url: "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png",
					Err: csvErr,
				},
			},
		},
		"must still return following rows after error": {
			CSVContent: `https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png,extra
https://placekitten.com/408/287`,
			Size: 2,
			ExpectedOutput: []*ReadOut{
				{
					Url: "https://placekitten.com/408/287",
				},
			},
			ExpectedErrOut: []*ErrOut{
				{
					Url: "https://www.google.com/images/branding/googlelogo/1x/googlelogo_color_272x92dp.png",
					Err: csvErr,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r := csv.NewReader(strings.NewReader(test.CSVContent))

			p := NewPipeline(&Config{})
			readOut := make(chan *ReadOut, len(test.ExpectedOutput))
			p.errOut = make(chan *ErrOut, len(test.ExpectedErrOut))
			go p.Read(r, readOut)

			var gotArr []*ReadOut
			var gotErrArr []*ErrOut

			for i := 0; i < len(test.ExpectedOutput); i++ {
				got := <-readOut
				gotArr = append(gotArr, got)
			}

			for i := 0; i < len(test.ExpectedErrOut); i++ {
				got := <-p.errOut
				gotErrArr = append(gotErrArr, got)
			}

			require.ElementsMatch(t, test.ExpectedOutput, gotArr)
			require.ElementsMatch(t, test.ExpectedErrOut, gotErrArr)

		})

	}
}

func TestPipeline_Download(t *testing.T) {
	fs := http.FileServer(http.Dir("/inputs/test_assets"))

	srv := httptest.NewServer(fs)

	type Test struct {
		In             []*ReadOut
		ExpectedOutput []*DownloadOut
		ExpectedErrOut []*ErrOut
	}

	tests := map[string]Test{
		"valid PNG": {
			In: []*ReadOut{
				{
					Url: srv.URL + "/test.png",
				},
			},
			ExpectedOutput: []*DownloadOut{
				{
					Url: srv.URL + "/test.png",
					Key: "e5765cefe1b8d33fc78315437516439d",
					Ext: "png",
				},
			},
		},
		"valid JPG": {
			In: []*ReadOut{
				{
					Url: srv.URL + "/test.jpg",
				},
			},
			ExpectedOutput: []*DownloadOut{
				{
					Url: srv.URL + "/test.jpg",
					Key: "85e42ea4f380785dd6ae5c6361399d87",
					Ext: "jpeg",
				},
			},
		},
		"valid GIF": {
			In: []*ReadOut{
				{
					Url: srv.URL + "/test.gif",
				},
			},
			ExpectedOutput: []*DownloadOut{
				{
					Url: srv.URL + "/test.gif",
					Key: "d5c423921f7654ad178b167897564b13",
					Ext: "gif",
				},
			},
		},
		"Not an image #1": {
			In: []*ReadOut{
				{
					Url: srv.URL + "/test.txt",
				},
			},
			ExpectedErrOut: []*ErrOut{
				{
					Url: srv.URL + "/test.txt",
					Err: errors.New("image: unknown format"),
				},
			},
		},
		"Not an image #2 (Remote URL)": {
			In: []*ReadOut{
				{
					Url: "https://www.google.com/",
				},
			},
			ExpectedErrOut: []*ErrOut{
				{
					Url: "https://www.google.com/",
					Err: errors.New("image: unknown format"),
				},
			},
		},
		"invalid url": {
			In: []*ReadOut{
				{
					Url: "http://notavalidurl:8080/test.png",
				},
			},
			ExpectedErrOut: []*ErrOut{
				{
					Url: "http://notavalidurl:8080/test.png",
					Err: invalidHostErr(t, "http://notavalidurl:8080/test.png"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			p := NewPipeline(&Config{
				Aws: &AWSConfig{
					GetObject: func(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
						return &s3.GetObjectOutput{}, fmt.Errorf("404 not found")
					},
				},
			})
			p.maxRetries = 1

			readOut := make(chan *ReadOut, len(test.In))
			downloadOut := make(chan *DownloadOut, len(test.In))

			for _, v := range test.In {
				readOut <- v
			}
			close(readOut)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go p.Download(wg, readOut, downloadOut)
			wg.Wait()
			close(downloadOut)

			var gotArr []*DownloadOut

			for v := range downloadOut {
				gotArr = append(gotArr, v)
			}

			t.Cleanup(func() {
				for _, outRow := range gotArr {
					os.Remove(InputPath(outRow.Key, outRow.Ext))
				}
			})

			require.Equal(t, len(gotArr), len(test.ExpectedOutput))

			for _, downloadRow := range gotArr {
				verifyFile(t, InputPath(downloadRow.Key, downloadRow.Ext))
			}

			require.ElementsMatch(t, gotArr, test.ExpectedOutput)

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
		In            []*DownloadOut
		ExpectedCalls [][]string
		ExpectedOut   []*ConvertOut
	}{
		"valid image": {
			In: []*DownloadOut{
				{
					Key: "test",
					Ext: "jpg",
				},
			},
			ExpectedCalls: [][]string{
				{"convert", "/outputs/test.jpg", "-set", "colorspace", "Gray", "/outputs/test-converted.jpg"},
			},
			ExpectedOut: []*ConvertOut{
				{
					Key: "test",
					Ext: "jpg",
				},
			},
		},
		"multiple images": {
			In: []*DownloadOut{
				{
					Key: "test1",
					Ext: "jpg",
				},
				{
					Key: "test2",
					Ext: "png",
				},
			},
			ExpectedCalls: [][]string{
				{"convert", "/outputs/test1.jpg", "-set", "colorspace", "Gray", "/outputs/test1-converted.jpg"},
				{"convert", "/outputs/test2.png", "-set", "colorspace", "Gray", "/outputs/test2-converted.png"},
			},
			ExpectedOut: []*ConvertOut{
				{
					Key: "test1",
					Ext: "jpg",
				},
				{
					Key: "test2",
					Ext: "png",
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

			downOut := make(chan *DownloadOut, len(test.In))
			convertOut := make(chan *ConvertOut, len(test.In))

			for _, v := range test.In {
				downOut <- v
			}

			close(downOut)

			wg := &sync.WaitGroup{}
			wg.Add(1)
			go p.Convert(wg, downOut, convertOut)
			wg.Wait()
			close(convertOut)

			var gotArr []*ConvertOut

			for v := range convertOut {
				gotArr = append(gotArr, v)
			}

			require.Equal(t, len(gotArr), len(test.In))

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

		p.errOut = make(chan *ErrOut, 1)

		downOut := make(chan *DownloadOut, 1)
		convertOut := make(chan *ConvertOut, 1)

		downOut <- &DownloadOut{
			Url: "http://localhost:8080/test1.jpg",
			Key: "test1",
			Ext: "jpg",
		}

		close(downOut)

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go p.Convert(wg, downOut, convertOut)
		wg.Wait()
		close(convertOut)

		var gotArr []*ConvertOut

		for v := range convertOut {
			gotArr = append(gotArr, v)
		}

		expErr := &ErrOut{
			Url: "http://localhost:8080/test1.jpg",
			Key: "test1",
			Ext: "jpg",
			Err: fmt.Errorf("error converting image: some error"),
		}

		require.Equal(t, 0, len(gotArr), "should not have any output")

		require.Equal(t, expErr, <-p.errOut)
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

	p.maxRetries = 1
	p.tmpWorkingDir = "/inputs/test_assets"

	tests := map[string]struct {
		In             []*ConvertOut
		ExpectedCall   *s3.PutObjectInput
		ExpectedOut    []*UploadOut
		ExpectedErrOut []*ErrOut
	}{
		"valid file": {
			In: []*ConvertOut{
				{
					Url: "http://someurl:8080/test.jpg",
					Key: "test",
					Ext: "jpg",
				},
			},
			ExpectedCall: &s3.PutObjectInput{
				Bucket: aws.String("some_bucket"),
				Key:    aws.String("test-converted.jpg"),
			},
			ExpectedOut: []*UploadOut{
				{
					Url:   "http://someurl:8080/test.jpg",
					Key:   "test",
					Ext:   "jpg",
					S3url: "https://some_bucket.s3.amazonaws.com/test-converted.jpg",
				},
			},
		},
		"erroneous file": {
			In: []*ConvertOut{
				{
					Url: "http://someurl:8080/not-found.jpg",
					Key: "not-found",
					Ext: "jpg",
				},
			},
			ExpectedCall: &s3.PutObjectInput{},
			ExpectedErrOut: []*ErrOut{
				{
					Url: "http://someurl:8080/not-found.jpg",
					Key: "not-found",
					Ext: "jpg",
					Err: fmt.Errorf("some error"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			convertOut := make(chan *ConvertOut, len(test.In))
			uploadOut := make(chan *UploadOut, len(test.In))

			t.Cleanup(func() {
				call = &s3.PutObjectInput{}
			})

			for _, v := range test.In {
				convertOut <- v
			}

			close(convertOut)

			wg := &sync.WaitGroup{}
			wg.Add(1)
			go p.Upload(wg, convertOut, uploadOut)
			wg.Wait()
			close(uploadOut)

			var gotArr []*UploadOut

			for v := range uploadOut {
				gotArr = append(gotArr, v)
			}

			require.ElementsMatch(t, test.ExpectedOut, gotArr)

			require.Equal(t, test.ExpectedCall.Bucket, call.Bucket)
			require.Equal(t, test.ExpectedCall.Key, call.Key)
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

		p.maxRetries = 1
		p.tmpWorkingDir = "/inputs/test_assets"

		convertOut := make(chan *ConvertOut, 1)
		uploadOut := make(chan *UploadOut, 1)

		convertOut <- &ConvertOut{
			Url: "http://someurl:8080/test.jpg",
			Key: "test",
			Ext: "jpg",
		}

		close(convertOut)

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go p.Upload(wg, convertOut, uploadOut)
		wg.Wait()
		close(uploadOut)

		var gotArr []*UploadOut

		for v := range uploadOut {
			gotArr = append(gotArr, v)
		}

		expErr := &ErrOut{
			Url: "http://someurl:8080/test.jpg",
			Key: "test",
			Ext: "jpg",
			Err: fmt.Errorf("failed to upload to S3: some error"),
		}

		require.Equal(t, len(gotArr), 0)

		require.Equal(t, expErr, <-p.errOut)
	})

}

func WriteSetup(t *testing.T, outputFile, failedOutputFile string) (*Pipeline, func()) {
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

func TestPipeline_WriteSuccess(t *testing.T) {

	tests := map[string]struct {
		OutputFilepath string
		In             []*UploadOut
		ExpectedCSV    [][]string
	}{
		"valid output": {
			OutputFilepath: "test1.csv",
			In: []*UploadOut{
				{
					Url:   "https://some-url.com/1.jpg",
					Key:   "1",
					Ext:   "jpg",
					S3url: "https://some-bucket.s3.amazonaws.com/1-converted.jpg",
				},
			},
			ExpectedCSV: [][]string{
				OutputHeader,
				{"https://some-url.com/1.jpg", "/outputs/1.jpg", "/outputs/1-converted.jpg", "https://some-bucket.s3.amazonaws.com/1-converted.jpg"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, teardown := WriteSetup(t, test.OutputFilepath, "")
			defer teardown()

			uploadOut := make(chan *UploadOut, len(test.In))

			for _, v := range test.In {
				uploadOut <- v
			}

			close(uploadOut)

			done := make(chan bool)
			go WriteSuccess(done, uploadOut, p.config.OutputFilepath)

			<-done

			assertCSVOutputFile(t, p.config.OutputFilepath, test.ExpectedCSV)

		})
	}
}

func TestPipeline_WriteError(t *testing.T) {

	tests := map[string]struct {
		FailedOutputFilepath string
		In                   []*ErrOut
		ExpectedCSV          [][]string
	}{
		"valid output": {
			FailedOutputFilepath: "test2.csv",
			In: []*ErrOut{
				{
					Url: "https://some-url.com/1.jpg",
					Err: fmt.Errorf("some error"),
				},
			},
			ExpectedCSV: [][]string{
				FailedOutputHeader,
				{"https://some-url.com/1.jpg"},
			},
		},
		"multiple lines": {
			FailedOutputFilepath: "test3.csv",
			In: []*ErrOut{
				{
					Url: "https://some-url.com/1.jpg",
					Err: fmt.Errorf("some error"),
				},
				{
					Url: "https://some-url.com/2.jpg",
					Err: fmt.Errorf("some other error"),
				},
			},
			ExpectedCSV: [][]string{
				FailedOutputHeader,
				{"https://some-url.com/1.jpg"},
				{"https://some-url.com/2.jpg"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, teardown := WriteSetup(t, "", test.FailedOutputFilepath)
			defer teardown()

			errOut := make(chan *ErrOut, len(test.In))

			for _, v := range test.In {
				errOut <- v
			}

			close(errOut)

			done := make(chan bool)
			go WriteError(done, errOut, p.config.FailedOutputFilepath)

			<-done

			assertCSVOutputFile(t, p.config.FailedOutputFilepath, test.ExpectedCSV)
		})
	}
}

func TestPipeline_Execute(t *testing.T) {

	setupSuite := func(t *testing.T, data [][]string) (*Pipeline, func()) {
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
				GetObject: func(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
					return &s3.GetObjectOutput{}, fmt.Errorf("404 not found")
				},
			},
		})

		p.maxRetries = 1
		p.tmpWorkingDir = tempDir

		p.config.OutputFilepath = path.Join(tempDir, "test.csv")
		p.config.FailedOutputFilepath = path.Join(tempDir, "failed.csv")

		return p, func() {
			os.RemoveAll(tempDir)
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
			{srv.URL + "/test.jpg", "/outputs/85e42ea4f380785dd6ae5c6361399d87.jpeg", "/outputs/85e42ea4f380785dd6ae5c6361399d87-converted.jpeg", "https://some-bucket.s3.amazonaws.com/85e42ea4f380785dd6ae5c6361399d87-converted.jpeg"},
			{srv.URL + "/test.gif", "/outputs/d5c423921f7654ad178b167897564b13.gif", "/outputs/d5c423921f7654ad178b167897564b13-converted.gif", "https://some-bucket.s3.amazonaws.com/d5c423921f7654ad178b167897564b13-converted.gif"},
			{srv.URL + "/test.png", "/outputs/e5765cefe1b8d33fc78315437516439d.png", "/outputs/e5765cefe1b8d33fc78315437516439d-converted.png", "https://some-bucket.s3.amazonaws.com/e5765cefe1b8d33fc78315437516439d-converted.png"},
		}

		wantFailedOutputCSV := [][]string{
			FailedOutputHeader,
			{srv.URL + "/test.txt"},
			{srv.URL + "/not-found"},
		}

		p, teardown := setupSuite(t, inputCSV)
		defer teardown()

		p.Execute()

		assertCSVOutputFile(t, p.config.FailedOutputFilepath, wantFailedOutputCSV)
		assertCSVOutputFile(t, p.config.OutputFilepath, wantOutputCSV)
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

func invalidHostErr(t *testing.T, url string) error {
	t.Helper()
	_, err := http.Get(url)
	if err != nil {
		return err
	}
	return nil
}
