package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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
	maxRetries uint64    // Max number of retries for downloading and uploading an image
	urlMap     *sync.Map // this will record urls that are already processed so we don't download again.
	errOut     chan *ErrOut

	// channels
	readOut     chan *ReadOut
	downloadOut chan *DownloadOut
	convertOut  chan *ConvertOut
	uploadOut   chan *UploadOut
}

func (p *Pipeline) Read(csvReader *csv.Reader, out chan *ReadOut) {
	// Read the input CSV file
	// For each line, send the URL to the download channel
	defer close(out)

	csvReader.FieldsPerRecord = 1

	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			p.errOut <- &ErrOut{Err: err}
			continue
		}

		// check if URL was already read
		_, ok := p.urlMap.Load(row[0])
		if ok {
			p.errOut <- &ErrOut{Err: ErrDuplicateURL}
			continue
		}

		p.urlMap.Store(row[0], true)
		out <- &ReadOut{Url: row[0]}
	}
	log.Println("Read done")
}

func (p *Pipeline) Download(wg *sync.WaitGroup, in <-chan *ReadOut, out chan *DownloadOut) {
	// Download the image from the URL
	// Send the image to the convert channel
	defer wg.Done()
	var (
		hash string
		ext  string
	)
	for row := range in {
		md5 := md5.Sum([]byte(row.Url))
		urlHash := hex.EncodeToString(md5[:])
		inputPath := fmt.Sprintf("/outputs/%s", urlHash)
		log.Println("input path:", inputPath)
		file, err := os.Create(inputPath)
		if err != nil {
			p.errOut <- &ErrOut{
				Url: row.Url,
				Err: err,
			}
			continue
		}
		hash, ext, err = DownloadWithBackoff(row.Url, p.maxRetries, file)
		file.Close()
		if err != nil {
			os.Remove(inputPath)
			p.errOut <- &ErrOut{
				Url: row.Url,
				Err: err,
			}
			continue
		}

		err = os.Rename(inputPath, InputPath(hash, ext))

		if err != nil {
			log.Println(err)
		}

		s3key := fmt.Sprintf("%s-converted.%s", hash, ext)

		// Check if the image hash is already in the S3 bucket
		_, err = p.config.Aws.GetObject(&s3.GetObjectInput{
			Bucket: &p.config.Aws.s3bucket,
			Key:    &s3key,
		})

		if err == nil {
			os.Remove(inputPath)
			p.errOut <- &ErrOut{
				Url: row.Url,
				Key: hash,
				Ext: ext,
				Err: ErrImageExistsInS3,
			}
			continue
		}

		out <- &DownloadOut{
			Url: row.Url,
			Key: hash,
			Ext: ext,
		}
	}
}

func (p *Pipeline) Convert(wg *sync.WaitGroup, in <-chan *DownloadOut, out chan *ConvertOut) {
	// Convert the image
	// Send the image to the upload channel
	defer wg.Done()
	for row := range in {
		err := p.config.Converter.Grayscale(InputPath(row.Key, row.Ext), OutputPath(row.Key, row.Ext))
		if err != nil {
			p.errOut <- &ErrOut{
				Url: row.Url,
				Key: row.Key,
				Ext: row.Ext,
				Err: fmt.Errorf("error converting image: %v", err),
			}
			continue
		}
		out <- &ConvertOut{
			Url: row.Url,
			Key: row.Key,
			Ext: row.Ext,
		}
	}
}

func (p *Pipeline) Upload(wg *sync.WaitGroup, in <-chan *ConvertOut, out chan *UploadOut) {
	// Upload the image to S3
	// Send the image to the result channel
	defer wg.Done()
	for row := range in {

		file, err := os.Open(OutputPath(row.Key, row.Ext))

		if err != nil {
			p.errOut <- &ErrOut{
				Url: row.Url,
				Key: row.Key,
				Ext: row.Ext,
				Err: fmt.Errorf("failed to open file: %v", err),
			}
			continue
		}

		key := fmt.Sprintf("%s-converted.%s", row.Key, row.Ext)

		err = UploadToS3WithBackoff(file, key, p.config.Aws, p.maxRetries)

		if err != nil {
			p.errOut <- &ErrOut{
				Url: row.Url,
				Key: row.Key,
				Ext: row.Ext,
				Err: fmt.Errorf("failed to upload to S3: %v", err),
			}
			continue
		}

		out <- &UploadOut{
			Url:   row.Url,
			Key:   row.Key,
			Ext:   row.Ext,
			S3url: fmt.Sprintf("https://%s.s3.amazonaws.com/%s", p.config.Aws.s3bucket, key),
		}
	}
}

func WriteSuccess(done chan bool, in <-chan *UploadOut, OutputFilepath string) {

	f, err := os.Create(OutputFilepath)
	if err != nil {
		log.Fatalf("error creating output file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write(OutputHeader)

	for row := range in {
		w.Write([]string{row.Url, InputPath(row.Key, row.Ext), OutputPath(row.Key, row.Ext), row.S3url})
	}
	done <- true
}

func WriteError(done chan bool, in <-chan *ErrOut, ErrorFilepath string) {

	if ErrorFilepath == "" {
		for range in {
			// do nothing, only unblock the channel

		}
		done <- true
		return
	}

	f, err := os.Create(ErrorFilepath)
	if err != nil {
		log.Fatalf("error creating error file: %v", err)
	}

	defer f.Close()

	w := csv.NewWriter(f)

	defer w.Flush()

	w.Write(FailedOutputHeader)

	for row := range in {
		fmt.Println("Received error: ", row.Err)
		w.Write([]string{row.Url, row.Key, row.Ext, row.Err.Error()})
	}
	done <- true
}

func (p *Pipeline) closeChannel(task string) {
	switch task {
	case DOWNLOAD:
		close(p.downloadOut)
	case CONVERT:
		close(p.convertOut)
	case UPLOAD:
		close(p.uploadOut)
	}
}

func (p *Pipeline) Execute() error {

	start := time.Now()

	doneSuccess := make(chan bool)
	go WriteSuccess(doneSuccess, p.uploadOut, p.config.OutputFilepath)
	doneError := make(chan bool)
	go WriteError(doneError, p.errOut, p.config.FailedOutputFilepath)

	// Start the workers
	wgTasks := &sync.WaitGroup{}
	for _, task := range TASKS {
		wgTasks.Add(1)
		go func(task string) {
			fmt.Printf("Executing task: %s, with %d workers\n", task, p.workers[task])
			wg := &sync.WaitGroup{}
			wg.Add(p.workers[task])
			for i := 0; i < p.workers[task]; i++ {
				switch task {
				case DOWNLOAD:
					go p.Download(wg, p.readOut, p.downloadOut)
				case CONVERT:
					go p.Convert(wg, p.downloadOut, p.convertOut)
				case UPLOAD:
					go p.Upload(wg, p.convertOut, p.uploadOut)
				default:
					log.Fatalf("unknown task: %s", task)
				}
			}
			wg.Wait()
			p.closeChannel(task)
			wgTasks.Done()
		}(task)
	}

	// Start the reader
	csvReader, err := OpenCSVFile(p.config.InputFilepath)

	if err != nil {
		return err
	}
	go p.Read(csvReader, p.readOut)

	// Wait for all tasks to finish
	wgTasks.Wait()
	close(p.errOut)
	elapsed := time.Since(start)

	log.Printf("Finished in %s", elapsed)

	<-doneSuccess
	<-doneError

	return nil
}

func NewPipeline(config *Config) *Pipeline {
	return &Pipeline{
		config:     config,
		maxRetries: 3,
		urlMap:     &sync.Map{},
		workers: map[string]int{
			DOWNLOAD: 10,
			CONVERT:  3, // this sometimes fails due to C binding issue. In case of failure, reduce to 1
			UPLOAD:   10,
		},
		readOut:     make(chan *ReadOut),
		downloadOut: make(chan *DownloadOut),
		convertOut:  make(chan *ConvertOut),
		uploadOut:   make(chan *UploadOut),
		errOut:      make(chan *ErrOut, 5),
	}
}
