package env

import (
	"os"
	"strings"

	"github.com/teejays/gokutil/panics"
)

type Environment string

func (e Environment) String() string {
	return string(e)
}

const (
	UNKNOWN Environment = "unknown"
	DEV     Environment = "dev"
	TEST    Environment = "test" // For running tests
	STG     Environment = "stage"
	PROD    Environment = "prod"
)

var (
	env           Environment
	isInitialized bool
)

func Init() {
	switch strings.ToLower(os.Getenv("APP_ENV")) {
	case "production", "prod", "prd":
		env = PROD
	case "staging", "stage", "stg":
		env = STG
	case "development", "dev":
		env = DEV
	case "testing", "test":
		env = TEST
	case "":
		env = UNKNOWN
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

func IsDev() bool {
	return GetEnv() == DEV
}
