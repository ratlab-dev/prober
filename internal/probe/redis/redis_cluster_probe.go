package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type ClusterProbe struct {
	Addrs    []string
	Password string
}

// NewClusterProbe creates a ClusterProbe
func NewClusterProbe(addrs []string, password string) *ClusterProbe {
	return &ClusterProbe{
		Addrs:    addrs,
		Password: password,
	}
}

func (p *ClusterProbe) Probe(ctx context.Context) error {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    p.Addrs,
		Password: p.Password,
	})
	defer client.Close()

	var lastErr error
	err := client.ForEachShard(ctx, func(ctx context.Context, c *redis.Client) error {
		_, err := c.Ping(ctx).Result()
		if err != nil {
			lastErr = err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return lastErr
}
