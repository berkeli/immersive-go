package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	READ     = "read"
	DOWNLOAD = "download"
	CONVERT  = "convert"
	UPLOAD   = "upload"
	WRITE    = "write"
)

var TASKS = []string{DOWNLOAD, CONVERT, UPLOAD}

var (
	OutputHeader       = []string{"url", "input", "output", "s3url", "error"}
	FailedOutputHeader = []string{"url"}
)

var (
	ErrDuplicateURL    = errors.New("duplicate URL, skipping")
	ErrImageExistsInS3 = errors.New("image already exists in S3, skipping")
)

// Pipeline struct
type Pipeline struct {
	config     *Config
	workers    map[string]int
	channels   map[string]chan *Out
	maxRetries uint64    // Max number of retries for downloading and uploading an image
	urlMap     *sync.Map // this will record urls that are already processed so we don't download again.
}

type Task func(wg *sync.WaitGroup)

func (p *Pipeline) Read(csvReader *csv.Reader) {
	// Read the input CSV file
	// For each line, send the URL to the download channel
	defer close(p.channels[READ])

	csvReader.FieldsPerRecord = 1

	for {
		row, err := csvReader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			p.channels[READ] <- &Out{Err: err}
			continue
		}
		p.channels[READ] <- &Out{Url: row[0]}
	}
	log.Println("Read done")
}

func (p *Pipeline) Download(wg *sync.WaitGroup) {
	// Download the image from the URL
	// Send the image to the convert channel
	defer wg.Done()
	var (
		row  *Out
		ok   bool
		body io.Reader
		hash string
		ext  string
		err  error
	)
	for {

		row, ok = <-p.channels[READ]
		if !ok {
			break
		}
		if row.Err != nil {
			p.channels[DOWNLOAD] <- row
			continue
		}

		// Check if the URL has already been processed
		// If it has, skip it
		_, ok := p.urlMap.Load(row.Url)
		if ok {
			p.channels[DOWNLOAD] <- &Out{
				Url: row.Url,
				Err: ErrDuplicateURL,
			}
			continue
		} else {
			p.urlMap.Store(row.Url, true)
		}

		body, ext, hash, err = DownloadWithBackoff(row.Url, p.maxRetries)

		if err != nil {
			row.Err = err
			p.channels[DOWNLOAD] <- row
			continue
		}

		key := fmt.Sprintf("%s-converted.%s", hash, ext)

		// Check if the image hash is already in the S3 bucket
		_, err = p.config.Aws.GetObject(&s3.GetObjectInput{
			Bucket: &p.config.Aws.s3bucket,
			Key:    &key,
		})

		if err == nil {
			row.Err = ErrImageExistsInS3
			row.S3url = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", p.config.Aws.s3bucket, hash)
			p.channels[DOWNLOAD] <- row
			continue
		}

		//Create an empty file
		inputPath := fmt.Sprintf("/outputs/%s.%s", hash, ext)
		outputPath := fmt.Sprintf("/outputs/%s", key)

		file, err := os.Create(inputPath)
		if err != nil {
			p.channels[DOWNLOAD] <- &Out{
				Url: row.Url,
				Err: err,
			}
			continue
		}
		defer file.Close()

		//Write bytes to the file
		_, err = io.Copy(file, body)
		if err != nil {
			p.channels[DOWNLOAD] <- &Out{
				Url: row.Url,
				Err: err,
			}
			continue
		}

		p.channels[DOWNLOAD] <- &Out{
			Url:    row.Url,
			Input:  inputPath,
			Output: outputPath,
		}
	}
}

func (p *Pipeline) Convert(wg *sync.WaitGroup) {
	// Convert the image
	// Send the image to the upload channel
	defer wg.Done()
	for {
		row, ok := <-p.channels[DOWNLOAD]
		if !ok {
			break
		}
		if row.Err != nil {
			p.channels[CONVERT] <- &Out{
				Url:    row.Url,
				Err:    row.Err,
				Input:  row.Input,
				Output: "",
			}
			continue
		}
		err := p.config.Converter.Grayscale(row.Input, row.Output)
		if err != nil {
			p.channels[CONVERT] <- &Out{
				Url:    row.Url,
				Err:    err,
				Input:  row.Input,
				Output: "",
			}
			continue
		}
		p.channels[CONVERT] <- row
	}
}

func (p *Pipeline) Upload(wg *sync.WaitGroup) {
	// Upload the image to S3
	// Send the image to the result channel
	defer wg.Done()
	for {
		row, ok := <-p.channels[CONVERT]
		if !ok {
			break
		}

		if row.Err != nil {
			p.channels[UPLOAD] <- row
			continue
		}

		file, err := os.Open(row.Output)

		if err != nil {
			row.Err = err
			p.channels[UPLOAD] <- row
			continue
		}

		// Upload to S3
		path := strings.Split(row.Output, "/")
		key := path[len(path)-1]

		err = UploadToS3WithBackoff(file, key, p.config.Aws, p.maxRetries)

		if err != nil {
			row.Err = err
			p.channels[UPLOAD] <- row
			continue
		}

		row.S3url = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", p.config.Aws.s3bucket, key)

		p.channels[UPLOAD] <- row
	}
}

func (p *Pipeline) Write(done chan bool) {

	f, err := os.Create(p.config.OutputFilepath)
	if err != nil {
		log.Fatalf("error creating output file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write(OutputHeader)
	var failedW *csv.Writer

	// Create csv for failed output if param is set
	if p.config.FailedOutputFilepath != "" {
		failedF, err := os.Create(p.config.FailedOutputFilepath)
		if err != nil {
			log.Fatalf("error creating failed output file: %v", err)
		}
		defer failedF.Close()

		failedW = csv.NewWriter(failedF)
		defer failedW.Flush()
		failedW.Write(FailedOutputHeader)
	}

	for {
		row, ok := <-p.channels[UPLOAD]

		if !ok {
			break
		}
		rowErr := ""
		// only add to failed output if row has an error, but not a duplicate error or image exists error
		if row.Err != nil {
			if p.config.FailedOutputFilepath != "" && row.Err != ErrDuplicateURL && row.Err != ErrImageExistsInS3 {
				failedW.Write([]string{row.Url})
			}
			rowErr = row.Err.Error()
		}

		w.Write([]string{row.Url, row.Input, row.Output, row.S3url, rowErr})
	}
	done <- true
}

func (p *Pipeline) taskPicker(task string) Task {
	switch task {
	case DOWNLOAD:
		return p.Download
	case CONVERT:
		return p.Convert
	case UPLOAD:
		return p.Upload
	}

	return nil
}

func (p *Pipeline) Execute() error {

	start := time.Now()

	done := make(chan bool)
	go p.Write(done)

	// Start the workers
	wgTasks := &sync.WaitGroup{}
	for _, task := range TASKS {
		wgTasks.Add(1)
		go func(task string) {
			var wg sync.WaitGroup
			for i := 0; i < p.workers[task]; i++ {
				wg.Add(1)
				go p.taskPicker(task)(&wg)
			}
			wg.Wait()
			close(p.channels[task])
			wgTasks.Done()
		}(task)
	}

	// Start the reader
	csvReader, err := OpenCSVFile(p.config.InputFilepath)

	if err != nil {
		return err
	}
	go p.Read(csvReader)

	// Wait for all tasks to finish
	wgTasks.Wait()
	elapsed := time.Since(start)

	log.Printf("Finished in %s", elapsed)

	<-done

	return nil
}

func NewPipeline(config *Config) *Pipeline {
	return &Pipeline{
		config:     config,
		maxRetries: 3,
		urlMap:     &sync.Map{},
		channels: map[string]chan *Out{
			READ:     make(chan *Out),
			DOWNLOAD: make(chan *Out),
			CONVERT:  make(chan *Out),
			UPLOAD:   make(chan *Out),
		},
		workers: map[string]int{
			DOWNLOAD: 10,
			CONVERT:  3, // this sometimes fails due to C binding issue. In case of failure, reduce to 1
			UPLOAD:   10,
		},
	}
}
