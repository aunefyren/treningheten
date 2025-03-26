package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func InitLogger() {
	Log = logrus.New()

	// Define log file
	logFile, err := os.OpenFile("files/treningheten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logrus.Fatalf("Failed to load log file: %v", err)
	}

	// Set log format
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Output to both stdout and log file
	mw := io.MultiWriter(os.Stdout, logFile)
	Log.SetOutput(mw)

	// Set log level (adjust as needed)
	Log.SetLevel(logrus.InfoLevel)
}
