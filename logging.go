package main

import (
	"log"
	"strings"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelDebug
)

// Logger provides structured logging with configurable levels
type Logger struct {
	level LogLevel
}

// NewLogger creates a new logger with the specified level
func NewLogger(levelStr string) *Logger {
	var level LogLevel
	switch strings.ToLower(levelStr) {
	case "debug":
		level = LogLevelDebug
	case "info":
		level = LogLevelInfo
	default:
		level = LogLevelInfo // Default to info
	}

	return &Logger{level: level}
}

// Info logs messages at info level (always shown)
func (l *Logger) Info(v ...interface{}) {
	log.Print(v...)
}

// Infof logs formatted messages at info level (always shown)
func (l *Logger) Infof(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Debug logs messages at debug level (only shown when debug is enabled)
func (l *Logger) Debug(v ...interface{}) {
	if l.level >= LogLevelDebug {
		log.Print(v...)
	}
}

// Debugf logs formatted messages at debug level (only shown when debug is enabled)
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.level >= LogLevelDebug {
		log.Printf(format, v...)
	}
}

// Fatal logs fatal messages and exits (always shown)
func (l *Logger) Fatal(v ...interface{}) {
	log.Fatal(v...)
}

// Fatalf logs formatted fatal messages and exits (always shown)
func (l *Logger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// Global logger instance
var logger *Logger

// InitializeLogger initializes the global logger with config
func InitializeLogger(config *Config) {
	logger = NewLogger(config.Logging.Level)
}

// Convenience functions for global logger
func LogInfo(v ...interface{}) {
	if logger != nil {
		logger.Info(v...)
	} else {
		log.Print(v...)
	}
}

func LogInfof(format string, v ...interface{}) {
	if logger != nil {
		logger.Infof(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func LogDebug(v ...interface{}) {
	if logger != nil {
		logger.Debug(v...)
	}
}

func LogDebugf(format string, v ...interface{}) {
	if logger != nil {
		logger.Debugf(format, v...)
	}
}

func LogFatal(v ...interface{}) {
	if logger != nil {
		logger.Fatal(v...)
	} else {
		log.Fatal(v...)
	}
}

func LogFatalf(format string, v ...interface{}) {
	if logger != nil {
		logger.Fatalf(format, v...)
	} else {
		log.Fatalf(format, v...)
	}
}
