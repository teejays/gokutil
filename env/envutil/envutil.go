package envutil

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/teejays/gokutil/env"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"
)

func LoadEnvFiles(ctx context.Context, wd string) error {
	// Load the env file
	_env := env.GetEnv()

	if wd == "" {
		wd = "."
	}

	files := []string{
		".env." + string(_env) + ".local",
		".env.local",
		".env." + string(_env),
		".env",
	}
	for i, file := range files {
		files[i] = filepath.Join(wd, file)
	}
	for _, file := range files {
		log.Debug(ctx, "Loading env file", "file", file)
		err := godotenv.Load(file)
		if err != nil {
			log.Debug(ctx, "Could not load env file", "file", file, "error", err)
		}
	}

	return nil
}

func GetEnvVarStr(key string) string {
	val := os.Getenv(key)
	return val
}

// GetEnvVarInt returns 0 if the variable is not set
func GetEnvVarInt(key string) int {
	val := os.Getenv(key)
	if val == "" {
		return 0
	}
	valInt, err := strconv.Atoi(val)
	panics.IfError(err, "Expected env variable [%s] to be a number", key)
	return valInt
}

func MustGetEnvVarStr(key string) string {
	val := os.Getenv(key)
	panics.If(val == "", "Env var not found [%s]", key)
	return val
}
