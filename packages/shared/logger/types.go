package logger

import "log"

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
