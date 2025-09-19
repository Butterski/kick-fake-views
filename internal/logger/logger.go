package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// NewLogger creates a new configured logger instance
func NewLogger() *logrus.Logger {
	logger := logrus.New()

	// Set output to stdout
	logger.SetOutput(os.Stdout)

	// Set log level to Info by default
	logger.SetLevel(logrus.InfoLevel)

	// Use JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	return logger
}

// NewTextLogger creates a logger with text formatter for better readability during development
func NewTextLogger() *logrus.Logger {
	logger := logrus.New()

	// Set output to stdout
	logger.SetOutput(os.Stdout)

	// Set log level to Info by default
	logger.SetLevel(logrus.InfoLevel)

	// Use text formatter with colors
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})

	return logger
}

// SetLogLevel sets the log level from string
func SetLogLevel(logger *logrus.Logger, level string) {
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
}
