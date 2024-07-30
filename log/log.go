package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/teejays/goku-util/env"
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
		logger = slog.Default()
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
}

func Debug(ctx context.Context, msg string, args ...interface{}) {
	logger.DebugContext(ctx, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	logger.InfoContext(ctx, msg, args...)
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

func ErrorNoCtx(msg string, args ...interface{}) {
	Error(context.Background(), msg, args...)
}
