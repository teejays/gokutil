// Package log provides a simple logging interface that can be used to log messages to the console.
// It implements a singleton pattern.
package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/teejays/gokutil/env"
	"github.com/teejays/gokutil/panics"
	"github.com/teejays/gokutil/sclog"
)

var defaultLogger LoggerI = nil

func init() {
	switch env.GetEnv() {
	/*
		Can split the logging functionality based on env: env.PROD, env.STG, env.DEV
	*/
	case env.PROD:
		defaultLogger = Logger{
			sclogHandler: nil,
			logger:       slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{})),
		}
	default:
		clogHandler := sclog.NewHandler(sclog.NewHandlerRequest{
			StdOut:    os.Stdout,
			StdErr:    os.Stderr,
			Level:     slog.LevelDebug,
			Color:     true,
			Timestamp: true,
		})
		defaultLogger = Logger{
			logger:       slog.New(clogHandler),
			sclogHandler: &clogHandler,
		}
	}
}

type LoggerI interface {
	Debug(ctx context.Context, msg string, args ...interface{})
	Info(ctx context.Context, msg string, args ...interface{})
	Warn(ctx context.Context, msg string, args ...interface{})
	Error(ctx context.Context, msg string, args ...interface{})
	None(ctx context.Context, msg string, args ...interface{})

	DebugWithoutCtx(msg string, args ...interface{})
	InfoWithoutCtx(msg string, args ...interface{})
	WarnWithoutCtx(msg string, args ...interface{})
	ErrorWithoutCtx(msg string, args ...interface{})
	NoneWithoutCtx(msg string, args ...interface{})

	WithHeading(heading string) LoggerI
}

type Logger struct {
	sclogHandler *sclog.Handler
	logger       *slog.Logger
}

func (l Logger) Debug(ctx context.Context, msg string, args ...interface{}) {
	l.logger.DebugContext(ctx, msg, args...)
}
func (l Logger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.logger.InfoContext(ctx, msg, args...)
}
func (l Logger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.logger.WarnContext(ctx, msg, args...)
}
func (l Logger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.logger.ErrorContext(ctx, msg, args...)
}
func (l Logger) None(ctx context.Context, msg string, args ...interface{}) {}

func (l Logger) DebugWithoutCtx(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}
func (l Logger) InfoWithoutCtx(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}
func (l Logger) WarnWithoutCtx(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}
func (l Logger) ErrorWithoutCtx(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}
func (l Logger) NoneWithoutCtx(msg string, args ...interface{}) {}

func (l Logger) WithHeading(heading string) LoggerI {
	panics.IfNil(l.sclogHandler, "Cannot set heading on a logger that does not have a sclog handler")

	handler := l.sclogHandler.WithHeading(heading)
	return Logger{
		sclogHandler: &handler,
		logger:       slog.New(handler),
	}
}

// Default Logger methods

// GetLogger returns the logger instance
func GetLogger() LoggerI {
	return defaultLogger
}

func Debug(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.Debug(ctx, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	defaultLogger.Info(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.Warn(ctx, msg, args...)
}

func None(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.None(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.Error(ctx, msg, args...)
}

func DebugWithoutCtx(msg string, args ...interface{}) {
	defaultLogger.DebugWithoutCtx(msg, args...)
}

func InfoWithoutCtx(msg string, args ...interface{}) {
	defaultLogger.InfoWithoutCtx(msg, args...)
}

func WarnWithoutCtx(msg string, args ...interface{}) {
	defaultLogger.WarnWithoutCtx(msg, args...)
}

func ErrorWithoutCtx(msg string, args ...interface{}) {
	defaultLogger.ErrorWithoutCtx(msg, args...)
}

func NoneWithoutCtx(msg string, args ...interface{}) {
	defaultLogger.NoneWithoutCtx(msg, args...)
}
