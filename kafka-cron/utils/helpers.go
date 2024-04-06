package utils

import (
	"fmt"
	"os"

	"github.com/berkeli/kafka-cron/types"
	"github.com/goccy/go-yaml"
)

func WithTopicPrefix() func(string) string {
	prefix := os.Getenv("TOPIC_PREFIX")

	if prefix == "" {
		prefix = "jobs"
	}

	return func(cluster string) string {
		return fmt.Sprintf("%s-%s", prefix, cluster)
	}
}

func ReadConfig(path string) (*types.ConfigFile, error) {
	var cnf types.ConfigFile
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &cnf)
	if err != nil {
		return nil, err
	}

	return &cnf, nil
}
