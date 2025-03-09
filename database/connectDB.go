package database

import (
	"database/sql"
	"errors"
	"notify/logs"
	"strconv"
	"sync"

	"github.com/jmoiron/sqlx"
)

var (
	db        *sqlx.DB
	once      sync.Once
	initError error
)

func GetDB() (*sqlx.DB, error) {
	once.Do(func() {
		logger := logs.GetLogger()
		db, initError = connectToDB()
		db.SetMaxOpenConns(25) // Maximum number of open connections
		db.SetMaxIdleConns(5)  // Maximum number of idle connections

		logger.Info("Database connection successfully established")
	})
	return db, initError
}

func connectToDB() (*sqlx.DB, error) {
	logger := logs.GetLogger()
	connectionConfig := ConnectionParams{
		"notify",
		"postgres",
		"cr07fr01@EN",
		"5432",
		"disable",
		"30000",
	}
	connStr := connectionConfig.getConnectionString()
	notifyDB, err := sqlx.Open("postgres", connStr)
	if err != nil {
		logger.Error("Could not connect to DB ", err)
		return nil, err
	}
	res := notifyDB.Ping()
	if res != nil {
		logger.Error("Connection to DB failed", res)
		return nil, errors.New("connection to DB failed")
	}
	logger.Info("Connection to DB Successfull!")
	return notifyDB, nil
}
func GetConsumersCount() (int, error) {
	logger := logs.GetLogger()
	db, err := GetDB()
	if err != nil {
		logger.Error("failed to connect to db while querying to get the consumer's count", err)
		return -1, err
	}
	queryString := `
		SELECT h['client_consumers_count'] from variables;
	`
	res := db.QueryRow(queryString)
	var r sql.NullString
	res.Scan(&r)
	val, err := strconv.Atoi(r.String)
	return val, err
}
