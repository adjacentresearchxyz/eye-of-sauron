package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

var levelStrings = map[LogLevel]string{
	DEBUG:   "DEBUG",
	INFO:    "INFO",
	WARNING: "WARNING",
	ERROR:   "ERROR",
}

type Logger struct {
	loggers    map[LogLevel]*log.Logger
	level      LogLevel
	moduleName string
}

func NewLogger(moduleName, logPath string, maxSize, maxBackups, maxAge int, minLevel LogLevel) (*Logger, error) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Setup log rotation
	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    maxSize,    // megabytes
		MaxBackups: maxBackups, // number of backups
		MaxAge:     maxAge,     // days
		Compress:   true,       // compress backups
	}

	// MultiWriter to write to both file and stdout
	multiWriter := io.MultiWriter(rotator, os.Stdout)

	// Create logger for each level
	loggers := make(map[LogLevel]*log.Logger)
	for level, prefix := range levelStrings {
		loggers[level] = log.New(multiWriter, fmt.Sprintf("[%s] [%s] ", prefix, moduleName), log.LstdFlags)
	}

	return &Logger{
		loggers:    loggers,
		level:      minLevel,
		moduleName: moduleName,
	}, nil
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.loggers[DEBUG].Printf(format, v...)
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		l.loggers[INFO].Printf(format, v...)
	}
}

func (l *Logger) Warning(format string, v ...interface{}) {
	if l.level <= WARNING {
		l.loggers[WARNING].Printf(format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.loggers[ERROR].Printf(format, v...)
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func GetLogLevelFromString(level string) LogLevel {
	switch level {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARNING":
		return WARNING
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}