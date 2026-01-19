package config

import (
	"os"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

func LoadEnvironment() Environment {
	env := os.Getenv(ENV_PREFIX + "_ENV")

	switch env {
	case string(EnvDevelopment):
		return EnvDevelopment
	case string(EnvProduction), "":
		return EnvProduction
	default:
		panic("invalid " + ENV_PREFIX + "_ENV: " + env)
	}
}
