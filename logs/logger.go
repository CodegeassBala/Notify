package logs

import (
	"io"
	"os"
	"sync"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger
	once   sync.Once
)

func GetLogger() *logrus.Logger {
	once.Do(
		func() {
			logger = logrus.New()
			logfile := &lumberjack.Logger{
				Filename:   "sever.logs",
				MaxSize:    10,
				MaxAge:     1,
				MaxBackups: 1,
				Compress:   true,
			}
			multiWriter := io.MultiWriter(os.Stdout, logfile)
			logger.SetOutput(multiWriter)
			logger.SetFormatter(&logrus.JSONFormatter{})
		},
	)
	return logger
}
