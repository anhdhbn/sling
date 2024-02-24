package sling

import (
	"context"
	"io"
	stdlog "log"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"go.opentelemetry.io/otel/trace"
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

type defaultLogger struct {
	logger    logr.Logger
	extractor []ContextExtractor
}

type ContextExtractor func(context.Context) map[string]interface{}

func NewDefaultLogger() Logger {
	logr := stdr.New(
		stdlog.New(io.Discard, "", stdlog.LstdFlags),
	)
	return &defaultLogger{logger: logr}
}

func (l *defaultLogger) WithContext(ctx context.Context) Logger {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	ctxValues := make([]interface{}, 0)

	if sc.HasTraceID() && sc.TraceID().IsValid() {
		ctxValues = append(ctxValues, "TraceId", sc.TraceID().String())
	}

	if sc.HasSpanID() && sc.SpanID().IsValid() {
		ctxValues = append(ctxValues, "SpanId", sc.SpanID().String())
	}

	for _, extractor := range l.extractor {
		m := extractor(ctx)
		for k, v := range m {
			ctxValues = append(ctxValues, k, v)
		}
	}

	return &defaultLogger{
		logger:    l.logger.WithValues(ctxValues...),
		extractor: l.extractor,
	}
}

func (log *defaultLogger) Info(msg string) {
	log.logger.Info(msg)
}

func (log *defaultLogger) Infof(format string, args ...interface{}) {
	log.logger.Info(format)
}

func (l *defaultLogger) WithFields(keyValues Fields) Logger {
	kvs := make([]interface{}, 0)
	for k, v := range keyValues {
		kvs = append(kvs, k, v)
	}
	return &defaultLogger{
		logger:    l.logger.WithValues(kvs...),
		extractor: l.extractor,
	}
}

func (log *defaultLogger) Error(msg string) {
	log.logger.Error(nil, msg)
}

func (log *defaultLogger) Errorf(format string, args ...interface{}) {
	log.logger.Error(nil, format, args...)
}
