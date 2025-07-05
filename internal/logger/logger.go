package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/farhapartex/bolt-load-balancer/internal/config"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) DataReprensation() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	level     LogLevel
	format    string
	accessLog bool
	stdLogger *log.Logger
}

func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)

	if l.format == "json" {
		entry := LogEntry{
			Timestamp: timestamp,
			Level:     level.DataReprensation(),
			Message:   message,
			Fields:    fields,
		}

		jsonData, err := json.Marshal(entry)
		if err != nil {
			l.stdLogger.Printf("[ERROR] Failed to marshal log entry: %v", err)
			return
		}
		l.stdLogger.Println(string(jsonData))
	} else {
		logLine := fmt.Sprintf("[%s] %s %s", level.DataReprensation(), timestamp, message)

		if len(fields) > 0 {
			for key, value := range fields {
				logLine += fmt.Sprintf(" %s=%v", key, value)
			}
		}
		l.stdLogger.Println(logLine)
	}
}

func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(DEBUG, message, f)
}

func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(INFO, message, f)
}

func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(WARN, message, f)
}

func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ERROR, message, f)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

func (l *Logger) LogRequest(method, path, remoteAddr, userAgent string, statusCode int, duration time.Duration) {
	if !l.accessLog {
		return
	}

	fields := map[string]interface{}{
		"method":      method,
		"path":        path,
		"remote_addr": remoteAddr,
		"user_agent":  userAgent,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	}

	message := fmt.Sprintf("%s %s - %d", method, path, statusCode)
	l.log(INFO, message, fields)
}

func (l *Logger) LogBackendRequest(backendURL, method, path string, statusCode int, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"backend_url": backendURL,
		"method":      method,
		"path":        path,
		"duration_ms": duration.Milliseconds(),
	}

	if err != nil {
		fields["error"] = err.Error()
		l.log(ERROR, fmt.Sprintf("Backend request failed: %s %s", method, backendURL), fields)
	} else {
		fields["status_code"] = statusCode
		l.log(DEBUG, fmt.Sprintf("Backend request: %s %s - %d", method, backendURL, statusCode), fields)
	}
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func parseLogLevel(levelStr string) LogLevel {
	switch levelStr {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

func NewLogger(config config.LoggingConfig) *Logger {
	level := parseLogLevel(config.Level)

	logger := &Logger{
		level:     level,
		format:    config.Format,
		accessLog: config.AccessLog,
		stdLogger: log.New(os.Stdout, "", 0),
	}

	return logger
}
