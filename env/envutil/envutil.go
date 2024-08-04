package envutil

import (
	"context"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/teejays/gokutil/env"
	"github.com/teejays/gokutil/log"
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
