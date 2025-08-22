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

type Config struct {
	S3 struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []struct {
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
		} `yaml:"clusters"`
	} `yaml:"s3"`
	DefaultDuration DurationString `yaml:"defaultDuration"`
	// Add more per-type defaults as needed

	MySQL struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []struct {
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
		} `yaml:"clusters"`
	} `yaml:"mysql"`
	Kafka struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []struct {
			Name     string         `yaml:"name"`
			Brokers  []string       `yaml:"brokers"`
			Topic    string         `yaml:"topic"`
			Duration DurationString `yaml:"duration"`
		} `yaml:"clusters"`
	} `yaml:"kafka"`
	Redis struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []struct {
			Name     string         `yaml:"name"`
			Nodes    []string       `yaml:"nodes"`
			Password string         `yaml:"password"`
			Duration DurationString `yaml:"duration"`
			Tasks    struct {
				Read  bool `yaml:"read"`
				Write bool `yaml:"write"`
			} `yaml:"tasks"`
		} `yaml:"clusters"`
	} `yaml:"redis"`
	RedisCluster struct {
		DefaultDuration DurationString `yaml:"defaultDuration"`
		Clusters        []struct {
			Name     string         `yaml:"name"`
			Nodes    []string       `yaml:"nodes"`
			Password string         `yaml:"password"`
			Duration DurationString `yaml:"duration"`
			Tasks    struct {
				Read  bool `yaml:"read"`
				Write bool `yaml:"write"`
			} `yaml:"tasks"`
		} `yaml:"clusters"`
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
