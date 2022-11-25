package uploader

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type AWSConfig struct {
	region   string
	s3bucket string

	PutObject func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	GetObject func(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

func InitAwsClient() (*AWSConfig, error) {
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
