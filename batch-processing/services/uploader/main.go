package uploader

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	u "github.com/berkeli/immersive-go/batch-processing/utils"
	"github.com/segmentio/kafka-go"
)

type Config struct {
	KafkaBrokers []string
	InTopic      string
	OutTopic     string
	OutputPath   string
}

type UploaderService struct {
	config *Config

	aws    *AWSConfig
	pub    u.Publisher
	errPub u.Publisher
}

func NewUploaderService(config *Config) *UploaderService {
	return &UploaderService{
		config: config,
	}
}

func (us *UploaderService) Run(ctx context.Context) error {
	a, err := InitAwsClient()

	if err != nil {
		return err
	}

	us.aws = a

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: us.config.KafkaBrokers,
		Topic:   us.config.InTopic,
	})

	defer r.Close()

	kfkW := &kafka.Writer{
		Addr:                   kafka.TCP(us.config.KafkaBrokers...),
		AllowAutoTopicCreation: true,
	}

	us.pub = u.GetPublisher(kfkW, us.config.OutTopic)
	us.errPub = u.GetPublisher(kfkW, "errors")

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			return err
		}

		log.Printf("Uploading %s", string(m.Value))

		f, err := os.Open(fmt.Sprintf("%s/%s", us.config.OutputPath, m.Value))

		if err != nil {
			log.Println("error opening file: ", err)
			us.errPub(m.Key, m.Value, m.Headers)
			continue
		}

		_, err = us.aws.PutObject(&s3.PutObjectInput{
			Bucket: aws.String("batch-processing-berkeli"),
			Key:    aws.String(string(m.Value)),
			Body:   f,
		})

		if err != nil {
			log.Println("error uploading file: ", err)
			us.errPub(m.Key, []byte(fmt.Sprintf("error uploading file: %s", err)), m.Headers)
		}

		headers := []kafka.Header{
			{Key: "input", Value: []byte(strings.Replace(string(m.Value), "converted_", "", 1))},
			{Key: "output", Value: m.Value},
		}

		err = us.pub(m.Key, []byte(fmt.Sprintf("https://%s.s3.amazonaws.com/%s", us.aws.s3bucket, string(m.Value))), headers)

		if err != nil {
			return err
		}

	}
}
