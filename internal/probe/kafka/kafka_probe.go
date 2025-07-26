package kafka

import "context"

type ReadProbe struct {
	Brokers []string
	Topic   string
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	// TODO: Implement Kafka read health check
	return nil
}

type WriteProbe struct {
	Brokers []string
	Topic   string
}

func (p *WriteProbe) Probe(ctx context.Context) error {
	// TODO: Implement Kafka write health check
	return nil
}
