package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr           string
	DatabaseURL        string
	KafkaBrokers       []string
	KafkaTopic         string
	KafkaConsumerGroup string
	KafkaDLQTopic      string
	SyncMaxAttempts    int
	SyncRetryBackoff   time.Duration
}

func Load() (Config, error) {
	attempts, err := strconv.Atoi(envOrDefault("FINLINK_SYNC_MAX_ATTEMPTS", "3"))
	if err != nil || attempts < 1 {
		return Config{}, fmt.Errorf("FINLINK_SYNC_MAX_ATTEMPTS must be a positive integer")
	}
	backoff, err := time.ParseDuration(envOrDefault("FINLINK_SYNC_RETRY_BACKOFF", "2s"))
	if err != nil || backoff < 0 {
		return Config{}, fmt.Errorf("FINLINK_SYNC_RETRY_BACKOFF must be a non-negative duration")
	}
	cfg := Config{
		HTTPAddr:           envOrDefault("FINLINK_HTTP_ADDR", ":8080"),
		DatabaseURL:        envOrDefault("FINLINK_DATABASE_URL", "postgres://finlink:finlink@localhost:5432/finlink?sslmode=disable"),
		KafkaBrokers:       splitCSV(envOrDefault("FINLINK_KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:         envOrDefault("FINLINK_KAFKA_SYNC_TOPIC", "finlink.sync.requested"),
		KafkaConsumerGroup: envOrDefault("FINLINK_KAFKA_CONSUMER_GROUP", "finlink-sync-workers"),
		KafkaDLQTopic:      envOrDefault("FINLINK_KAFKA_DLQ_TOPIC", "finlink.sync.dead-letter"),
		SyncMaxAttempts:    attempts,
		SyncRetryBackoff:   backoff,
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
