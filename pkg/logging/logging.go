package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type LogLevel string

const (
	DEBUG LogLevel = "debug"
	INFO  LogLevel = "info"
	WARN  LogLevel = "warn"
	ERROR LogLevel = "error"
)

type Logger struct {
	level  LogLevel
	format string
	writer io.Writer
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func NewLogger(level, format, output string) *Logger {
	var writer io.Writer = os.Stdout
	if output == "stderr" {
		writer = os.Stderr
	} else if output != "" && output != "stdout" {
		file, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			writer = file
		}
	}

	return &Logger{
		level:  LogLevel(level),
		format: format,
		writer: writer,
	}
}

func shouldLog(currentLevel, targetLevel LogLevel) bool {
	levels := map[LogLevel]int{
		DEBUG: 0,
		INFO:  1,
		WARN:  2,
		ERROR: 3,
	}
	return levels[currentLevel] <= levels[targetLevel]
}

func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if !shouldLog(l.level, level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     string(level),
		Message:   message,
		Fields:    fields,
	}

	if l.format == "json" {
		data, err := json.Marshal(entry)
		if err != nil {
			log.Printf("error marshaling log entry: %v", err)
			return
		}
		fmt.Fprintln(l.writer, string(data))
	} else {
		log.SetOutput(l.writer)
		if len(fields) > 0 {
			log.Printf("[%s] %s %v", level, message, fields)
		} else {
			log.Printf("[%s] %s", level, message)
		}
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
