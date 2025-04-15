package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

type ctxKey struct{}

var (
	defaultOnce   sync.Once
	defaultLogger *Logger
)

type Logger struct {
	mu    sync.Mutex
	level Level
	out   io.Writer
}

type Options struct {
	Level Level
}

func New(opts Options) *Logger {
	return &Logger{
		level: opts.Level,
	}
}

func Default() *Logger {
	defaultOnce.Do(func() {
		defaultLogger = New(Options{
			Level: LevelInfo,
		})
	})
	return defaultLogger
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) log(level Level, msg string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	var sb strings.Builder

	sb.WriteString(time.Now().Format("2006-01-02 15:04:05.000"))
	sb.WriteString(" ")

	sb.WriteString(levelNames[level])
	sb.WriteString(" ")

	sb.WriteString("[traefik-ratelimit]")
	sb.WriteString(" ")

	sb.WriteString(msg)

	if len(fields) > 0 {
		sb.WriteString(" ")
		for i := 0; i < len(fields); i += 2 {
			if i > 0 {
				sb.WriteString(" ")
			}
			key := fmt.Sprint(fields[i])
			value := ""
			if i+1 < len(fields) {
				value = fmt.Sprint(fields[i+1])
			}
			sb.WriteString(fmt.Sprintf("%s=%s", key, value))
		}
	}

	if level <= LevelDebug {
		_, file, line, ok := runtime.Caller(3) // skip 3 frames
		if ok {
			sb.WriteString(fmt.Sprintf(" [%s:%d]", filepath.Base(file), line))
		}
	}

	sb.WriteString("\n")

	_, _ = os.Stderr.WriteString(sb.String())
}

func (l *Logger) Debug(msg string, fields ...any) {
	l.log(LevelDebug, msg, fields...)
}

func (l *Logger) Info(msg string, fields ...any) {
	l.log(LevelInfo, msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...any) {
	l.log(LevelWarn, msg, fields...)
}

func (l *Logger) Error(msg string, fields ...any) {
	l.log(LevelError, msg, fields...)
}

func FromCtx(ctx context.Context) *Logger {
	if ctx != nil {
		if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
			return l
		}
	}
	return Default()
}

func WithCtx(ctx context.Context, logger *Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKey{}, logger)
}

func SetLevel(ctx context.Context, level Level) {
	FromCtx(ctx).SetLevel(level)
}

func SetDebug(ctx context.Context, debug bool) {
	if debug {
		FromCtx(ctx).SetLevel(LevelDebug)
	}
}

func Debug(ctx context.Context, msg string, fields ...any) {
	FromCtx(ctx).Debug(msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...any) {
	FromCtx(ctx).Info(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...any) {
	FromCtx(ctx).Warn(msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...any) {
	FromCtx(ctx).Error(msg, fields...)
}
