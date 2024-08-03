package envutil

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/teejays/gokutil/env"
	"github.com/teejays/gokutil/log"
)

func LoadEnvFiles(ctx context.Context) error {
	// Load the env file
	_env := env.GetEnv()

	filePath := ".env." + string(_env) + ".local"
	log.Info(ctx, "Loading env", "file", filePath)
	err := godotenv.Load(filePath)
	if err != nil {
		log.Warn(ctx, "Could not load env file", "file", filePath, "error", err)
	}

	if _env != env.TEST {
		filePath = ".env.local"
		log.Info(ctx, "Loading env", "file", filePath)
		err = godotenv.Load(filePath)
		if err != nil {
			log.Warn(ctx, "Could not load env file", "file", filePath, "error", err)
		}
	}
	filePath = ".env." + string(_env)
	log.Info(ctx, "Loading env", "file", filePath)
	err = godotenv.Load(".env." + string(_env))
	if err != nil {
		log.Warn(ctx, "Could not load env file", "file", filePath, "error", err)
	}

	log.Info(ctx, "Loading env", "file", ".env")
	err = godotenv.Load() // The Original .env
	if err != nil {
		log.Warn(ctx, "Could not load env file", "file", ".env", "error", err)
	}
	return nil
}
