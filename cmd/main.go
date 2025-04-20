package main

import (
	"context"
	"encoding/json"
	"net/http"
	DB "notify/database"
	"notify/database/sqlc"
	kafkaModels "notify/kafka"
	"notify/logs"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

var logger *logrus.Logger

func main() {
	// setup logger
	logger = logs.GetLogger()
	// Connect to DB..
	notifyDBPool, err := DB.GetDBPool()
	defer notifyDBPool.Close()
	if err != nil {
		return
	}
	logger.Info("Connected to DB..")
	queries := sqlc.New(notifyDBPool)
	//Connect to KAFKA MQ..
	p := kafkaModels.KafkaProducer{}
	err = p.Initialize()
	if err != nil {
		logger.Error("Error connecting to kafka", err.Error())
		return
	} else {
		logger.Info("Connected to KAFKA..")
	}

	kafkaModels.StartConsumers()
	cc := kafkaModels.ClientConsumers{}
	err = cc.StartConsumers()
	if err != nil {
		logger.Error("Failed to start client consumers", err)
		// panic(err)
	}
	router := gin.Default()
	router.GET("/healthCheck", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Health Check successful"})
	})
	router.POST("/topic/add", addTopic(queries))
	router.POST("/client", addUser(queries))
	router.POST("/topic/subscribe", subscribeUser(queries))
	router.POST("/topic/unsubscribe", unSubscribeUser(queries))
	router.POST("/notify", pushMessage(queries, p.Producer))
	router.Run("localhost:8085")
}

func addTopic(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var topicConfig DB.TopicConfig
		if err := c.BindJSON(&topicConfig); err != nil {

			// If there's an error during binding, send a 400 Bad Request response.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		err := queries.InsertTopic(context.TODO(), topicConfig.TopicName)

		if err != nil {
			logger.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Addition of topic completed",
			"data":    topicConfig,
		})
	}
}

func addUser(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var clientConfig sqlc.InsertClientParams
		if err := c.BindJSON(&clientConfig); err != nil {

			// If there's an error during binding, send a 400 Bad Request response.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// add client to the table
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := queries.InsertClient(ctx, clientConfig)

		if err != nil {
			logger.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Addition of client was completed",
			"data":    clientConfig,
		})
	}
}

func subscribeUser(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var subscribeConfig struct {
			TopicName string `json:"topicName"`
			ClientID  string `json:"clientID"`
		}
		err := c.BindJSON(&subscribeConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		logger.Info(subscribeConfig)

		err = queries.InsertTopicClient(context.TODO(), sqlc.InsertTopicClientParams{
			Clientid:  subscribeConfig.ClientID,
			Topicname: subscribeConfig.TopicName})

		if err != nil {
			logger.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Subscription of client was completed",
			"data":    subscribeConfig,
		})
	}
}

func unSubscribeUser(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var subscribeConfig struct {
			TopicName string `json:"topicName"`
			ClientID  string `json:"clientID"`
		}
		err := c.BindJSON(&subscribeConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		err = queries.DeleteTopicClient(context.TODO(), sqlc.DeleteTopicClientParams{
			Clientid:  subscribeConfig.ClientID,
			Topicname: subscribeConfig.TopicName})
		if err != nil {
			logger.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Unsubscribed client from topic",
			"data":    subscribeConfig,
		})
	}
}

func handleError(c *gin.Context, statusCode int, err error, msg string) bool {
	if err != nil {
		c.JSON(statusCode, gin.H{
			"error":   msg,
			"details": err.Error(),
		})
		return true
	}
	return false
}

func pushMessage(queries *sqlc.Queries, producer *kafka.Producer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data DB.Notification

		// Parse request body
		if err := c.BindJSON(&data); handleError(c, http.StatusBadRequest, err, "Invalid JSON payload") {
			return
		}
		// Check if topic exists in the database
		topic, err := queries.GetTopic(context.TODO(), data.TopicName)
		if handleError(c, http.StatusInternalServerError, err, "Database query failed") {
			return
		}
		if topic == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Given topic does not exist in the topic database",
			})
			return
		}
		// Serialize key and value
		key, err := json.Marshal(data.Key)
		if handleError(c, http.StatusInternalServerError, err, "Failed to serialize key") {
			return
		}

		value, err := json.Marshal(data)
		if handleError(c, http.StatusInternalServerError, err, "Failed to serialize value") {
			return
		}

		// Push notification into the queue
		deliveryChan := make(chan kafka.Event)
		go func() {
			defer close(deliveryChan)
			for e := range deliveryChan {
				switch ev := e.(type) {
				case *kafka.Message:
					if ev.TopicPartition.Error != nil {
						logger.Info("Failed to deliver message: %v\n", ev.TopicPartition)
					} else {
						logger.Info("Successfully produced record to topic %s partition [%d] @ offset %v\n",
							*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
					}
				}
			}
		}()

		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &data.TopicName,
				Partition: kafka.PartitionAny,
			},
			Key:   key,
			Value: value,
		}, deliveryChan)
		if handleError(c, http.StatusInternalServerError, err, "Failed to push message to Kafka") {
			return
		}

		// Respond to the client
		c.JSON(http.StatusAccepted, gin.H{
			"message": "Your message is getting processed",
		})
	}
}
