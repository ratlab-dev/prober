package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type ReadProbe struct {
	Addr     string
	Password string
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     p.Addr,
		Password: p.Password,
		DB:       0,
	})
	defer rdb.Close()
	_, err := rdb.Ping(ctx).Result()
	return err
}

type WriteProbe struct {
	Addr     string
	Password string
}

func (p *WriteProbe) Probe(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     p.Addr,
		Password: p.Password,
		DB:       0,
	})
	defer rdb.Close()
	return rdb.Set(ctx, "probe_key", "ok", 0).Err()
}
