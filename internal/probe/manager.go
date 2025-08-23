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
				log.Println(m.Status, m.Details)
			}
		}()
	}

	// --- Launch or update probes for each kind ---

	// S3
	for _, cluster := range cfg.S3.Clusters {
		singleCfg := &Config{}
		singleCfg.S3.DefaultDuration = cfg.S3.DefaultDuration
		singleCfg.S3.Clusters = []struct {
			Name      string         `yaml:"name"`
			Endpoint  string         `yaml:"endpoint"`
			Region    string         `yaml:"region"`
			AccessKey string         `yaml:"accessKey"`
			SecretKey string         `yaml:"secretKey"`
			Bucket    string         `yaml:"bucket"`
			UseSSL    bool           `yaml:"useSSL"`
			Duration  DurationString `yaml:"duration"`
			Timeout   DurationString `yaml:"timeout"`
			Tasks     struct {
				Read  bool `yaml:"read"`
				Write bool `yaml:"write"`
			} `yaml:"tasks"`
		}{cluster}
		startOrUpdateProbe("s3", cluster.Name, cluster, RunS3, singleCfg)
	}

	// MySQL
	for _, cluster := range cfg.MySQL.Clusters {
		singleCfg := &Config{}
		singleCfg.MySQL.DefaultDuration = cfg.MySQL.DefaultDuration
		singleCfg.MySQL.Clusters = []struct {
			Name       string         `yaml:"name"`
			ReadHosts  []string       `yaml:"read_hosts"`
			WriteHosts []string       `yaml:"write_hosts"`
			User       string         `yaml:"user"`
			Password   string         `yaml:"password"`
			Database   string         `yaml:"database"`
			Duration   DurationString `yaml:"duration"`
			ReadQuery  string         `yaml:"read_query"`
			WriteQuery string         `yaml:"write_query"`
			Tasks      struct {
				Read  bool `yaml:"read"`
				Write bool `yaml:"write"`
			} `yaml:"tasks"`
		}{cluster}
		startOrUpdateProbe("mysql", cluster.Name, cluster, RunMySQL, singleCfg)
	}

	// Kafka
	for _, cluster := range cfg.Kafka.Clusters {
		singleCfg := &Config{}
		singleCfg.Kafka.DefaultDuration = cfg.Kafka.DefaultDuration
		singleCfg.Kafka.Clusters = []struct {
			Name     string         `yaml:"name"`
			Brokers  []string       `yaml:"brokers"`
			Topic    string         `yaml:"topic"`
			Duration DurationString `yaml:"duration"`
		}{cluster}
		startOrUpdateProbe("kafka", cluster.Name, cluster, RunKafka, singleCfg)
	}

	// Redis
	for _, cluster := range cfg.Redis.Clusters {
		singleCfg := &Config{}
		singleCfg.Redis.DefaultDuration = cfg.Redis.DefaultDuration
		singleCfg.Redis.Clusters = []struct {
			Name     string         `yaml:"name"`
			Nodes    []string       `yaml:"nodes"`
			Password string         `yaml:"password"`
			Duration DurationString `yaml:"duration"`
			Tasks    struct {
				Read  bool `yaml:"read"`
				Write bool `yaml:"write"`
			} `yaml:"tasks"`
		}{cluster}
		startOrUpdateProbe("redis", cluster.Name, cluster, RunRedis, singleCfg)
	}

	// RedisCluster
	for _, cluster := range cfg.RedisCluster.Clusters {
		singleCfg := &Config{}
		singleCfg.RedisCluster.DefaultDuration = cfg.RedisCluster.DefaultDuration
		singleCfg.RedisCluster.Clusters = []struct {
			Name     string         `yaml:"name"`
			Nodes    []string       `yaml:"nodes"`
			Password string         `yaml:"password"`
			Duration DurationString `yaml:"duration"`
		}{cluster}
		startOrUpdateProbe("redisCluster", cluster.Name, cluster, RunRedisCluster, singleCfg)
	}

	// HTTP
	for _, cluster := range cfg.HTTP.Clusters {
		singleCfg := &Config{}
		singleCfg.HTTP.DefaultDuration = cfg.HTTP.DefaultDuration
		singleCfg.HTTP.Clusters = []struct {
			Name                    string            `yaml:"name"`
			Endpoint                string            `yaml:"endpoint"`
			Method                  string            `yaml:"method"`
			Body                    string            `yaml:"body"`
			Headers                 map[string]string `yaml:"headers"`
			ProxyURL                string            `yaml:"proxyURL"`
			UnacceptableStatusCodes []int             `yaml:"unacceptableStatusCodes"`
			Timeout                 DurationString    `yaml:"timeout"`
			Duration                DurationString    `yaml:"duration"`
			SkipTLSVerify           bool              `yaml:"skipTLSVerify"`
		}{cluster}
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
