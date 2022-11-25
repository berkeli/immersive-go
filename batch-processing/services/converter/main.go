package converter

import (
	"context"
	"fmt"

	u "github.com/berkeli/immersive-go/batch-processing/utils"
	"github.com/segmentio/kafka-go"
	"gopkg.in/gographics/imagick.v2/imagick"
)

const (
	ERR_TOPIC = "errors"
)

type Config struct {
	InTopic      string
	OutTopic     string
	OutputPath   string
	KafkaBrokers []string
}

type ConverterService struct {
	config *Config

	pub     u.Publisher
	errPub  u.Publisher
	convert *Converter
}

func NewConverterService(config *Config) *ConverterService {
	return &ConverterService{
		config: config,
	}
}

func (cs *ConverterService) Run(ctx context.Context) error {

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cs.config.KafkaBrokers,
		Topic:   cs.config.InTopic,
	})

	defer r.Close()

	kfk := &kafka.Writer{
		Addr: kafka.TCP(cs.config.KafkaBrokers...),
	}

	cs.pub = u.GetPublisher(kfk, cs.config.OutTopic)
	cs.errPub = u.GetPublisher(kfk, ERR_TOPIC)

	imagick.Initialize()
	defer imagick.Terminate()

	cs.convert = &Converter{
		cmd: imagick.ConvertImageCommand,
	}

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			return err
		}

		inputFile := fmt.Sprintf("%s/%s", cs.config.OutputPath, string(m.Value))
		outputFile := fmt.Sprintf("converted_%s", string(m.Value))

		err = cs.convert.Grayscale(inputFile, fmt.Sprintf("%s/%s", cs.config.OutputPath, outputFile))

		if err != nil {
			cs.errPub(m.Key, []byte(fmt.Sprintf("could not convert image: %s", err)), m.Headers)
			continue
		}

		err = cs.pub(m.Key, []byte(outputFile), nil)

		if err != nil {
			return err
		}
	}
}
