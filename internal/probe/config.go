package probe

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type DurationString string

func (d DurationString) ToDuration(defaultDuration time.Duration) time.Duration {
	if d == "" {
		return defaultDuration
	}
	dur, err := time.ParseDuration(string(d))
	if err != nil {
		return defaultDuration
	}
	return dur
}

type TCPCluster struct {
	Name      string         `yaml:"name"`
	Addresses []string       `yaml:"addresses"`
	Duration  DurationString `yaml:"duration"`
	Timeout   DurationString `yaml:"timeout"`
	Region    string         `yaml:"region"`
}
type S3Tasks struct {
	Read  bool `yaml:"read"`
	Write bool `yaml:"write"`
}
type S3Cluster struct {
	Name      string         `yaml:"name"`
	Endpoint  string         `yaml:"endpoint"`
	Region    string         `yaml:"region"`
	AccessKey string         `yaml:"accessKey"`
	SecretKey string         `yaml:"secretKey"`
	Bucket    string         `yaml:"bucket"`
	UseSSL    bool           `yaml:"useSSL"`
	Duration  DurationString `yaml:"duration"`
	Timeout   DurationString `yaml:"timeout"`
	Tasks     S3Tasks        `yaml:"tasks"`
}
type MySQLTasks struct {
	Read  bool `yaml:"read"`
	Write bool `yaml:"write"`
}
type MySQLCluster struct {
	Name       string         `yaml:"name"`
	ReadHosts  []string       `yaml:"read_hosts"`
	WriteHosts []string       `yaml:"write_hosts"`
	User       string         `yaml:"user"`
	Password   string         `yaml:"password"`
	Database   string         `yaml:"database"`
	Duration   DurationString `yaml:"duration"`
	ReadQuery  string         `yaml:"read_query"`
	WriteQuery string         `yaml:"write_query"`
	Region     string         `yaml:"region"`
	Tasks      MySQLTasks     `yaml:"tasks"`
}
type KafkaCluster struct {
	Name     string         `yaml:"name"`
	Brokers  []string       `yaml:"brokers"`
	Topic    string         `yaml:"topic"`
	Duration DurationString `yaml:"duration"`
	Region   string         `yaml:"region"`
}
type RedisTasks struct {
	Read  bool `yaml:"read"`
	Write bool `yaml:"write"`
}
type RedisCluster struct {
	Name     string         `yaml:"name"`
	Nodes    []string       `yaml:"nodes"`
	Password string         `yaml:"password"`
	Duration DurationString `yaml:"duration"`
	Region   string         `yaml:"region"`
	Tasks    RedisTasks     `yaml:"tasks"`
}
type HTTPCluster struct {
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
	Region                  string            `yaml:"region"`
}
type RedisClusterCluster struct {
	Name     string         `yaml:"name"`
	Nodes    []string       `yaml:"nodes"`
	Password string         `yaml:"password"`
	Duration DurationString `yaml:"duration"`
	Region   string         `yaml:"region"`
}

// Config struct
type Config struct {
	TCP struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []TCPCluster   `yaml:"clusters"`
	} `yaml:"tcp"`
	S3 struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []S3Cluster    `yaml:"clusters"`
	} `yaml:"s3"`
	DefaultDuration DurationString `yaml:"defaultDuration"`
	MySQL           struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []MySQLCluster `yaml:"clusters"`
	} `yaml:"mysql"`
	Kafka struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []KafkaCluster `yaml:"clusters"`
	} `yaml:"kafka"`
	Redis struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []RedisCluster `yaml:"clusters"`
	} `yaml:"redis"`
	HTTP struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []HTTPCluster  `yaml:"clusters"`
	} `yaml:"http"`
	RedisCluster struct {
		DefaultDuration DurationString        `yaml:"defaultDuration"`
		Clusters        []RedisClusterCluster `yaml:"clusters"`
	} `yaml:"redisCluster"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
