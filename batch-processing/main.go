package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

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
		region:    awsRegion,
		s3bucket:  s3Bucket,
		PutObject: S3Client.PutObject,
		GetObject: S3Client.GetObject,
	}, nil
}

type Config struct {
	InputFilepath        string
	OutputFilepath       string
	FailedOutputFilepath string
	Converter            *Converter
	Aws                  *AWSConfig
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

	// start monitoring the pipeline
	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	config := &Config{
		InputFilepath:        *inputFilepath,
		OutputFilepath:       *outputFilepath,
		FailedOutputFilepath: *failedOutputFilepath,
		Converter:            c,
		Aws:                  a,
	}

	// Run the pipeline
	p := NewPipeline(config)

	p.Execute()

	// Log what we did
	log.Printf("processed: %q to %q\n", *inputFilepath, *outputFilepath)

	if *failedOutputFilepath != "" {
		log.Printf("failed: %q\n", *failedOutputFilepath)
	}
}
