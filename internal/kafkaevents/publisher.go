package kafkaevents

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/anshsaluja/finlink/internal/syncs"
	"github.com/segmentio/kafka-go"
)

// Publisher writes durable sync requests to Kafka. The run ID is used as the
// message key so retries for a run are consistently partitioned.
type Publisher struct {
	writer *kafka.Writer
}

func NewPublisher(brokers []string, topic string) *Publisher {
	return &Publisher{writer: &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
	}}
}

func (p *Publisher) PublishSyncRequested(ctx context.Context, event syncs.RequestedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal sync.requested: %w", err)
	}
	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(strconv.FormatInt(event.RunID, 10)),
		Value: payload,
	}); err != nil {
		return fmt.Errorf("write sync.requested: %w", err)
	}
	return nil
}

func (p *Publisher) Close() error { return p.writer.Close() }
