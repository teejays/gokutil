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

func LoadGokuEnvFiles(ctx context.Context, wd string) error {

	err := LoadEnvFilesV2(ctx, []string{
		filepath.Join(wd, ".env.goku.local"),
		filepath.Join(wd, ".env.goku"),
	})
	if err != nil {
		return errutil.Wrap(err, "Could not load goku env files")
	}

	// Read app env after loading above files since they may declare the app env
	_env := env.GetEnv()
	return LoadEnvFilesV2(ctx, []string{
		filepath.Join(wd, ".env.goku."+string(_env)),
	})

}

func LoadAppEnvFiles(ctx context.Context, wd string) error {
	_env := env.GetEnv()

	return LoadEnvFilesV2(ctx, []string{
		filepath.Join(wd, ".env"),
		filepath.Join(wd, ".env."+string(_env)),
		filepath.Join(wd, ".env.app"),
		filepath.Join(wd, ".env.app."+string(_env)),
	})
}

func LoadEnvFilesMatch(ctx context.Context, wd string, patterns []string) error {

	files := []string{}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return errutil.Wrap(err, "Could not match pattern [%s]", pattern)
		}
		files = append(files, matches...)
	}

	// Make sure the files are unique
	uniqueFiles := map[string]struct{}{}
	for _, file := range files {
		uniqueFiles[file] = struct{}{}
	}

	files = []string{}
	for file := range uniqueFiles {
		files = append(files, file)
	}

	return LoadEnvFilesV2(ctx, files)

}

// LoadEnvFilesV2 loads the env files into the environment as env vars
func LoadEnvFilesV2(ctx context.Context, filesPaths []string) error {
	log.Debug(ctx, "Loading env files", "files", filesPaths)
	// Load the env file
	for _, file := range filesPaths {
		log.None(ctx, "Looking for env file", "file", file)
		err := godotenv.Load(file)
		if err != nil {
			// If the error is that the file does not exist, that's fine. We can continue.
			if os.IsNotExist(err) {
				log.None(ctx, "Env file does not exist", "file", file)
				continue
			}
			return errutil.Wrap(err, "Could not load env file [%s]", file)
		} else {
			log.Debug(ctx, "Loaded env file", "file", file)
		}
	}

	return nil

}

// // LoadEnvFiles loads the env files into the environment as env vars
// func LoadEnvFiles(ctx context.Context, wd string) error {
// 	// Load the env file
// 	_env := env.GetEnv()

// 	if wd == "" {
// 		wd = "."
// 	}

// 	files := []string{
// 		".env.goku",
// 		".env.goku." + string(_env),
// 		".env." + string(_env) + ".local",
// 		".env.local",
// 		".env." + string(_env),
// 		".env",
// 	}
// 	for i, file := range files {
// 		files[i] = filepath.Join(wd, file)
// 	}
// 	for _, file := range files {
// 		log.None(ctx, "Looking for env file", "file", file)
// 		err := godotenv.Load(file)
// 		if err != nil {
// 			// If the error is that the file does not exist, that's fine. We can continue.
// 			if os.IsNotExist(err) {
// 				log.None(ctx, "Env file does not exist", "file", file)
// 				continue
// 			}
// 			return errutil.Wrap(err, "Could not load env file [%s]", file)
// 		} else {
// 			log.Debug(ctx, "Loaded env file", "file", file)
// 		}
// 	}

// 	return nil
// }

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
