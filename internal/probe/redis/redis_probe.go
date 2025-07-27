package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type ReadProbe struct {
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

type WriteProbe struct {
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
	err := p.client.Set(ctx, "probe_key", "ok", 0).Err()
	if err != nil {
		// Try to reconnect once
		p.client.Close()
		p.client = redis.NewClient(&redis.Options{
			Addr:     p.Addr,
			Password: p.Password,
			DB:       0,
		})
		err = p.client.Set(ctx, "probe_key", "ok", 0).Err()
	}
	return err
}
