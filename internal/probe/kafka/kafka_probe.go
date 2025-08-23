package kafka

import (
	"context"
	"fmt"
)

type ReadProbe struct {
	Brokers []string
	Topic   string
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	// TODO: Implement Kafka read health check
	return nil
}

func (p *ReadProbe) MetadataString() string {
	return fmt.Sprintf("KafkaReadProbe | Brokers: %v | Topic: %s", p.Brokers, p.Topic)
}

type WriteProbe struct {
	Brokers []string
	Topic   string
}

func (p *WriteProbe) Probe(ctx context.Context) error {
	// TODO: Implement Kafka write health check
	return nil
}

func (p *WriteProbe) MetadataString() string {
	return fmt.Sprintf("KafkaWriteProbe | Brokers: %v | Topic: %s", p.Brokers, p.Topic)
}
