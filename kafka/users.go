package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	DB "notify/database"
	"notify/database/sqlc"
	"notify/logs"
	"strconv"
	"sync"
	"time"

	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var CLIENTS_LIMIT int = 0
var MAX_CLIENTS_LIMIT int = 0
var WEBHOOK_URL string = ""

func initializeGlobalVariables() {
	CLIENTS_LIMIT, _ = strconv.Atoi(os.Getenv("CLIENTS_LIMIT"))
	WEBHOOK_URL = os.Getenv("WEBHOOK_URL")
	MAX_CLIENTS_LIMIT = CLIENTS_LIMIT + (CLIENTS_LIMIT+1)/2
}

type ClientConsumers struct {
	total_consumers int
	SPIN_UP_CHAN    chan bool
	mu              sync.Mutex
	queries         *sqlc.Queries
}

func (c *ClientConsumers) initialize() {
	logger := logs.GetLogger()
	db, err := DB.GetDBPool()
	if err != nil {
		logger.Error("failed to connect to db fromm the user-consumer", err)
	}
	c.queries = sqlc.New(db)
	return
}

func (c *ClientConsumers) StartConsumers() error {
	logger := logs.GetLogger()
	initializeGlobalVariables()
	var err error
	c.mu.Lock()
	c.total_consumers, err = c.getConsumersCount()
	c.mu.Unlock()
	c.initialize()
	if err != nil {
		return err
	}
	c.SPIN_UP_CHAN = make(chan bool, 10)
	go func() {
		for e := range c.SPIN_UP_CHAN {
			switch e {
			case true:
				logger.Info("MAX limit reached for consumer creating a new queue/consumer...")
				go c.AddClientQueue(c.total_consumers + 1)
			}
		}
	}()

	if c.total_consumers == 0 {
		err = c.AddClientQueue(1)
		if err != nil {
			logger.Error("Failed to create/add client Queue", err)
			return err
		}
	} else {
		for i := 1; i <= c.total_consumers; i++ {
			userTopic := "client-group-" + strconv.Itoa(i)
			go c.createConsumer(userTopic, i)
		}
	}
	logger.Info("Successfully started client consumers...")
	return nil
}

func (c *ClientConsumers) AddClientQueue(id int) error {
	logger := logs.GetLogger()
	config := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		// "security.protocol": "SSL",
	}
	admin, err := kafka.NewAdminClient(config)
	if err != nil {
		logger.Info("unable to create admin kafka client", err)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	userTopic := "client-group-" + strconv.Itoa(id)
	res, err := admin.CreateTopics(ctx, []kafka.TopicSpecification{{Topic: userTopic, NumPartitions: 1, ReplicationFactor: 1}})
	if err != nil {
		logger.Error("Failed to create queue", err)
		return err
	}
	logger.Println(res)
	go c.createConsumer(userTopic, id)
	c.setConsumersCount(c.total_consumers + 1)
	logger.Info("Succesfully added client queue", c.total_consumers)
	return nil
}

func (c *ClientConsumers) isLastConsumer(id int) bool {
	return id == c.total_consumers
}

func (c *ClientConsumers) createConsumer(topic string, id int) {
	logger := logs.GetLogger()
	config := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          topic,
		// "security.protocol": "SSL",
	}
	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		logger.Error("Unable to create consumer for user-queue", err)
	}
	kafkaAdminClient, err := kafka.NewAdminClientFromConsumer(consumer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = kafkaAdminClient.CreateTopics(ctx, []kafka.TopicSpecification{
		{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	},
		kafka.SetAdminOperationTimeout(10*time.Second),
	)
	if err != nil {
		logger.Error("Failed to create a new topic: ", err)
	}
	logger.Info("Successfully created a new topic: ", topic)
	err = consumer.Subscribe(topic, nil)
	if err != nil {
		logger.Error("Failed to subscribe to topic", err)
	}
	run := true

	for run {
		ev := consumer.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			{
				var notificationData DB.Notification
				err = json.Unmarshal(e.Value, &notificationData)
				if err != nil {
					logger.Error("failed to deserialize value from 'user_queue'. The producer might have sent data in an incompatible format.")

				}
				start := (id-1)*CLIENTS_LIMIT + 1
				var end int
				if c.isLastConsumer(id) {
					end = math.MaxInt32
				} else {
					end = id * CLIENTS_LIMIT
				}
				var rows []sqlc.Client
				rows, err := c.queries.GetClientsInRange(context.Background(), sqlc.GetClientsInRangeParams{
					TopicName: topic,
					StartID:   int32(start),
					EndID:     int32(end),
				})
				if err != nil {
					logger.Error("unable to query the clients from the database:", err)
					return
				}
				notificationService(topic, rows, notificationData)
				if c.isLastConsumer(id) && len(rows) >= MAX_CLIENTS_LIMIT {
					select {
					case c.SPIN_UP_CHAN <- true:
						logger.Println("Sent value")
					default:
						logger.Println("Channel might be closed or no receiver is ready")
					}
				}
			}
		case kafka.Error:
			logger.Error("Error while receiving message", e)
			run = false
		}
	}
	defer consumer.Close()
}

func (c *ClientConsumers) getConsumersCount() (int, error) {
	val, err := c.queries.GetConsumerQueuesCount(context.TODO())
	return int(val), err
}

func (c *ClientConsumers) setConsumersCount(count int) {
	logger := logs.GetLogger()
	err := c.queries.SetConsumerQueuesCount(context.TODO(), int32(count))
	if err != nil {
		logger.Error("Failed to update consumers_count", err)
	}
	c.mu.Lock()
	c.total_consumers = count
	c.mu.Unlock()
}

func notificationService(topicName string, subscribedClients []sqlc.Client, notification DB.Notification) {
	logger := logs.GetLogger()
	webhookUrl := os.Getenv("WEBHOOK_URL")
	if webhookUrl == "" {
		logger.Error("WEBHOOK_URL is not set")
		return
	}
	// Send notification for subscribed clients
	postBody, err := json.Marshal(DB.NotificationPostBody{
		Clients:      subscribedClients,
		Notification: notification,
	})
	if err != nil {
		logger.Error("Failed to marshal notification data:", err)
		return
	}

	_, err = http.Post(webhookUrl, "application/json", bytes.NewBuffer(postBody))

	if err != nil {
		logger.Error("Failed to send notification to webhook URL:", err)
		return
	}
	logger.Info("Notification sent successfully to webhook URL")
	return
}
