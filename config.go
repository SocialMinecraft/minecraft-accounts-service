package main

import (
	"errors"
	"os"
)

type Config struct {
	NatsUrl     string
	PostgresUrl string
}

func getConfig() (Config, error) {
	config := Config{
		os.Getenv("NATS_URL"),
		os.Getenv("POSTGRES_URL"),
	}

	if len(config.NatsUrl) <= 0 {
		return config, errors.New("NATS_URL missing")
	}
	if len(config.PostgresUrl) <= 0 {
		return config, errors.New("POSTGRES_URL is missing, example: postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full")
	}

	return config, nil
}
