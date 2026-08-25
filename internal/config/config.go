package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL string
	HTTPAddress string
	LogLevel    string

	RedisAddress  string
	RedisPassword string
	RedisDatabase int
	RoomCacheTTL  time.Duration

	NATSURL string
}

func Load() (*Config, error) {
	redisDatabase, err := strconv.Atoi(
		getEnv("REDIS_DATABASE", "0"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse REDIS_DATABASE: %w",
			err,
		)
	}

	roomCacheTTL, err := time.ParseDuration(
		getEnv("ROOM_CACHE_TTL", "5m"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse ROOM_CACHE_TTL: %w",
			err,
		)
	}

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddress: getEnv("HTTP_ADDRESS", ":8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		RedisAddress:  getEnv("REDIS_ADDRESS", "localhost:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDatabase: redisDatabase,
		RoomCacheTTL:  roomCacheTTL,
		NATSURL:       getEnv("NATS_URL", "nats://localhost:4222"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
