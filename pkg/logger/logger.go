package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Level represents the severity of a log entry.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// Fields is a typed alias so callers never need to write map[string]interface{} inline.
type Fields map[string]interface{}

// Logger is a minimal structured logger with level filtering and key=value fields.
type Logger struct {
	level Level
	inner *log.Logger
}

// New creates a Logger with the given minimum level string ("debug","info","warn","error").
func New(levelStr string) *Logger {
	lvl := INFO
	switch levelStr {
	case "debug":
		lvl = DEBUG
	case "warn":
		lvl = WARN
	case "error":
		lvl = ERROR
	}
	return &Logger{level: lvl, inner: log.New(os.Stdout, "", 0)}
}

func (l *Logger) emit(lvl Level, msg string, fields Fields) {
	if lvl < l.level {
		return
	}
	label := [...]string{"DEBUG", "INFO", "WARN", "ERROR"}[lvl]
	line := fmt.Sprintf("%s [%s] %s", time.Now().Format("15:04:05"), label, msg)
	for k, v := range fields {
		line += fmt.Sprintf(" %s=%v", k, v)
	}
	l.inner.Println(line)
}

func (l *Logger) Debug(msg string, fields ...Fields) {
	l.emit(DEBUG, msg, mergeFields(fields))
}

func (l *Logger) Info(msg string, fields ...Fields) {
	l.emit(INFO, msg, mergeFields(fields))
}

func (l *Logger) Warn(msg string, fields ...Fields) {
	l.emit(WARN, msg, mergeFields(fields))
}

func (l *Logger) Error(msg string, fields ...Fields) {
	l.emit(ERROR, msg, mergeFields(fields))
}

func mergeFields(all []Fields) Fields {
	if len(all) == 0 {
		return Fields{}
	}
	return all[0]
}
