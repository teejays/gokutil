package env

import (
	"os"

	"github.com/teejays/gokutil/panics"
)

type Environment ENV

type ENV string

const (
	DEV  = "development"
	TEST = "testing" // For running tests
	STG  = "staging"
	PROD = "production"
)

var (
	env           Environment
	isInitialized bool
)

func Init() {
	switch os.Getenv("APP_ENV") {
	case "production":
		env = PROD
	case "staging":
		env = STG
	case "development":
		env = DEV
	case "testing":
		env = TEST
	case "":
		env = DEV
	default:
		// Could potentially error/warn
		panics.P("Unknown APP_ENV environment variable value [%s]", os.Getenv("APP_ENV"))
	}
	isInitialized = true
}

func GetEnv() Environment {
	if !isInitialized {
		Init()
	}
	return env
}
