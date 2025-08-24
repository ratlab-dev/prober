package kafka

import (
	"context"
	"fmt"
)

type ReadProbe struct {
	Region  string
	Brokers []string
	Topic   string
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	// TODO: Implement Kafka read health check
	return nil
}

func (p *ReadProbe) MetadataString() string {
	return fmt.Sprintf("Brokers: %v , Topic: %s , Region: %s", p.Brokers, p.Topic, p.Region)
}

type WriteProbe struct {
	Region  string
	Brokers []string
	Topic   string
}

func (p *WriteProbe) Probe(ctx context.Context) error {
	// TODO: Implement Kafka write health check
	return nil
}

func (p *WriteProbe) MetadataString() string {
	return fmt.Sprintf("Brokers: %v , Topic: %s , Region: %s", p.Brokers, p.Topic, p.Region)
}
