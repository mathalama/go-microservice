package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

type NatsPublisher struct {
	nc *nats.Conn
}

func NewNatsPublisher(nc *nats.Conn) *NatsPublisher {
	return &NatsPublisher{nc: nc}
}

func (p *NatsPublisher) Publish(ctx context.Context, subject string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}

	return nil
}
