package redis

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type ClusterReadProbe struct {
	Addrs    []string
	Password string
	client   *redis.ClusterClient
}

// NewClusterReadProbe creates a ClusterReadProbe with a persistent client
func NewClusterReadProbe(addrs []string, password string) *ClusterReadProbe {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: password,
	})
	return &ClusterReadProbe{
		Addrs:    addrs,
		Password: password,
		client:   client,
	}
}

func (p *ClusterReadProbe) Probe(ctx context.Context) error {
	_, err := p.client.Ping(ctx).Result()
	if err != nil {
		// Try to reconnect once
		p.client.Close()
		p.client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    p.Addrs,
			Password: p.Password,
		})
		_, err = p.client.Ping(ctx).Result()
	}
	return err
}

type ClusterWriteProbe struct {
	Addrs    []string
	Password string
	client   *redis.ClusterClient
}

// NewClusterWriteProbe creates a ClusterWriteProbe with a persistent client
func NewClusterWriteProbe(addrs []string, password string) *ClusterWriteProbe {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: password,
	})
	return &ClusterWriteProbe{
		Addrs:    addrs,
		Password: password,
		client:   client,
	}
}

func (p *ClusterWriteProbe) Probe(ctx context.Context) error {
	key := "probe_key_" + RandString(12)
	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][RedisCluster][%v] Writing key: %s", p.Addrs, key)
	}
	err := p.client.Set(ctx, key, "ok", 30*time.Second).Err()
	if err != nil {
		// Try to reconnect once
		p.client.Close()
		p.client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    p.Addrs,
			Password: p.Password,
		})
		err = p.client.Set(ctx, key, "ok", 30*time.Second).Err()
	}
	return err
}
