package probe

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"log"
	"sync"
)

type probeKey struct {
	Kind string
	Name string
}

type ProbeManager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	probes  map[probeKey]context.CancelFunc
	configs map[probeKey][32]byte // hash of config for change detection
}

func NewProbeManager(ctx context.Context) *ProbeManager {
	c, cancel := context.WithCancel(ctx)
	return &ProbeManager{
		ctx:     c,
		cancel:  cancel,
		probes:  make(map[probeKey]context.CancelFunc),
		configs: make(map[probeKey][32]byte),
	}
}

func (pm *ProbeManager) Stop() {
	pm.cancel()
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, cancel := range pm.probes {
		cancel()
	}
	pm.probes = make(map[probeKey]context.CancelFunc)
	pm.configs = make(map[probeKey][32]byte)
}

// LaunchOrUpdateProbes launches or updates probes for all clusters in the config
func (pm *ProbeManager) LaunchOrUpdateProbes(cfg *Config) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Track which clusters are still present after reload
	activeClusters := make(map[probeKey]struct{})

	// Helper function to start or restart a probe for a cluster
	startOrUpdateProbe := func(kind, name string, config interface{}, runner func(context.Context, *Config, chan<- statusMsg), singleCfg *Config) {
		key := probeKey{Kind: kind, Name: name}
		configBytes, _ := json.Marshal(config)
		configHash := sha256.Sum256(configBytes)
		activeClusters[key] = struct{}{}

		// If config is unchanged, do nothing
		if prevHash, ok := pm.configs[key]; ok && prevHash == configHash {
			return
		}

		// If probe is already running, stop it
		if cancel, ok := pm.probes[key]; ok {
			log.Printf("[ProbeManager] Restarting probe for kind=%s, cluster=%s due to config change", kind, name)
			cancel()
		} else {
			log.Printf("[ProbeManager] Starting probe for kind=%s, cluster=%s", kind, name)
		}

		// Start new probe goroutine
		ctx, cancel := context.WithCancel(pm.ctx)
		pm.probes[key] = cancel
		pm.configs[key] = configHash

		go func() {
			// Each probe gets its own status channel
			statusCh := make(chan statusMsg, 10)
			go runner(ctx, singleCfg, statusCh)
			for m := range statusCh {
				log.Printf("status: %v | target_type: %v | cluster: %v | details: %v | Error: %v", m.Status, m.TargetType, m.Cluster, m.Details, m.Err)
			}
		}()
	}

	// --- Launch or update probes for each kind ---

	// TCP
	for _, cluster := range cfg.TCP.Clusters {
		singleCfg := &Config{}
		singleCfg.TCP.DefaultDuration = cfg.TCP.DefaultDuration
		singleCfg.TCP.Clusters = []TCPCluster{cluster}
		startOrUpdateProbe("tcp", cluster.Name, cluster, RunTCP, singleCfg)
	}

	// S3
	for _, cluster := range cfg.S3.Clusters {
		singleCfg := &Config{}
		singleCfg.S3.DefaultDuration = cfg.S3.DefaultDuration
		singleCfg.S3.Clusters = []S3Cluster{cluster}
		startOrUpdateProbe("s3", cluster.Name, cluster, RunS3, singleCfg)
	}

	// MySQL
	for _, cluster := range cfg.MySQL.Clusters {
		singleCfg := &Config{}
		singleCfg.MySQL.DefaultDuration = cfg.MySQL.DefaultDuration
		singleCfg.MySQL.Clusters = []MySQLCluster{cluster}
		startOrUpdateProbe("mysql", cluster.Name, cluster, RunMySQL, singleCfg)
	}

	// Kafka
	for _, cluster := range cfg.Kafka.Clusters {
		singleCfg := &Config{}
		singleCfg.Kafka.DefaultDuration = cfg.Kafka.DefaultDuration
		singleCfg.Kafka.Clusters = []KafkaCluster{cluster}
		startOrUpdateProbe("kafka", cluster.Name, cluster, RunKafka, singleCfg)
	}

	// Redis
	for _, cluster := range cfg.Redis.Clusters {
		singleCfg := &Config{}
		singleCfg.Redis.DefaultDuration = cfg.Redis.DefaultDuration
		singleCfg.Redis.Clusters = []RedisCluster{cluster}
		startOrUpdateProbe("redis", cluster.Name, cluster, RunRedis, singleCfg)
	}

	// RedisCluster
	for _, cluster := range cfg.RedisCluster.Clusters {
		singleCfg := &Config{}
		singleCfg.RedisCluster.DefaultDuration = cfg.RedisCluster.DefaultDuration
		singleCfg.RedisCluster.Clusters = []RedisClusterCluster{cluster}
		startOrUpdateProbe("redisCluster", cluster.Name, cluster, RunRedisCluster, singleCfg)
	}

	// HTTP
	for _, cluster := range cfg.HTTP.Clusters {
		singleCfg := &Config{}
		singleCfg.HTTP.DefaultDuration = cfg.HTTP.DefaultDuration
		singleCfg.HTTP.Clusters = []HTTPCluster{cluster}
		startOrUpdateProbe("http", cluster.Name, cluster, RunHTTP, singleCfg)
	}

	// --- Remove probes for clusters that no longer exist ---
	for key := range pm.probes {
		if _, stillActive := activeClusters[key]; !stillActive {
			pm.probes[key]()
			delete(pm.probes, key)
			delete(pm.configs, key)
		}
	}
}
