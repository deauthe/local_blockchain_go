package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	gokitlog "github.com/go-kit/log"
)

// NewLogger creates a new logger that writes to both stdout and a file
func NewLogger(prefix string) (gokitlog.Logger, error) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join("logs", fmt.Sprintf("%s_%s.log", prefix, timestamp))

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer to write to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Create logger with timestamp and file/line information
	logger := gokitlog.NewLogfmtLogger(multiWriter)
	logger = gokitlog.With(logger, "ts", gokitlog.DefaultTimestampUTC, "caller", gokitlog.DefaultCaller)

	return logger, nil
}

// LogWithTiming logs a message with timing information
func LogWithTiming(logger gokitlog.Logger, startTime time.Time, format string, v ...interface{}) {
	elapsed := time.Since(startTime)
	message := fmt.Sprintf(format, v...)
	logger.Log("msg", fmt.Sprintf("%s (took %v)", message, elapsed))
}

// TimeFunction wraps a function with timing information
func TimeFunction(logger gokitlog.Logger, name string, fn func() error) error {
	startTime := time.Now()
	logger.Log("msg", fmt.Sprintf("Starting %s", name))

	err := fn()

	elapsed := time.Since(startTime)
	if err != nil {
		logger.Log("msg", fmt.Sprintf("Completed %s with error: %v (took %v)", name, err, elapsed))
	} else {
		logger.Log("msg", fmt.Sprintf("Completed %s successfully (took %v)", name, elapsed))
	}

	return err
}
