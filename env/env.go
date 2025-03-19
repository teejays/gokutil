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
	UNKNOWN   Environment = "unknown"
	DEV       Environment = "dev"       // Any dev environments
	DEV_LOCAL Environment = "dev_local" // Local dev
	DEV_CLOUD Environment = "dev_cloud" // Cloud dev
	TEST      Environment = "test"      // For running tests
	STG       Environment = "stage"
	PROD      Environment = "prod"
)

func GetEnv() Environment {
	// Check the APP_ENV environment variable
	env := UNKNOWN
	switch strings.ToLower(os.Getenv("APP_ENV")) {
	case "production", "prod", "prd":
		env = PROD
	case "staging", "stage", "stg":
		env = STG
	case "development", "dev":
		env = DEV
	case "local", "local-dev", "local-development", "local_dev", "local_development":
		env = DEV_LOCAL
	case "cloud", "cloud-dev", "cloud-development", "cloud_dev", "cloud_development":
		env = DEV_CLOUD
	case "testing", "test":
		env = TEST
	case "":
		env = UNKNOWN
	default:
		// Could potentially error/warn
		panics.P("Unknown APP_ENV environment variable value [%s]", os.Getenv("APP_ENV"))
	}

	return env
}

func SetEnv(e Environment) {
	os.Setenv("APP_ENV", e.String())
}

func IsDev() bool {
	return GetEnv() == DEV || GetEnv() == DEV_LOCAL || GetEnv() == DEV_CLOUD
}
