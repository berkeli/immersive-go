package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"encoding/json"

	"github.com/Shopify/sarama"
	"github.com/berkeli/kafka-cron/types"
	"github.com/google/uuid"
)

var (
	brokerList = kingpin.Flag("brokerList", "List of brokers to connect").Default("localhost:9092").Strings()
	topic      = kingpin.Flag("topic", "Topic name").Default("jobs-cluster-a").String()
	retryTopic = kingpin.Flag("retryTopic", "Retry topic name").Default(*topic + "-retries").String()
	partition  = kingpin.Flag("partition", "Partition number").Default("0").Int32()
)

func main() {

	InitMonitoring(2112)

	kingpin.Parse()
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	brokers := *brokerList
	master, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Panic(err)
	}

	config = sarama.NewConfig()
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

	defer func() {
		if err := master.Close(); err != nil {
			log.Panic(err)
		}
	}()
	consumer, err := master.ConsumePartition(*topic, *partition, sarama.OffsetNewest)
	if err != nil {
		log.Panic(err)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	chDone := make(chan bool)

	go func() {
		for {
			select {
			case err := <-consumer.Errors():
				log.Println(err)
				ErrorCounter.WithLabelValues(*topic, "read-message").Inc()
			case msg := <-consumer.Messages():
				var cmd types.Command
				err := json.Unmarshal(msg.Value, &cmd)
				if err != nil {
					log.Println(err)
					ErrorCounter.WithLabelValues(*topic, "json-unmarshall").Inc()
				}

				duration := time.Since(msg.Timestamp)

				JobQueueTime.WithLabelValues(*topic, cmd.Description).Observe(duration.Seconds())
				processCommand(producer, cmd)
			case <-signals:
				chDone <- true
				return
			}
		}
	}()
	<-chDone
}

func processCommand(producer sarama.SyncProducer, cmd types.Command) {
	log.Println("Starting a job for: ", cmd.Description)
	//metrics
	status := "success"
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		fmt.Println("Job took: ", duration.Seconds(), " seconds")
		JobDuration.WithLabelValues(*topic, cmd.Description, status).Observe(duration.Seconds())
	}()

	JobsTotal.WithLabelValues(*topic, cmd.Description).Inc()
	JobsInFlight.WithLabelValues(*topic, cmd.Description).Inc()
	defer JobsInFlight.WithLabelValues(*topic, cmd.Description).Dec()

	out, err := executeCommand(cmd.Command)
	if err != nil {
		status = "failed"
		log.Printf("Command: %s resulted in error: %s\n", cmd.Command, err)
		if cmd.MaxRetries > 0 {
			cmd.MaxRetries--
			log.Printf("Retrying command: %s, %d retries left\n", cmd.Command, cmd.MaxRetries)
			cmdBytes, err := json.Marshal(cmd)

			if err != nil {
				log.Println(err)
				ErrorCounter.WithLabelValues(*topic, "json-marshall").Inc()
				return
			}
			_, _, err = producer.SendMessage(&sarama.ProducerMessage{
				Topic: *retryTopic,
				Key:   sarama.StringEncoder(uuid.New().String()),
				Value: sarama.ByteEncoder(cmdBytes),
			})

			if err != nil {
				log.Println(err)
				ErrorCounter.WithLabelValues(*topic, "publish-retry").Inc()
				return
			}
			JobsPublished.WithLabelValues(*retryTopic, cmd.Description).Inc()
			JobsRetried.WithLabelValues(*topic, cmd.Description).Inc()
		} else {
			JobsFailed.WithLabelValues(*topic, cmd.Description).Inc()
			ErrorCounter.WithLabelValues(*topic, "execution").Inc()
			log.Printf("Command: %s failed, no more retries left\n", cmd.Command)
		}
	}
	JobsProcessed.WithLabelValues(*topic, cmd.Description).Inc()
	log.Println(out)
}

func executeCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	err := cmd.Run()

	if err != nil {
		log.Println(outErr.String())
	}
	return out.String(), err
}
