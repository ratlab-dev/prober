package redis

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
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
	Region   string
	Addr     string
	Password string
	client   *redis.Client
}

// NewReadProbe creates a ReadProbe with a persistent client
func NewReadProbe(addr, password string) *ReadProbe {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	return &ReadProbe{
		Addr:     addr,
		Password: password,
		client:   client,
	}
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	_, err := p.client.Ping(ctx).Result()
	if err != nil {
		// Try to reconnect once
		p.client.Close()
		p.client = redis.NewClient(&redis.Options{
			Addr:     p.Addr,
			Password: p.Password,
			DB:       0,
		})
		_, err = p.client.Ping(ctx).Result()
	}
	return err
}

func (p *ReadProbe) MetadataString() string {
	return fmt.Sprintf("Node: %s , Region: %s", p.Addr, p.Region)
}

type WriteProbe struct {
	Region   string
	Addr     string
	Password string
	client   *redis.Client
}

// NewWriteProbe creates a WriteProbe with a persistent client
func NewWriteProbe(addr, password string) *WriteProbe {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	return &WriteProbe{
		Addr:     addr,
		Password: password,
		client:   client,
	}
}

func (p *WriteProbe) Probe(ctx context.Context) error {
	key := "probe_key_" + RandString(12)
	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][Redis][%s] Writing key: %s", p.Addr, key)
	}
	err := p.client.Set(ctx, key, "ok", 30*time.Second).Err()
	if err != nil {
		// Try to reconnect once
		p.client.Close()
		p.client = redis.NewClient(&redis.Options{
			Addr:     p.Addr,
			Password: p.Password,
			DB:       0,
		})
		err = p.client.Set(ctx, key, "ok", 30*time.Second).Err()
	}
	return err
}

func (p *WriteProbe) MetadataString() string {
	return fmt.Sprintf("Node: %s , Region: %s", p.Addr, p.Region)
}
