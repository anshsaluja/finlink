package config

import (
	"reflect"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"FINLINK_HTTP_ADDR", "FINLINK_DATABASE_URL", "FINLINK_KAFKA_BROKERS", "FINLINK_KAFKA_SYNC_TOPIC"} {
		t.Setenv(key, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://finlink:finlink@localhost:5432/finlink?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if !reflect.DeepEqual(cfg.KafkaBrokers, []string{"localhost:9092"}) {
		t.Fatalf("KafkaBrokers = %#v", cfg.KafkaBrokers)
	}
}

func TestLoadParsesKafkaBrokers(t *testing.T) {
	t.Setenv("FINLINK_KAFKA_BROKERS", "kafka-a:9092, kafka-b:9092")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"kafka-a:9092", "kafka-b:9092"}
	if !reflect.DeepEqual(cfg.KafkaBrokers, want) {
		t.Fatalf("KafkaBrokers = %#v, want %#v", cfg.KafkaBrokers, want)
	}
}
