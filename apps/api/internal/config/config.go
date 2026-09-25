// Package config reads the API's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"strconv"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

const defaultPort = 8080

type Config struct {
	DatabaseURL string
	Port        int
	Env         string
}

func (c Config) IsProduction() bool {
	return c.Env == EnvProduction
}

// Load builds a Config from getenv, which is os.Getenv outside tests.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DatabaseURL: getenv("DATABASE_URL"),
		Port:        defaultPort,
		Env:         EnvDevelopment,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	if v := getenv("PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("PORT must be a number from 1 to 65535, got %q", v)
		}
		cfg.Port = port
	}

	// An unknown value is rejected rather than treated as development:
	// a typo such as "prod" would otherwise silently drop production
	// settings like Secure cookies.
	if v := getenv("APP_ENV"); v != "" {
		if v != EnvDevelopment && v != EnvProduction {
			return Config{}, fmt.Errorf("APP_ENV must be %q or %q, got %q", EnvDevelopment, EnvProduction, v)
		}
		cfg.Env = v
	}

	return cfg, nil
}
