package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type ClusterProbe struct {
	Region   string
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

type ShardError struct {
	Addr string
	Err  error
}

func (e ShardError) Error() string {
	return fmt.Sprintf("Redis cluster shard %s error: %v", e.Addr, e.Err)
}

func (p *ClusterProbe) Probe(ctx context.Context) error {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    p.Addrs,
		Password: p.Password,
	})
	defer client.Close()

	var firstErr *ShardError
	_ = client.ForEachShard(ctx, func(ctx context.Context, c *redis.Client) error {
		if _, err := c.Ping(ctx).Result(); err != nil && firstErr == nil {
			firstErr = &ShardError{Addr: c.Options().Addr, Err: err}
		}
		return nil
	})
	if firstErr != nil {
		return firstErr
	}
	return nil
}

func (p *ClusterProbe) MetadataString() string {
	return fmt.Sprintf("Nodes: %v | Region: %s", p.Addrs, p.Region)
}
