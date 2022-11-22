package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/gographics/imagick.v2/imagick"
)

type ConvertImageCommand func(args []string) (*imagick.ImageCommandResult, error)

type Converter struct {
	cmd ConvertImageCommand
}

func (c *Converter) Grayscale(inputFilepath string, outputFilepath string) error {
	// Convert the image to grayscale using imagemagick
	// We are directly calling the convert command
	_, err := c.cmd([]string{
		"convert", inputFilepath, "-set", "colorspace", "Gray", outputFilepath,
	})
	return err
}

type ReadOut struct {
	Url string
}

type DownloadOut struct {
	Url string
	Key string
	Ext string
}

type ConvertOut struct {
	Url string
	Key string
	Ext string
}

type UploadOut struct {
	Url   string
	Key   string
	Ext   string
	S3url string
}

func (r *ConvertOut) AwsKey() string {
	return fmt.Sprintf("%s-converted.%s", r.Key, r.Ext)
}

type ErrOut struct {
	Url string
	Key string
	Ext string
	Err error
}

type AWSConfig struct {
	region   string
	s3bucket string

	PutObject func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	GetObject func(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}
