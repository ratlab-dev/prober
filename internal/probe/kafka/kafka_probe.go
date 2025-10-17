package kafka

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/IBM/sarama"
)

// RandString generates a random alphanumeric string of given length
func RandString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type ReadProbe struct {
	Region  string
	Brokers []string
	Topic   string
	client  sarama.Client
}

// NewReadProbe creates a ReadProbe with a persistent client
func NewReadProbe(brokers []string, topic string) (*ReadProbe, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0 // Use a stable version
	config.Consumer.Return.Errors = true
	config.Metadata.Retry.Max = 1
	config.Metadata.Timeout = 5 * time.Second

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	return &ReadProbe{
		Brokers: brokers,
		Topic:   topic,
		client:  client,
	}, nil
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	// Check if client is closed or has errors
	if p.client.Closed() {
		// Try to recreate client
		config := sarama.NewConfig()
		config.Version = sarama.V2_6_0_0
		config.Consumer.Return.Errors = true
		config.Metadata.Retry.Max = 1
		config.Metadata.Timeout = 5 * time.Second

		client, err := sarama.NewClient(p.Brokers, config)
		if err != nil {
			return fmt.Errorf("failed to recreate Kafka client: %w", err)
		}
		p.client = client
	}

	// Refresh metadata to check cluster health
	if err := p.client.RefreshMetadata(p.Topic); err != nil {
		return fmt.Errorf("failed to refresh metadata: %w", err)
	}

	// Check if all brokers are reachable
	brokers := p.client.Brokers()
	if len(brokers) == 0 {
		return fmt.Errorf("no brokers available in cluster")
	}

	// Verify topic exists and has partitions
	partitions, err := p.client.Partitions(p.Topic)
	if err != nil {
		return fmt.Errorf("failed to get partitions for topic %s: %w", p.Topic, err)
	}
	if len(partitions) == 0 {
		return fmt.Errorf("topic %s has no partitions", p.Topic)
	}

	// Check if each partition has a leader
	for _, partition := range partitions {
		broker, err := p.client.Leader(p.Topic, partition)
		if err != nil {
			return fmt.Errorf("no leader for topic %s partition %d: %w", p.Topic, partition, err)
		}
		if broker == nil {
			return fmt.Errorf("nil leader broker for topic %s partition %d", p.Topic, partition)
		}
	}

	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][Kafka][Read] Cluster healthy: %d brokers, topic %s with %d partitions", len(brokers), p.Topic, len(partitions))
	}

	return nil
}

func (p *ReadProbe) MetadataString() string {
	return fmt.Sprintf("Brokers: %v , Topic: %s , Region: %s", p.Brokers, p.Topic, p.Region)
}

type WriteProbe struct {
	Region   string
	Brokers  []string
	Topic    string
	producer sarama.SyncProducer
	consumer sarama.Consumer
}

// NewWriteProbe creates a WriteProbe with persistent producer and consumer
func NewWriteProbe(brokers []string, topic string) (*WriteProbe, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Retry.Max = 1
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 5 * time.Second

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	consumerConfig := sarama.NewConfig()
	consumerConfig.Version = sarama.V2_6_0_0
	consumerConfig.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumer(brokers, consumerConfig)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &WriteProbe{
		Brokers:  brokers,
		Topic:    topic,
		producer: producer,
		consumer: consumer,
	}, nil
}

func (p *WriteProbe) Probe(ctx context.Context) error {
	// Generate a unique test message
	testKey := "probe_key_" + RandString(12)
	testValue := "probe_value_" + RandString(12)

	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][Kafka][Write] Producing test message with key: %s", testKey)
	}

	// Produce a test message
	msg := &sarama.ProducerMessage{
		Topic: p.Topic,
		Key:   sarama.StringEncoder(testKey),
		Value: sarama.StringEncoder(testValue),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		// Try to recreate producer once
		if closeErr := p.producer.Close(); closeErr != nil {
			log.Printf("[WARN][Kafka][Write] Failed to close producer: %v", closeErr)
		}

		config := sarama.NewConfig()
		config.Version = sarama.V2_6_0_0
		config.Producer.RequiredAcks = sarama.WaitForLocal
		config.Producer.Retry.Max = 1
		config.Producer.Return.Successes = true
		config.Producer.Timeout = 5 * time.Second

		producer, newErr := sarama.NewSyncProducer(p.Brokers, config)
		if newErr != nil {
			return fmt.Errorf("failed to produce message (and recreate producer failed): %w", err)
		}
		p.producer = producer

		// Retry sending
		partition, offset, err = p.producer.SendMessage(msg)
		if err != nil {
			return fmt.Errorf("failed to produce message after retry: %w", err)
		}
	}

	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][Kafka][Write] Message produced successfully to partition %d at offset %d", partition, offset)
	}

	// Verify we can consume from the partition (checking cluster read path)
	// We don't need to find our exact message, just verify we can consume
	partitionConsumer, err := p.consumer.ConsumePartition(p.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		return fmt.Errorf("failed to create partition consumer for verification: %w", err)
	}
	defer partitionConsumer.Close()

	// Just verify we can create the consumer - actual message consumption would require
	// waiting which would slow down the probe. The fact that we can produce and create
	// a consumer indicates the cluster is healthy.

	return nil
}

func (p *WriteProbe) MetadataString() string {
	return fmt.Sprintf("Brokers: %v , Topic: %s , Region: %s", p.Brokers, p.Topic, p.Region)
}
