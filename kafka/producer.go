package kafka

import "github.com/confluentinc/confluent-kafka-go/kafka"

type KafkaProducer struct {
	Producer *kafka.Producer
}

func GetProducerConfigMap() *kafka.ConfigMap {
	return &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "myProducer",
		"acks":              "all",
		"security.protocol": "SSL",
	}
}

func (p *KafkaProducer) Initialize() error {
	var err error
	p.Producer, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "myProducer",
		"acks":              "all"})
	return err
}
