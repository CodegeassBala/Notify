package kafka

import (
	DB "notify/database"
	"notify/logs"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func GetConsumerConfigMap(groupId string) *kafka.ConfigMap {
	return &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          groupId,
		"auto.offset.reset": "smallest"}
}

func pushMessageIntoCLientQueues(p *kafka.Producer, e *kafka.Message) {
	logger := logs.GetLogger()
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

func startConsumer(groupId string, topics []string) {
	logger := logs.GetLogger()
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          groupId,
		"auto.offset.reset": "smallest"})
	if err != nil {
		logger.Error(err.Error())
	}
	topicsToSubscribe := topics
	consumer.SubscribeTopics(topicsToSubscribe, nil)
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"acks":              "all",
	})
	if err != nil {
		logger.Error("Failed to create producer for pushing messages into userQueue")
	}

	run := true
	for run {
		ev := consumer.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			{
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
