package probe

import (
	"context"
	"log"
	"strings"
	"time"

	httpprobe "github.com/yourorg/prober/internal/probe/http"
	kafkaprobe "github.com/yourorg/prober/internal/probe/kafka"
	mysqlprobe "github.com/yourorg/prober/internal/probe/mysql"
	redisprobe "github.com/yourorg/prober/internal/probe/redis"
	"github.com/yourorg/prober/internal/probe/s3"
)

type statusMsg struct {
	TargetType string
	Cluster    string
	Host       string
	Status     string
	Err        error
}

func RunAll(ctx context.Context, cfg *Config) {
	statusCh := make(chan statusMsg, 100)

	RunS3(ctx, cfg, statusCh)
	RunMySQL(ctx, cfg, statusCh)
	RunKafka(ctx, cfg, statusCh)
	RunRedis(ctx, cfg, statusCh)
	RunRedisCluster(ctx, cfg, statusCh)
	RunHTTP(ctx, cfg, statusCh)

	printStatusUpdates(ctx, statusCh)
}

func launchProbeWithDuration(ctx context.Context, ms int, clusterName, host, targetType string, probe Prober, statusCh chan<- statusMsg, onSuccess, onFailure func()) {
	go func() {
		ticker := newTickerWithContext(ctx, ms)
		defer ticker.Stop()
		for range ticker.C {
			err := probe.Probe(ctx)
			status := "OK"
			if err != nil {
				status = "ERROR"
				if onFailure != nil {
					onFailure()
				}
			} else {
				if onSuccess != nil {
					onSuccess()
				}
			}
			statusCh <- statusMsg{
				TargetType: targetType,
				Cluster:    clusterName,
				Host:       host,
				Status:     status,
				Err:        err,
			}
		}
	}()
}

func RunS3(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	for _, cluster := range cfg.S3.Clusters {
		dur := cluster.Duration.ToDuration(
			cfg.S3.DefaultDuration.ToDuration(
				cfg.DefaultDuration.ToDuration(10 * time.Second),
			),
		)
		ms := int(dur.Milliseconds())
		if ms < 100 {
			ms = 100
		}
		objectKey := "probe-test-file"
		if cluster.Tasks.Write {
			probe := s3.NewWriteProbe(cluster.Endpoint,
				cluster.Region,
				cluster.AccessKey,
				cluster.SecretKey,
				cluster.Bucket,
				objectKey,
				cluster.UseSSL,
				cluster.Timeout.ToDuration(time.Second),
			)
			launchProbeWithDuration(ctx, ms, cluster.Name, cluster.Endpoint, "S3-WRITE", probe, statusCh,
				func() { IncProbeSuccess("s3", "write", cluster.Endpoint, cluster.Name) },
				func() { IncProbeFailure("s3", "write", cluster.Endpoint, cluster.Name) },
			)
		}
		if cluster.Tasks.Read {
			probe := s3.NewReadProbe(
				cluster.Endpoint,
				cluster.Region,
				cluster.AccessKey,
				cluster.SecretKey,
				cluster.Bucket,
				objectKey,
				cluster.UseSSL,
				cluster.Timeout.ToDuration(time.Second),
			)
			launchProbeWithDuration(ctx, ms, cluster.Name, cluster.Endpoint, "S3-READ", probe, statusCh,
				func() { IncProbeSuccess("s3", "read", cluster.Endpoint, cluster.Name) },
				func() { IncProbeFailure("s3", "read", "", cluster.Endpoint) },
			)
		}
	}
}

func RunMySQL(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	for _, cluster := range cfg.MySQL.Clusters {
		dur := cluster.Duration.ToDuration(
			cfg.MySQL.DefaultDuration.ToDuration(
				cfg.DefaultDuration.ToDuration(10 * time.Second),
			),
		)
		ms := int(dur.Milliseconds())
		if ms < 100 {
			ms = 100
		}
		if len(cluster.ReadHosts) > 0 && cluster.Tasks.Read {
			for _, host := range cluster.ReadHosts {
				probe, err := mysqlprobe.NewReadProbe(host, cluster.User, cluster.Password, cluster.Database, cluster.ReadQuery)
				if err != nil {
					log.Printf("culd not create mysql probe for cluster: %s, host: %s, err: %v", cluster.Name, host, err)
					continue
				}
				launchProbeWithDuration(ctx, ms, cluster.Name, host, "MySQL-READ", probe, statusCh,
					func() { IncProbeSuccess("mysql", "read", host, cluster.Name) },
					func() { IncProbeFailure("mysql", "read", host, cluster.Name) },
				)
			}
		}
		if len(cluster.WriteHosts) > 0 && cluster.Tasks.Write {
			for _, host := range cluster.WriteHosts {
				probe, err := mysqlprobe.NewWriteProbe(host, cluster.User, cluster.Password, cluster.Database, cluster.WriteQuery)
				if err != nil {
					log.Printf("culd not create mysql probe for cluster: %s, host: %s, err: %v", cluster.Name, host, err)
					continue
				}
				launchProbeWithDuration(ctx, ms, cluster.Name, host, "MySQL-WRITE", probe, statusCh,
					func() { IncProbeSuccess("mysql", "write", host, cluster.Name) },
					func() { IncProbeFailure("mysql", "write", host, cluster.Name) },
				)
			}
		}
	}
}

func RunKafka(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	for _, cluster := range cfg.Kafka.Clusters {
		dur := cluster.Duration.ToDuration(
			cfg.Kafka.DefaultDuration.ToDuration(
				cfg.DefaultDuration.ToDuration(10 * time.Second),
			),
		)
		ms := int(dur.Milliseconds())
		if ms < 100 {
			ms = 100
		}
		readProbe := &kafkaprobe.ReadProbe{
			Brokers: cluster.Brokers,
			Topic:   cluster.Topic,
		}
		launchProbeWithDuration(ctx, ms, cluster.Name, "", "Kafka-READ", readProbe, statusCh, nil, nil)

		writeProbe := &kafkaprobe.WriteProbe{
			Brokers: cluster.Brokers,
			Topic:   cluster.Topic,
		}
		launchProbeWithDuration(ctx, ms, cluster.Name, "", "Kafka-WRITE", writeProbe, statusCh, nil, nil)
	}
}

func RunRedis(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	for _, cluster := range cfg.Redis.Clusters {
		dur := cluster.Duration.ToDuration(
			cfg.Redis.DefaultDuration.ToDuration(
				cfg.Redis.DefaultDuration.ToDuration(
					cfg.DefaultDuration.ToDuration(10 * time.Second),
				),
			),
		)
		ms := int(dur.Milliseconds())
		if ms < 100 {
			ms = 100
		}
		if cluster.Tasks.Read {
			for _, node := range cluster.Nodes {
				probe := redisprobe.NewReadProbe(node, cluster.Password)
				launchProbeWithDuration(ctx, ms, cluster.Name, node, "Redis-READ", probe, statusCh,
					func() { IncProbeSuccess("redis", "read", node, cluster.Name) },
					func() { IncProbeFailure("redis", "read", node, cluster.Name) },
				)
			}
		}
		if cluster.Tasks.Write {
			for _, node := range cluster.Nodes {
				probe := redisprobe.NewWriteProbe(node, cluster.Password)
				launchProbeWithDuration(ctx, ms, cluster.Name, node, "Redis-WRITE", probe, statusCh,
					func() { IncProbeSuccess("redis", "write", node, cluster.Name) },
					func() { IncProbeFailure("redis", "write", node, cluster.Name) },
				)
			}
		}
	}
}

func RunRedisCluster(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	for _, cluster := range cfg.RedisCluster.Clusters {
		dur := cluster.Duration.ToDuration(
			cfg.RedisCluster.DefaultDuration.ToDuration(
				cfg.RedisCluster.DefaultDuration.ToDuration(
					cfg.DefaultDuration.ToDuration(10 * time.Second),
				),
			),
		)
		ms := int(dur.Milliseconds())
		if ms < 100 {
			ms = 100
		}
		probe := redisprobe.NewClusterProbe(cluster.Nodes, cluster.Password)
		launchProbeWithDuration(ctx, ms, cluster.Name, strings.Join(cluster.Nodes, ","), "RedisCluster-READWRITE", probe, statusCh,
			func() { IncProbeSuccess("redisCluster", "read", strings.Join(cluster.Nodes, ","), cluster.Name) },
			func() { IncProbeFailure("redisCluster", "read", strings.Join(cluster.Nodes, ","), cluster.Name) },
		)
	}
}

func RunHTTP(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	for _, cluster := range cfg.HTTP.Clusters {
		dur := cluster.Duration.ToDuration(
			cfg.HTTP.DefaultDuration.ToDuration(
				cfg.DefaultDuration.ToDuration(10 * time.Second),
			),
		)
		ms := int(dur.Milliseconds())
		if ms < 100 {
			ms = 100
		}
		probe := httpprobe.NewHTTPProbe(
			cluster.Endpoint,
			cluster.Method,
			cluster.Body,
			cluster.ProxyURL,
			cluster.Headers,
			cluster.UnacceptableStatusCodes,
			cluster.Timeout.ToDuration(2*time.Second),
			cluster.SkipTLSVerify,
		)
		launchProbeWithDuration(ctx, ms, cluster.Name, cluster.Endpoint, "HTTP", probe, statusCh,
			func() { IncProbeSuccess("http", "probe", cluster.Endpoint, cluster.Name) },
			func() { IncProbeFailure("http", "probe", cluster.Endpoint, cluster.Name) },
		)
	}
}

func printStatusUpdates(ctx context.Context, statusCh <-chan statusMsg) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Probing stopped.")
			return
		case msg := <-statusCh:
			if msg.Status == "OK" {
				if msg.Host != "" {
					log.Printf("[%s][%s][%s] OK\n", msg.TargetType, msg.Cluster, msg.Host)
				} else {
					log.Printf("[%s][%s] OK\n", msg.TargetType, msg.Cluster)
				}
			} else {
				if msg.Host != "" {
					log.Printf("[%s][%s][%s] ERROR: %v\n", msg.TargetType, msg.Cluster, msg.Host, msg.Err)
				} else {
					log.Printf("[%s][%s] ERROR: %v\n", msg.TargetType, msg.Cluster, msg.Err)
				}
			}
		}
	}
}

// newTickerWithContext returns a ticker that stops when ctx is done.
func newTickerWithContext(ctx context.Context, ms int) *tickerWithContext {
	t := &tickerWithContext{
		C:    make(chan struct{}),
		stop: make(chan struct{}),
	}
	go func() {
		tick := make(chan struct{}, 1)
		go func() {
			for {
				tick <- struct{}{}
				time.Sleep(time.Duration(ms) * time.Millisecond)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				close(t.C)
				return
			case <-t.stop:
				close(t.C)
				return
			case <-tick:
				t.C <- struct{}{}
			}
		}
	}()
	return t
}

type tickerWithContext struct {
	C    chan struct{}
	stop chan struct{}
}

func (t *tickerWithContext) Stop() {
	close(t.stop)
}

// No-op: all probe logic now uses Prober interface and structs
