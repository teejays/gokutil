// Package log provides a simple logging interface that can be used to log messages to the console.
// It implements a singleton pattern.
package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/teejays/gokutil/env"
	"github.com/teejays/gokutil/panics"
	"github.com/teejays/gokutil/sclog"
)

// LevelTrace is a custom log level that is lower than Debug.
// Other log levels are defined in slog package, with the values of -4 (debug) to 8 (error) in increments of 4.
var LevelTrace slog.Level = -8

var _logLevel = slog.LevelDebug // default to debug
func GetLogLevel() slog.Level {
	return _logLevel
}

var defaultLogger LoggerI = nil

func ParseLevel(levelStr string) (slog.Level, error) {
	return parseLevel(levelStr)
}

func parseLevel(levelStr string) (slog.Level, error) {

	switch strings.ToLower(levelStr) {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		// Use the slog package to parse the level
		var lvl slog.Level
		err := lvl.UnmarshalText([]byte(levelStr))
		if err != nil {
			return slog.LevelDebug, fmt.Errorf("Invalid log level: %s", levelStr)
		}
		return lvl, nil
	}
}

func mustParseLevel(levelStr string) slog.Level {
	level, err := parseLevel(levelStr)
	panics.IfError(err, "Parsing log level")
	return level
}

func init() {
	logLevel := _logLevel
	if logLevelStr := os.Getenv("GOKU_LOG_LEVEL"); logLevelStr != "" {
		logLevel = mustParseLevel(logLevelStr)
	}
	Init(logLevel)
}

func Init(logLevel slog.Level) {

	fmt.Printf("Initializing logging with level: %s\n", logLevel)

	switch env.GetEnv() {
	/*
		Can split the logging functionality based on env: env.PROD, env.STG, env.DEV
	*/
	case env.PROD:
		defaultLogger = Logger{
			logger: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{AddSource: true, Level: logLevel})),
		}
	default:
		clogHandler := sclog.NewHandler(sclog.NewHandlerRequest{
			StdOut:    os.Stdout,
			StdErr:    os.Stderr,
			Level:     logLevel,
			Color:     true,
			Timestamp: true,
		})
		defaultLogger = Logger{
			logger: slog.New(clogHandler),
		}
	}

	if _logLevel != logLevel {
		_logLevel = logLevel
	}
}

type LoggerI interface {
	Trace(ctx context.Context, msg string, args ...interface{})
	Debug(ctx context.Context, msg string, args ...interface{})
	Info(ctx context.Context, msg string, args ...interface{})
	Warn(ctx context.Context, msg string, args ...interface{})
	Error(ctx context.Context, msg string, args ...interface{})
	None(ctx context.Context, msg string, args ...interface{})

	TraceWithoutCtx(msg string, args ...interface{})
	DebugWithoutCtx(msg string, args ...interface{})
	InfoWithoutCtx(msg string, args ...interface{})
	WarnWithoutCtx(msg string, args ...interface{})
	ErrorWithoutCtx(msg string, args ...interface{})
	NoneWithoutCtx(msg string, args ...interface{})

	WithHeading(heading string) LoggerI
}

type Logger struct {
	logger *slog.Logger
}

func (l Logger) Trace(ctx context.Context, msg string, args ...interface{}) {
	l.logger.Log(ctx, LevelTrace, msg, args...)
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

func (l Logger) TraceWithoutCtx(msg string, args ...interface{}) {
	l.logger.Log(context.Background(), LevelTrace, msg, args...)
}

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

	handler := l.logger.Handler()
	if sclogHandler, ok := handler.(sclog.Handler); ok {
		newHandler := sclogHandler.WithHeading(heading)
		return Logger{
			logger: slog.New(newHandler),
		}
	}

	return l
}

// Default Logger methods

// GetLogger returns the logger instance
func GetLogger() LoggerI {
	return defaultLogger
}

func Trace(ctx context.Context, msg string, args ...interface{}) {
	defaultLogger.Trace(ctx, msg, args...)
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

func TraceWithoutCtx(msg string, args ...interface{}) {
	defaultLogger.TraceWithoutCtx(msg, args...)
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
