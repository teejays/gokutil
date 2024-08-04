package main

import (
	"log/slog"
	"os"

	"github.com/teejays/gokutil/clog/sclog"
)

func main() {
	devHandler := sclog.NewHandler(sclog.NewHandlerRequest{
		Out:       os.Stderr,
		Level:     slog.LevelDebug,
		Color:     true,
		Timestamp: true,
	})
	logger := slog.New(devHandler)

	// Testing
	type name struct {
		first string
		last  string
	}

	logger.Debug("This is a debug message")
	logger.Debug("This is a debug message with args", "name", name{first: "John", last: "Doe"}, "age", 34, "isMarried", true, "height", 5.9)

	logger.Info("This is an info message")
	logger.Info("This is an info message with args", "name", name{first: "John", last: "Doe"}, "age", 34, "isMarried", true, "height", 5.9)

	logger.Warn("This is a warning message")
	logger.Warn("This is a warning message with args", "name", name{first: "John", last: "Doe"}, "age", 34, "isMarried", true, "height", 5.9)

	logger.Error("This is an error message")
	logger.Error("This is an error message with args", "name", name{first: "John", last: "Doe"}, "age", 34, "isMarried", true, "height", 5.9)
}
