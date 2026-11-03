package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/anshsaluja/finlink/internal/config"
	"github.com/anshsaluja/finlink/internal/connections"
	"github.com/anshsaluja/finlink/internal/financialdata"
	"github.com/anshsaluja/finlink/internal/provider"
	"github.com/anshsaluja/finlink/internal/syncs"
	"github.com/anshsaluja/finlink/internal/syncworker"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		log.Fatalf("ping postgres: %v", err)
	}
	cancel()

	registry := provider.NewRegistry(provider.MockNorthstar{}, provider.MockHarbor{})
	connectionsService := connections.NewServiceWithRepository(registry, connections.NewSQLRepository(db))
	worker := syncworker.New(connectionsService, financialdata.NewSQLRepository(db), syncs.NewSQLRepository(db))
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.KafkaBrokers,
		Topic:    cfg.KafkaTopic,
		GroupID:  cfg.KafkaConsumerGroup,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  time.Second,
	})
	defer reader.Close()
	dlq := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBrokers...),
		Topic:        cfg.KafkaDLQTopic,
		RequiredAcks: kafka.RequireAll,
	}
	defer dlq.Close()

	log.Printf("finlink sync worker consuming %s", cfg.KafkaTopic)
	for {
		message, err := reader.FetchMessage(context.Background())
		if err != nil {
			log.Fatalf("fetch kafka message: %v", err)
		}

		var event syncs.RequestedEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			if err := deadLetter(context.Background(), dlq, message, 0, fmt.Errorf("decode sync event: %w", err)); err != nil {
				log.Printf("dead-letter invalid event: %v", err)
				continue
			}
			if err := reader.CommitMessages(context.Background(), message); err != nil {
				log.Printf("commit invalid event: %v", err)
			}
			continue
		}

		var processErr error
		for attempt := 1; attempt <= cfg.SyncMaxAttempts; attempt++ {
			processErr = worker.Process(context.Background(), event)
			if processErr == nil {
				break
			}
			log.Printf("sync run %d attempt %d/%d failed: %v", event.RunID, attempt, cfg.SyncMaxAttempts, processErr)
			if attempt < cfg.SyncMaxAttempts {
				time.Sleep(cfg.SyncRetryBackoff * time.Duration(attempt))
			}
		}

		if processErr != nil {
			if err := deadLetter(context.Background(), dlq, message, cfg.SyncMaxAttempts, processErr); err != nil {
				log.Printf("dead-letter sync run %d: %v", event.RunID, err)
				continue
			}
			if err := reader.CommitMessages(context.Background(), message); err != nil {
				log.Printf("commit dead-lettered run %d: %v", event.RunID, err)
			}
			continue
		}

		if err := reader.CommitMessages(context.Background(), message); err != nil {
			log.Printf("commit kafka message for run %d: %v", event.RunID, err)
			continue
		}
		log.Printf("sync run %d completed", event.RunID)
	}
}

func deadLetter(ctx context.Context, writer *kafka.Writer, source kafka.Message, attempts int, cause error) error {
	headers := append([]kafka.Header{}, source.Headers...)
	headers = append(
		headers,
		kafka.Header{Key: "finlink-source-topic", Value: []byte(source.Topic)},
		kafka.Header{Key: "finlink-source-partition", Value: []byte(strconv.Itoa(source.Partition))},
		kafka.Header{Key: "finlink-source-offset", Value: []byte(strconv.FormatInt(source.Offset, 10))},
		kafka.Header{Key: "finlink-attempts", Value: []byte(strconv.Itoa(attempts))},
		kafka.Header{Key: "finlink-error", Value: []byte(cause.Error())},
	)
	return writer.WriteMessages(ctx, kafka.Message{
		Key:     source.Key,
		Value:   source.Value,
		Headers: headers,
		Time:    time.Now().UTC(),
	})
}
