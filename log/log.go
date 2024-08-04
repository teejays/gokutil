package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/teejays/gokutil/clog/sclog"
	"github.com/teejays/gokutil/env"
)

var logger *slog.Logger = slog.Default()

func Init() {
	switch env.GetEnv() {
	/*
		Can split the logging functionality based on env: env.PROD, env.STG, env.DEV
	*/
	case env.PROD:
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{}))
	default:
		devHandler := sclog.NewHandler(sclog.NewHandlerRequest{
			Out:       os.Stderr,
			Level:     slog.LevelDebug,
			Color:     true,
			Timestamp: true,
		})
		logger = slog.New(devHandler)
	}
}

func Debug(ctx context.Context, msg string, args ...interface{}) {
	logger.DebugContext(ctx, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	logger.InfoContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...interface{}) {
	logger.WarnContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...interface{}) {
	logger.ErrorContext(ctx, msg, args...)
}

func DebugNoCtx(msg string, args ...interface{}) {
	Debug(context.Background(), msg, args...)
}

func InfoNoCtx(msg string, args ...interface{}) {
	Info(context.Background(), msg, args...)
}

func WarnNoCtx(msg string, args ...interface{}) {
	Warn(context.Background(), msg, args...)
}

func ErrorNoCtx(msg string, args ...interface{}) {
	Error(context.Background(), msg, args...)
}
