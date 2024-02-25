package sling

import (
	"context"
)

type Logger interface {
	WithContext(ctx context.Context) Logger
	WithFields(keyValues Fields) Logger

	Info(msg string)
	Infof(format string, args ...interface{})

	Error(msg string)
	Errorf(format string, args ...interface{})
}

type Fields map[string]interface{}

type emptyLogger struct {
}

func NewEmptyLogger() Logger {
	return emptyLogger{}
}

func (l emptyLogger) WithContext(ctx context.Context) Logger {
	return l
}

func (l emptyLogger) WithFields(keyValues Fields) Logger {
	return l
}

func (l emptyLogger) Info(msg string) {
}

func (l emptyLogger) Infof(format string, args ...interface{}) {
}

func (l emptyLogger) Error(msg string) {
}

func (l emptyLogger) Errorf(format string, args ...interface{}) {
}
