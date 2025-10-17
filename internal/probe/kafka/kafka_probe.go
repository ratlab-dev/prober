package kafka

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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
	client  *kafka.AdminClient
}

// NewReadProbe creates a ReadProbe with a persistent admin client
func NewReadProbe(brokers []string, topic string) (*ReadProbe, error) {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(brokers, ","),
		"socket.timeout.ms": 5000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka admin client: %w", err)
	}

	return &ReadProbe{
		Brokers: brokers,
		Topic:   topic,
		client:  adminClient,
	}, nil
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	// Get cluster metadata to check health
	metadata, err := p.client.GetMetadata(&p.Topic, false, 5000)
	if err != nil {
		// Try to recreate client once
		p.client.Close()
		adminClient, newErr := kafka.NewAdminClient(&kafka.ConfigMap{
			"bootstrap.servers": strings.Join(p.Brokers, ","),
			"socket.timeout.ms": 5000,
		})
		if newErr != nil {
			return fmt.Errorf("failed to get metadata (and recreate client failed): %w", err)
		}
		p.client = adminClient

		// Retry metadata fetch
		metadata, err = p.client.GetMetadata(&p.Topic, false, 5000)
		if err != nil {
			return fmt.Errorf("failed to get metadata after retry: %w", err)
		}
	}

	// Check if brokers are available
	if len(metadata.Brokers) == 0 {
		return fmt.Errorf("no brokers available in cluster")
	}

	// Verify topic exists and has partitions
	topicMetadata, exists := metadata.Topics[p.Topic]
	if !exists {
		return fmt.Errorf("topic %s does not exist", p.Topic)
	}

	if topicMetadata.Error.Code() != kafka.ErrNoError {
		return fmt.Errorf("topic %s has error: %v", p.Topic, topicMetadata.Error)
	}

	if len(topicMetadata.Partitions) == 0 {
		return fmt.Errorf("topic %s has no partitions", p.Topic)
	}

	// Check if each partition has a leader
	for _, partition := range topicMetadata.Partitions {
		if partition.Leader < 0 {
			return fmt.Errorf("no leader for topic %s partition %d", p.Topic, partition.ID)
		}
		if partition.Error.Code() != kafka.ErrNoError {
			return fmt.Errorf("partition %d has error: %v", partition.ID, partition.Error)
		}
	}

	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][Kafka][Read] Cluster healthy: %d brokers, topic %s with %d partitions",
			len(metadata.Brokers), p.Topic, len(topicMetadata.Partitions))
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
	producer *kafka.Producer
	consumer *kafka.Consumer
}

// NewWriteProbe creates a WriteProbe with persistent producer and consumer
func NewWriteProbe(brokers []string, topic string) (*WriteProbe, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(brokers, ","),
		"acks":              "1",
		"retries":           1,
		"socket.timeout.ms": 5000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Create a consumer for verification
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(brokers, ","),
		"group.id":          "prober-group-" + RandString(8),
		"auto.offset.reset": "latest",
		"socket.timeout.ms": 5000,
	})
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
	deliveryChan := make(chan kafka.Event, 1)
	err := p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Key:            []byte(testKey),
		Value:          []byte(testValue),
	}, deliveryChan)

	if err != nil {
		// Try to recreate producer once
		p.producer.Close()
		producer, newErr := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": strings.Join(p.Brokers, ","),
			"acks":              "1",
			"retries":           1,
			"socket.timeout.ms": 5000,
		})
		if newErr != nil {
			return fmt.Errorf("failed to produce message (and recreate producer failed): %w", err)
		}
		p.producer = producer

		// Retry producing
		err = p.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
			Key:            []byte(testKey),
			Value:          []byte(testValue),
		}, deliveryChan)
		if err != nil {
			return fmt.Errorf("failed to produce message after retry: %w", err)
		}
	}

	// Wait for delivery report with timeout
	select {
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("failed to deliver message: %w", m.TopicPartition.Error)
		}
		if os.Getenv("DEBUG") == "1" {
			log.Printf("[DEBUG][Kafka][Write] Message produced successfully to partition %d at offset %v",
				m.TopicPartition.Partition, m.TopicPartition.Offset)
		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for message delivery")
	}

	// Verify we can subscribe to the topic (checking cluster read path)
	err = p.consumer.Subscribe(p.Topic, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic for verification: %w", err)
	}
	// Unsubscribe immediately as we just want to verify connectivity
	_ = p.consumer.Unsubscribe()

	return nil
}

func (p *WriteProbe) MetadataString() string {
	return fmt.Sprintf("Brokers: %v , Topic: %s , Region: %s", p.Brokers, p.Topic, p.Region)
}
