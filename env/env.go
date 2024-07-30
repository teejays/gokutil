package env

import (
	"os"
)

type Environment int

const (
	_ = iota
	DEV
	STG
	PROD
)

var (
	env           Environment
	isInitialized bool
)

func Init() {
	switch os.Getenv("APP_ENV") {
	case "PROD":
		env = PROD
	case "STG":
		env = STG
	case "DEV":
		env = DEV
	default:
		// Could potentially error/warn
		env = DEV
	}
	isInitialized = true
}

func GetEnv() Environment {
	if !isInitialized {
		Init()
	}
	return env
}
