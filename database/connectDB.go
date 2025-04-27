package database

import (
	"context"
	"notify/database/sqlc"
	"notify/logs"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool    *pgxpool.Pool
	once      sync.Once
	initError error
)

func GetDBPool() (*pgxpool.Pool, error) {
	once.Do(func() {
		logger := logs.GetLogger()
		dbPool, initError = connectToDB()
		// db.SetMaxOpenConns(25) // Maximum number of open connections
		// db.SetMaxIdleConns(5)  // Maximum number of idle connections

		logger.Info("Database connection successfully established")
	})
	return dbPool, initError
}

func connectToDB() (*pgxpool.Pool, error) {
	logger := logs.GetLogger()
	connectionConfig := ConnectionParams{
		"notify",
		"postgres",
		"cr07fr01@EN",
		"5432",
		"disable",
		"30000",
		"localhost"}
	connStr := connectionConfig.getConnectionString()
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		logger.Error("Failed to parse connection string", err)
		return nil, err
	}
	poolConfig.MaxConns = 10
	pgxPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Error("Failed to create connection pool", err)
		return nil, err
	}
	return pgxPool, nil
}
func GetConsumersCount() (int, error) {
	logger := logs.GetLogger()
	db, err := GetDBPool()
	if err != nil {
		logger.Error("failed to connect to db while querying to get the consumer's count", err)
		return -1, err
	}
	queries := sqlc.New(db)
	val, err := queries.GetConsumerQueuesCount(context.TODO())
	return int(val), err
}
