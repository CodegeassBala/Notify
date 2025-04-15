package main

import (
	"encoding/json"
	"net/http"
	DB "notify/database"
	kafkaModels "notify/kafka"
	"notify/logs"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

var logger *logrus.Logger

func main() {
	// setup logger
	logger = logs.GetLogger()
	// Connect to DB..
	notifyDB, err := DB.GetDB()
	if err != nil {
		return
	}
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
	logger.Println("Started all consumers from main.go")
	router := gin.Default()
	router.GET("/healthCheck", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Health Check successful"})
	})
	router.POST("/topic/add", addTopic(notifyDB))
	router.POST("/client", addUser(notifyDB))
	router.POST("/topic/subscribe", subscribeUser(notifyDB))
	router.POST("/topic/unsubscribe", unSubscribeUser(notifyDB))
	router.POST("/notify", pushMessage(notifyDB, p.Producer))
	router.Run("localhost:8085")
}

func addTopic(notifyDB *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var topicConfig DB.TopicConfig
		if err := c.BindJSON(&topicConfig); err != nil {

			// If there's an error during binding, send a 400 Bad Request response.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// add topic to the table DB

		_, err := notifyDB.Query(`
		INSERT INTO topics (topicName)
		VALUES ($1)`,
			topicConfig.TopicName)

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

func addUser(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var clientConfig DB.ClientConfig
		if err := c.BindJSON(&clientConfig); err != nil {

			// If there's an error during binding, send a 400 Bad Request response.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// add topic to the table DB

		_, err := db.Query(`
		INSERT INTO "clients" (clientID, email, phone)
		VALUES ($1, $2, $3)`,
			clientConfig.ClientID, clientConfig.Email, clientConfig.Phone)

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

func subscribeUser(db *sqlx.DB) gin.HandlerFunc {
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
		_, err = db.Query(
			`INSERT INTO topics_clients (topicName,clientID) VALUES ($2,$1)`,
			subscribeConfig.ClientID,
			subscribeConfig.TopicName,
		)
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

func unSubscribeUser(db *sqlx.DB) gin.HandlerFunc {
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
		_, err = db.Query(
			`DELETE FROM topics_clients WHERE topicName = $2 AND clientID = $1`,
			subscribeConfig.ClientID,
			subscribeConfig.TopicName,
		)
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

func pushMessage(db *sqlx.DB, producer *kafka.Producer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data DB.Notification

		// Parse request body
		if err := c.BindJSON(&data); handleError(c, http.StatusBadRequest, err, "Invalid JSON payload") {
			return
		}

		// Check if topic exists in the database
		rows, err := db.Query(`SELECT * FROM topics WHERE topicName = $1`, data.TopicName)
		if handleError(c, http.StatusInternalServerError, err, "Database query failed") {
			return
		}
		defer rows.Close()
		if !rows.Next() {
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
