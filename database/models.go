package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"notify/database/sqlc"
	"notify/logs"
	"strings"
)

type ConnectionParams struct {
	Dbname          string `json:"dbname"`
	User            string `json:"user"`
	Password        string `json:"password"`
	Port            string `json:"port"`
	Sslmode         string `json:"sslmode"`
	Connect_timeout string `json:"connect_timeout"`
	Host            string `json:"host"`
}

func (c ConnectionParams) getConnectionString() string {
	logger := logs.GetLogger()
	jsonData, err := json.Marshal(c)
	if err != nil {
		logger.Error("Error marshaling to JSON:", err)
		return ""
	}
	var resultMap map[string]string
	if err := json.Unmarshal(jsonData, &resultMap); err != nil {
		logger.Error("Error unmarshaling JSON to map:", err)
		return ""
	}
	var builder strings.Builder
	for key, value := range resultMap {
		builder.WriteString(fmt.Sprintf("%s=%s ", key, value))
	}
	connectionString := builder.String()
	logger.Println(connectionString)
	return connectionString
}

type ClientConfig struct {
	ClientID   string         `json:"clientID"`
	Email      string         `json:"email"`
	Phone      string         `json:"phone"`
	Connection sql.NullString `json:"connection,omitempty"`
}
type TopicConfig struct {
	TopicName string `form:"topicName" json:"topicName"`
}

type Notification struct {
	TopicName string `json:"topic"`
	Value     string `json:"value"`
	Key       string `json:"key,omitempty"`
}

type Topic_Client struct {
	TopicName string `db:"topicname"`
	ClientId  string `db:"clientid"`
	SerialId  string `db:"serialId"`
}

type NotificationPostBody struct {
	Clients      []sqlc.Client `json:"clients"`
	Notification Notification  `json:"notification"`
}
