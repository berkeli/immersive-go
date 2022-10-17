package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/gographics/imagick.v2/imagick"
)

func initAwsClient() (*AWSConfig, error) {
	awsRoleArn := os.Getenv("AWS_ROLE_ARN")
	if awsRoleArn == "" {
		return nil, fmt.Errorf("AWS_ROLE_ARN is not set")
	}
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		return nil, fmt.Errorf("AWS_REGION is not set")
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is not set")
	}

	sess := session.Must(session.NewSession())

	creds := stscreds.NewCredentials(sess, awsRoleArn)

	// Create a new S3 client
	S3Client := s3.New(sess, &aws.Config{Credentials: creds})

	return &AWSConfig{
		s3:       S3Client,
		region:   awsRegion,
		s3bucket: s3Bucket,
	}, nil
}

func main() {
	// Accept --input and --output arguments for the images
	inputFilepath := flag.String("input", "", "A path to a CSV file containing image URLs to be processed")
	outputFilepath := flag.String("output", "", "A path to where the CSV file with processed image URLs should be written")
	failedOutputFilepath := flag.String("output-failed", "", "A path to where the CSV file with failed image URLs should be written (optional)")
	flag.Parse()

	// Ensure that both flags were set
	if *inputFilepath == "" || *outputFilepath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Create a new session
	a, err := initAwsClient()

	if err != nil {
		log.Fatal(err)
	}
	// Set up imagemagick
	imagick.Initialize()
	defer imagick.Terminate()

	// Log what we're going to do
	log.Printf("processing: %q to %q\n", *inputFilepath, *outputFilepath)

	// Build a Converter struct that will use imagick
	c := &Converter{
		cmd: imagick.ConvertImageCommand,
	}

	reader := ReadCSV(inputFilepath)
	result := make(chan *Output)

	wg := &sync.WaitGroup{}

	// Start a goroutine to write the results to a CSV file. Will be consuming output from the result channel in parallel
	go ResultToCSV(result, *outputFilepath, *failedOutputFilepath)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result <- &Output{url: row[0], err: err}
			continue
		}
		url := row[0]
		wg.Add(1)
		// each row is processed in a go routine
		go ProcessRow(url, *c, result, wg, a)
	}
	wg.Wait()
	close(result)
	// Log what we did
	log.Printf("processed: %q to %q\n", *inputFilepath, *outputFilepath)

	if *failedOutputFilepath != "" {
		log.Printf("failed: %q\n", *failedOutputFilepath)
	}
}
