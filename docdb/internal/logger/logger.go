package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	mu     sync.Mutex
	level  Level
	out    io.Writer
	prefix string
}

func New(out io.Writer, level Level, prefix string) *Logger {
	return &Logger{
		level:  level,
		out:    out,
		prefix: prefix,
	}
}

func Default() *Logger {
	return New(os.Stderr, LevelInfo, "[docdb]")
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) SetOutput(out io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = out
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := levelToString(level)
	message := fmt.Sprintf(format, args...)

	fmt.Fprintf(l.out, "%s %s%s %s\n", timestamp, l.prefix, levelStr, message)
}

func levelToString(level Level) string {
	switch level {
	case LevelDebug:
		return " [DEBUG]"
	case LevelInfo:
		return " [INFO]"
	case LevelWarn:
		return " [WARN]"
	case LevelError:
		return " [ERROR]"
	default:
		return " [?]"
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}
