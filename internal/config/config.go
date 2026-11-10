package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Server    Server    `yaml:"server"`
	Metrics   Metrics   `yaml:"metrics"`
	Defaults  Defaults  `yaml:"defaults"`
	Mirroring Mirroring `yaml:"mirroring"`
	Routes    []Route   `yaml:"routes"`
}

type Server struct {
	Address string `yaml:"address"`
}

type Metrics struct {
	Address string `yaml:"address"`
}

type Defaults struct {
	Timeout time.Duration `yaml:"timeout"`
}

type Mirroring struct {
	QueueSize    int           `yaml:"queue_size"`
	Workers      int           `yaml:"workers"`
	Timeout      time.Duration `yaml:"timeout"`
	MaxBodyBytes int64         `yaml:"max_body_bytes"`
}

type Route struct {
	Name     string            `yaml:"name"`
	Exact    string            `yaml:"exact"`
	Prefix   string            `yaml:"prefix"`
	Regex    string            `yaml:"regex"`
	Backend  string            `yaml:"backend"`
	Headers  map[string]string `yaml:"headers"`
	Timeout  time.Duration     `yaml:"timeout"`
	Fallback string            `yaml:"fallback"`
	Mirror   *Mirror           `yaml:"mirror"`
	Limit    *Limit            `yaml:"limit"`
}

type Mirror struct {
	Backend string `yaml:"backend"`
	Percent int    `yaml:"percent"`
}

type Limit struct {
	RPS   int `yaml:"rps"`
	Burst int `yaml:"burst"`
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}

	return cfg
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config %s: %w", path, err)
	}

	applyDefaults(&cfg)

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8080"
	}

	if cfg.Metrics.Address == "" {
		cfg.Metrics.Address = ":9090"
	}

	if cfg.Mirroring.QueueSize == 0 {
		cfg.Mirroring.QueueSize = 1000
	}

	if cfg.Mirroring.Workers == 0 {
		cfg.Mirroring.Workers = 5
	}

	if cfg.Mirroring.Timeout == 0 {
		cfg.Mirroring.Timeout = 5 * time.Second
	}

	if cfg.Mirroring.MaxBodyBytes == 0 {
		cfg.Mirroring.MaxBodyBytes = 1024 * 1024
	}

	if cfg.Defaults.Timeout == 0 {
		cfg.Defaults.Timeout = 5 * time.Second
	}

	for i := range cfg.Routes {
		if cfg.Routes[i].Timeout == 0 {
			cfg.Routes[i].Timeout = cfg.Defaults.Timeout
		}

		if cfg.Routes[i].Limit != nil && cfg.Routes[i].Limit.Burst == 0 {
			cfg.Routes[i].Limit.Burst = cfg.Routes[i].Limit.RPS
		}
	}
}
