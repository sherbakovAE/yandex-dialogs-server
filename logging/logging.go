package logging

import (
	"github.com/evalphobia/go-log-wrapper/log"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"sync"
)

var once sync.Once
var logger = new(log.Logger)

func GetInstance() *log.Logger {
	//
	once.Do(func() {
		logger = log.NewLogger()
		logger.Level = logrus.WarnLevel
		logger.SetFormatter(&logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})

		logger.SetOutput(&lumberjack.Logger{
			Filename:   "skills_server.log",
			MaxSize:    50, // megabytes
			MaxBackups: 30,
			MaxAge:     60, // days
			// Compress:   true, // disabled by default
		})
	})
	return logger
}
