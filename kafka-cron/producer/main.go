package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/Shopify/sarama"
	"github.com/berkeli/kafka-cron/types"
	"github.com/berkeli/kafka-cron/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/robfig/cron/v3"
)

var (
	brokerList = kingpin.Flag("brokerList", "List of brokers to connect").Default("localhost:9092").Strings()
	configPath = kingpin.Flag("configPath", "Path to config file").Default("./cron.yaml").String()
)

func main() {
	InitMonitoring(2112)

	kingpin.Parse()
	config := sarama.NewConfig()

	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(*brokerList, config)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			log.Panic(err)
		}
	}()
	cmds, err := ReadCrons(*configPath)

	if err != nil {
		log.Panic(err)
	}
	err = ScheduleJobs(producer, cmds)

	if err != nil {
		log.Panic(err)
	}

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
}

func ReadCrons(path string) ([]types.Command, error) {
	cnf, err := utils.ReadConfig(path)

	if err != nil {
		return nil, err
	}
	// with golang unset value defaults to 0 so we need to set it to 3
	// not ideal, but there's a comment to set max retries to -1 to disable retries
	if cnf.MaxAllowedRetries == 0 {
		cnf.MaxAllowedRetries = 3
	}

	var validationErrors error

	allowedClusters := make(map[string]struct{})

	for _, cluster := range cnf.Clusters {
		allowedClusters[cluster.Name] = struct{}{}
	}

	for _, cmd := range cnf.Crons {
		err := cmd.Validate(allowedClusters, cnf.MaxAllowedRetries)

		if err != nil {
			multierror.Append(err, validationErrors)
		}
	}

	if validationErrors != nil {
		return nil, validationErrors
	}

	return cnf.Crons, nil
}

func ScheduleJobs(prod sarama.SyncProducer, cmds []types.Command) error {
	c := cron.New()
	for _, cmd := range cmds {
		fmt.Println("scheduling: ", cmd.Description)

		sch, err := cron.ParseStandard(cmd.Schedule)

		if err != nil {
			return err
		}

		job := CommandPublisher{
			Command:   cmd,
			publisher: prod,
		}

		c.Schedule(sch, &job)
		ScheduledCrons.Inc()
	}
	c.Start()
	return nil
}

func PublishMessages(prod sarama.SyncProducer, msg string, clusters []string) error {
	getTopic := utils.WithTopicPrefix()
	for _, cluster := range clusters {
		topic := getTopic(cluster)
		msg := &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(uuid.New().String()),
			Value: sarama.StringEncoder(msg),
		}
		_, _, err := prod.SendMessage(msg)
		if err != nil {
			QueuedJobs.WithLabelValues(topic, "error").Inc()
			return err
		}
		QueuedJobs.WithLabelValues(topic, "success").Inc()
	}
	return nil
}

type CommandPublisher struct {
	types.Command
	publisher sarama.SyncProducer
}

func (c *CommandPublisher) Run() {
	fmt.Println("Running command: ", c.Description)
	msgString, err := json.Marshal(&c)
	if err != nil {
		log.Println(fmt.Errorf("error marshalling command: %v", err))
	}
	err = PublishMessages(c.publisher, string(msgString), c.Clusters)

	if err != nil {
		log.Println(fmt.Errorf("error publishing command: %v", err))
	}
}
