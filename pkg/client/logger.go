package client

import (
	"context"

	"github.com/rs/zerolog"
)

type GetLogger func(ctx context.Context) Logger

func LogField(key string, value any) KeyValue {
	return KeyValue{
		Key:   key,
		Value: value,
	}
}

type KeyValue struct {
	Key   string
	Value any
}

type Logger interface {
	Debug(msg string, args ...KeyValue)
	Info(msg string, args ...KeyValue)
	Error(msg string, args ...KeyValue)
}

func GetLoggerZerolog(ctx context.Context) Logger {
	return WrapZerolog(zerolog.Ctx(ctx))
}

func WrapZerolog(log *zerolog.Logger) Logger {
	return zerologWrapper{
		log: log.With().Logger(),
	}
}

type zerologWrapper struct {
	log zerolog.Logger
}

func (l zerologWrapper) Debug(msg string, args ...KeyValue) {
	event := l.log.Debug()
	for _, arg := range args {
		event.Interface(arg.Key, arg.Value)
	}

	event.Msg(msg)
}

func (l zerologWrapper) Info(msg string, args ...KeyValue) {
	event := l.log.Info()
	for _, arg := range args {
		event.Interface(arg.Key, arg.Value)
	}

	event.Msg(msg)
}

func (l zerologWrapper) Error(msg string, args ...KeyValue) {
	event := l.log.Error()
	for _, arg := range args {
		event.Interface(arg.Key, arg.Value)
	}

	event.Msg(msg)
}

func GetLoggerNoop(ctx context.Context) Logger { return noopLogger{} }

type noopLogger struct{}

func (noopLogger) Debug(format string, args ...KeyValue) {}
func (noopLogger) Info(format string, args ...KeyValue)  {}
func (noopLogger) Error(format string, args ...KeyValue) {}
