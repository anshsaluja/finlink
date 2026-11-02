package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	KafkaBrokers []string
	KafkaTopic  string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr: envOrDefault("FINLINK_HTTP_ADDR", ":8080"),
		DatabaseURL: envOrDefault("FINLINK_DATABASE_URL", "postgres://finlink:finlink@localhost:5432/finlink?sslmode=disable"),
		KafkaBrokers: splitCSV(envOrDefault("FINLINK_KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic: envOrDefault("FINLINK_KAFKA_SYNC_TOPIC", "finlink.sync.requested"),
	}
	if len(cfg.KafkaBrokers) == 0 {
		return Config{}, fmt.Errorf("FINLINK_KAFKA_BROKERS must contain at least one broker")
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return Config{}, fmt.Errorf("FINLINK_DATABASE_URL is required")
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if broker := strings.TrimSpace(part); broker != "" {
			result = append(result, broker)
		}
	}
	return result
}
