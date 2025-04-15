package kafka

import (
	"context"
	"fmt"
	DB "notify/database"
	"notify/logs"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func GetConsumerConfigMap(groupId string) *kafka.ConfigMap {
	return &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          groupId,
		"auto.offset.reset": "smallest",
		// "security.protocol": "SSL",
	}
}

func pushMessageIntoCLientQueues(p *kafka.Producer, e *kafka.Message) {
	logger := logs.GetLogger()
	logger.Info("pushMessageIntoCLientQueues called for event ==> ", e)
	consumersCount, err := DB.GetConsumersCount()
	if err != nil {
		logger.Error("failed to get client consumers count", err)
	}
	for i := 1; i <= consumersCount; i++ {
		clientTopic := "client-group-" + strconv.Itoa(i)
		err = p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &clientTopic,
				Partition: kafka.PartitionAny,
			},
			Value: e.Value,
			Key:   e.Key,
		}, nil)
		if err != nil {
			logger.Error("failed to push message into client queue ", clientTopic, "Error: ", err)
		}
	}
}

func waitForTopicReady(admin *kafka.AdminClient, topic string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	logger := logs.GetLogger()
	for time.Now().Before(deadline) {
		logger.Println("waitForTopicReady Called inside for loop")

		md, err := admin.GetMetadata(&topic, false, 5000)

		logger.Println(md.Topics[topic], err)

		if err == nil {
			if _, exists := md.Topics[topic]; exists {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	// logger.Errorln("timeout: topic", topic, "not available")
	return fmt.Errorf("timeout: topic %s not available", topic)
}

func startConsumer(groupId string, topics []string) {
	logger := logs.GetLogger()
	consumerConfigMap := GetConsumerConfigMap(groupId)
	consumer, err := kafka.NewConsumer(consumerConfigMap)
	if err != nil {
		logger.Error(err.Error())
	}
	logger.Info("Start topic consumer called for topics: ", topics)
	kafkaAdminClient, err := kafka.NewAdminClientFromConsumer(consumer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = kafkaAdminClient.CreateTopics(ctx, []kafka.TopicSpecification{
		{
			Topic:             topics[0],
			NumPartitions:     1,
			ReplicationFactor: 1},
	})
	if err != nil {
		logger.Error("Failed to create a new topic: ", err)
	}
	logger.Info("Successfully created a new topic: ", topics[0])
	topicsToSubscribe := topics
	err = consumer.SubscribeTopics(topicsToSubscribe, nil)
	if err != nil {
		logger.Errorln("Error while subscribing to topic", topics, "\nError: ", err)
	}
	producer, err := kafka.NewProducer(GetProducerConfigMap())
	if err != nil {
		logger.Error("Failed to create producer for pushing messages into userQueue")
	}

	run := true
	for run {
		ev := consumer.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			{
				logger.Info("Message received by topic consumer", e.Value)
				pushMessageIntoCLientQueues(producer, e)

			}
		case kafka.Error:
			logger.Error("error while reading message from topics", e)
			run = false
		}
	}
	defer consumer.Close()
}

func StartConsumers() {
	logger := logs.GetLogger()
	notifyDB, err := DB.GetDB()
	if err != nil {
		logger.Error(err)
		return
	}
	rows, err := notifyDB.Query(`SELECT * FROM topics`)
	if err != nil {
		logger.Error("Unable to query DB while starting consumers..", err)
		return
	}
	var topics []string
	for rows.Next() {
		var topic DB.TopicConfig
		err := rows.Scan(&topic.TopicName)
		if err != nil {
			logger.Error(err)
			return
		}
		topics = append(topics, topic.TopicName)
	}
	for _, t := range topics {
		logger.Printf("Topic: %s\n", t)
		go startConsumer("t", []string{t})
	}
}
