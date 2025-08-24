package probe

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	httpprobe "github.com/yourorg/prober/internal/probe/http"
	kafkaprobe "github.com/yourorg/prober/internal/probe/kafka"
	mysqlprobe "github.com/yourorg/prober/internal/probe/mysql"
	redisprobe "github.com/yourorg/prober/internal/probe/redis"
	"github.com/yourorg/prober/internal/probe/s3"
	tcpprobe "github.com/yourorg/prober/internal/probe/tcp"
)

// ...existing code...

func RunTCP(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	sourceRegion := os.Getenv("SOURCE_REGION")
	if sourceRegion == "" {
		sourceRegion = "local"
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "local"
	}
	for _, cluster := range cfg.TCP.Clusters {
		if cluster.Region == "" {
			log.Printf("ERROR: region missing for TCP cluster %s (source region: %s)", cluster.Name, sourceRegion)
		}
		dur := cluster.Duration.ToDuration(
			cfg.TCP.DefaultDuration.ToDuration(
				cfg.DefaultDuration.ToDuration(10 * time.Second),
			),
		)
		ms := int(dur.Milliseconds())
		if ms < 100 {
			ms = 100
		}
		probe := tcpprobe.NewTCPProbe(
			cluster.Addresses,
			cluster.Timeout.ToDuration(2*time.Second),
		)
		probe.Region = cluster.Region
		launchProbeWithDuration(ctx, ms, cluster.Name, strings.Join(cluster.Addresses, ","), "TCP", probe, statusCh,
			func() {
				IncProbeSuccess("tcp", "probe", cluster.Name, sourceRegion, cluster.Region)
			},
			func() {
				IncProbeFailure("tcp", "probe", cluster.Name, sourceRegion, cluster.Region)
			},
		)
	}
}

type statusMsg struct {
	TargetType string
	Cluster    string
	Host       string
	Status     string
	Err        error
	Details    string
}

func launchProbeWithDuration(ctx context.Context, ms int, clusterName, host, targetType string, probe Prober, statusCh chan<- statusMsg, onSuccess, onFailure func()) {
	go func() {
		ticker := newTickerWithContext(ctx, ms)
		defer ticker.Stop()
		for range ticker.C {
			err := probe.Probe(ctx)
			status := "OK "
			if err != nil {
				status = "ERR"
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
				Details:    probe.MetadataString(),
			}
		}
	}()
}

func RunS3(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	sourceRegion := os.Getenv("SOURCE_REGION")
	if sourceRegion == "" {
		sourceRegion = "local"
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "local"
	}
	for _, cluster := range cfg.S3.Clusters {
		if cluster.Region == "" {
			log.Printf("ERROR: region missing for S3 cluster %s (source region: %s)", cluster.Name, sourceRegion)
		}
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
			probe.Region = cluster.Region
			launchProbeWithDuration(ctx, ms, cluster.Name, cluster.Endpoint, "S3_WRITE", probe, statusCh,
				func() {
					IncProbeSuccess("s3", "write", cluster.Name, sourceRegion, cluster.Region)
				},
				func() {
					IncProbeFailure("s3", "write", cluster.Name, sourceRegion, cluster.Region)
				},
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
			probe.Region = cluster.Region
			launchProbeWithDuration(ctx, ms, cluster.Name, cluster.Endpoint, "S3_READ", probe, statusCh,
				func() {
					IncProbeSuccess("s3", "read", cluster.Name, sourceRegion, cluster.Region)
				},
				func() {
					IncProbeFailure("s3", "read", cluster.Name, sourceRegion, cluster.Region)
				},
			)
		}
	}
}

func RunMySQL(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	sourceRegion := os.Getenv("SOURCE_REGION")
	if sourceRegion == "" {
		sourceRegion = "local"
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "local"
	}
	for _, cluster := range cfg.MySQL.Clusters {
		if cluster.Region == "" {
			log.Printf("ERROR: region missing for MySQL cluster %s (source region: %s)", cluster.Name, sourceRegion)
		}
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
				probe.Region = cluster.Region
				launchProbeWithDuration(ctx, ms, cluster.Name, host, "MYSQL_READ", probe, statusCh,
					func() {
						IncProbeSuccess("mysql", "read", cluster.Name, sourceRegion, cluster.Region)
					},
					func() {
						IncProbeFailure("mysql", "read", cluster.Name, sourceRegion, cluster.Region)
					},
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
				probe.Region = cluster.Region
				launchProbeWithDuration(ctx, ms, cluster.Name, host, "MYSQL_WRITE", probe, statusCh,
					func() {
						IncProbeSuccess("mysql", "write", cluster.Name, sourceRegion, cluster.Region)
					},
					func() {
						IncProbeFailure("mysql", "write", cluster.Name, sourceRegion, cluster.Region)
					},
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
			Region:  cluster.Region,
		}
		launchProbeWithDuration(ctx, ms, cluster.Name, "", "KAFKA_READ", readProbe, statusCh, nil, nil)

		writeProbe := &kafkaprobe.WriteProbe{
			Brokers: cluster.Brokers,
			Topic:   cluster.Topic,
			Region:  cluster.Region,
		}
		launchProbeWithDuration(ctx, ms, cluster.Name, "", "KAFKA_WRITE", writeProbe, statusCh, nil, nil)
	}
}

func RunRedis(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	sourceRegion := os.Getenv("SOURCE_REGION")
	if sourceRegion == "" {
		sourceRegion = "local"
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "local"
	}
	for _, cluster := range cfg.Redis.Clusters {
		if cluster.Region == "" {
			log.Printf("ERROR: region missing for Redis cluster %s (source region: %s)", cluster.Name, sourceRegion)
		}
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
				probe.Region = cluster.Region
				launchProbeWithDuration(ctx, ms, cluster.Name, node, "REDIS_READ", probe, statusCh,
					func() {
						IncProbeSuccess("redis", "read", cluster.Name, sourceRegion, cluster.Region)
					},
					func() {
						IncProbeFailure("redis", "read", cluster.Name, sourceRegion, cluster.Region)
					},
				)
			}
		}
		if cluster.Tasks.Write {
			for _, node := range cluster.Nodes {
				probe := redisprobe.NewWriteProbe(node, cluster.Password)
				probe.Region = cluster.Region
				launchProbeWithDuration(ctx, ms, cluster.Name, node, "REDIS_WRITE", probe, statusCh,
					func() {
						IncProbeSuccess("redis", "write", cluster.Name, sourceRegion, cluster.Region)
					},
					func() {
						IncProbeFailure("redis", "write", cluster.Name, sourceRegion, cluster.Region)
					},
				)
			}
		}
	}
}

func RunRedisCluster(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	sourceRegion := os.Getenv("SOURCE_REGION")
	if sourceRegion == "" {
		sourceRegion = "local"
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "local"
	}
	for _, cluster := range cfg.RedisCluster.Clusters {
		if cluster.Region == "" {
			log.Printf("ERROR: region missing for RedisCluster cluster %s (source region: %s)", cluster.Name, sourceRegion)
		}
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
		probe.Region = cluster.Region
		launchProbeWithDuration(ctx, ms, cluster.Name, strings.Join(cluster.Nodes, ","), "REDISCLUSTER_READWRITE", probe, statusCh,
			func() {
				IncProbeSuccess("redisCluster", "read", cluster.Name, sourceRegion, cluster.Region)
			},
			func() {
				IncProbeFailure("redisCluster", "read", cluster.Name, sourceRegion, cluster.Region)
			},
		)
	}
}

func RunHTTP(ctx context.Context, cfg *Config, statusCh chan<- statusMsg) {
	sourceRegion := os.Getenv("SOURCE_REGION")
	if sourceRegion == "" {
		sourceRegion = "local"
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "local"
	}
	for _, cluster := range cfg.HTTP.Clusters {
		if cluster.Region == "" {
			log.Printf("ERROR: region missing for HTTP cluster %s (source region: %s)", cluster.Name, sourceRegion)
		}
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
		probe.Region = cluster.Region
		launchProbeWithDuration(ctx, ms, cluster.Name, cluster.Endpoint, "HTTP", probe, statusCh,
			func() {
				IncProbeSuccess("http", "probe", cluster.Name, sourceRegion, cluster.Region)
			},
			func() {
				IncProbeFailure("http", "probe", cluster.Name, sourceRegion, cluster.Region)
			},
		)
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
