package redis

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

type ClusterReadProbe struct {
	Addrs    []string
	Password string
}

func (p *ClusterReadProbe) Probe(ctx context.Context) error {
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    p.Addrs,
		Password: p.Password,
	})

	defer rdb.Close()
	log.Println("Addrs", rdb.Options().Addrs)
	_, err := rdb.Ping(ctx).Result()
	return err
}

type ClusterWriteProbe struct {
	Addrs    []string
	Password string
}

func (p *ClusterWriteProbe) Probe(ctx context.Context) error {
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    p.Addrs,
		Password: p.Password,
	})
	defer rdb.Close()
	return rdb.Set(ctx, "probe_key", "ok", 0).Err()
}
