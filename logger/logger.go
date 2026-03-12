package logger

import (
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log   *zap.Logger
	Sugar *zap.SugaredLogger
)

func init() {
	// Default to "info" level, but allow overriding with an environment variable.
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	var level zapcore.Level
	if err := level.Set(logLevel); err != nil {
		log.Fatalf("Invalid log level '%s': %v", logLevel, err)
	}

	// Choose config based on environment (e.g., "production" or "development")
	var config zap.Config
	if os.Getenv("ENV") == "production" {
		config = zap.NewProductionConfig() // JSON output, info level by default
	} else {
		config = zap.NewDevelopmentConfig() // Human-readable, debug level by default
	}
	config.Level = zap.NewAtomicLevelAt(level)

	// Build the logger with a buffered writer for asynchronous logging.
	// This improves performance by not blocking application goroutines on I/O.
	// The `defer logger.Sync()` in main.go becomes critical to ensure logs are flushed on exit.
	writer := zapcore.AddSync(os.Stdout)
	bufferedWriter := &zapcore.BufferedWriteSyncer{WS: writer, Size: 256 * 1024, FlushInterval: 30 * time.Second}

	// Use a different encoder based on the environment
	var encoder zapcore.Encoder
	if os.Getenv("ENV") == "production" {
		encoder = zapcore.NewJSONEncoder(config.EncoderConfig)
	} else {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Add color for development
		encoder = zapcore.NewConsoleEncoder(config.EncoderConfig)
	}

	core := zapcore.NewCore(encoder, bufferedWriter, config.Level)
	// Rebuild the logger with the new core.
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel), zap.ErrorOutput(writer))

	Log = logger
	Sugar = logger.Sugar()
}
