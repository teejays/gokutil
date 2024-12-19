package envutil

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/teejays/gokutil/env"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"
)

// LoadEnvFiles loads the env files into the environment as env vars
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
		log.None(ctx, "Looking for env file", "file", file)
		err := godotenv.Load(file)
		if err != nil {
			log.None(ctx, "Could not load env file", "file", file, "error", err)
		} else {
			log.Debug(ctx, "Loaded env file", "file", file)
		}
	}

	return nil
}

func GetEnvVarStr(key string) string {
	val := os.Getenv(key)
	return val
}

func MustGetEnvVarStr(key string) string {
	val := os.Getenv(key)
	panics.If(val == "", "An env var is not found [%s]", key)
	return val
}

func GetEnvVarStrOrDefault(ctx context.Context, key, defaultVal string) string {
	val := GetEnvVarStr(key)
	if val == "" {
		log.Warn(ctx, "Env is not set, using default value", "env", key, "default", defaultVal)
		return defaultVal
	}
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

func GetEnvVarBool(key string) (bool, error) {
	val := os.Getenv(key)
	if val == "" {
		return false, nil
	}
	valBool, err := strconv.ParseBool(val)
	if err != nil {
		return false, errutil.Wrap(err, "Expected env variable [%s] to be a boolean or empty, got [%s]", key, val)
	}
	return valBool, nil
}

func MustGetEnvVarBool(key string) bool {
	val, err := GetEnvVarBool(key)
	panics.IfError(err, "An env var is invalid")
	return val
}
