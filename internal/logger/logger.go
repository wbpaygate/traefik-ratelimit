package logger

import (
	"context"
	"fmt"
	"os"
	"time"
)

func NewLogger() *Logger {
	return &Logger{}
}

type Logger struct {
}

func (l Logger) Info(_ context.Context, message string) {
	_, _ = os.Stderr.WriteString(l.formatLogMsg("info", message))
}

func (l Logger) Error(_ context.Context, err error) {
	_, _ = os.Stderr.WriteString(l.formatLogMsg("error", err.Error()))
}

func (l Logger) formatLogMsg(level, msg string) string {
	return fmt.Sprintf("level=%s time=%q traefikPlugin=\"%s\" msg=%q\n",
		level,
		time.Now().UTC().Format("2006-01-02 15:04:05Z"),
		"ratelimit",
		msg,
	)
}
