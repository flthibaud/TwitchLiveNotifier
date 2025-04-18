package utils

import (
	"os"
	"time"

	"github.com/flthibaud/TwitchLiveNotifier/internal/config"
	"github.com/sirupsen/logrus"
)

// NewLogger creates and configures a logrus.Logger based on environment/config values
func NewLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// Output to stdout for container-friendly logging
	logger.Out = os.Stdout

	// Set log level from environment or default to Info
	if lvl, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
		logger.SetLevel(lvl)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Use a text formatter with timestamps
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
	})

	// Example of using config values if needed (e.g., adding fields)
	// logger = logger.WithField("app", cfg.AppName).Logger

	return logger
}
