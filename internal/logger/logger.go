package logger

import (
	"context"
	"fmt"
	"os"
	"time"
)

func New() *Logger {
	return &Logger{}
}

type Logger struct {
}

func (l Logger) Info(_ context.Context, message string) {
	_, _ = os.Stderr.WriteString(l.formatLogMsg(message))
}

func (l Logger) Error(_ context.Context, err error) {
	_, _ = os.Stderr.WriteString(l.formatLogMsg(err.Error()))
}

func (l Logger) formatLogMsg(msg string) string {
	return fmt.Sprintf("time=%q traefikPlugin=\"%s\" msg=%q\n",
		time.Now().UTC().Format("2006-01-02 15:04:05Z"),
		"ratelimit",
		msg,
	)
}
