package main

import (
	"errors"
	"log"

	"github.com/Shopify/sarama"
	"github.com/berkeli/kafka-cron/types"
	"github.com/berkeli/kafka-cron/utils"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	brokerList = kingpin.Flag("brokerList", "List of brokers to connect").Default("localhost:9092").Strings()
	configPath = kingpin.Flag("configPath", "Path to config file").Default("./cron.yaml").String()
)

func main() {
	kingpin.Parse()
	config := sarama.NewConfig()

	admin, err := sarama.NewClusterAdmin(*brokerList, config)

	if err != nil {
		log.Panic(err)
	}

	clusters, err := GetClusters(*configPath)

	if err != nil {
		log.Panic(err)
	}

	err = CreateTopics(admin, clusters)

	if err != nil {
		log.Panic(err)
	}

	if err := admin.Close(); err != nil {
		log.Panic(err)
	}

	log.Println("Setup completed successfully")
}

func GetClusters(path string) ([]types.Cluster, error) {
	cnf, err := utils.ReadConfig(path)

	if err != nil {
		return nil, err
	}

	return cnf.Clusters, nil
}

func CreateTopics(admin sarama.ClusterAdmin, clusters []types.Cluster) error {
	getTopic := utils.WithTopicPrefix()
	for _, cluster := range clusters {
		topicName := getTopic(cluster.Name)

		rf := int16(cluster.Replication)
		if rf > int16(len(*brokerList)) {
			return errors.New("replication factor cannot be greater than number of brokers")
		}

		if rf == 0 {
			rf = 1
		}

		numPartitions := int32(cluster.Partitions)

		if numPartitions == 0 {
			numPartitions = 1
		}

		err := admin.CreateTopic(topicName, &sarama.TopicDetail{
			NumPartitions:     int32(cluster.Partitions),
			ReplicationFactor: rf,
		}, false)

		if err != nil {
			if errors.Is(err, sarama.ErrTopicAlreadyExists) {
				admin.CreatePartitions(topicName, int32(cluster.Partitions), nil, false)
			} else {
				return err
			}
		}
	}

	return nil
}
