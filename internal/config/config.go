package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Env       string    `yaml:"env"`
	Server    Server    `yaml:"server"`
	Metrics   Metrics   `yaml:"metrics"`
	Defaults  Defaults  `yaml:"defaults"`
	Mirroring Mirroring `yaml:"mirroring"`
	Routes    []Route   `yaml:"routes"`
}

type Server struct {
	Address           string        `yaml:"address"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
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

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config %s:\n%w", path, err)
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8080"
	}

	if cfg.Server.ReadHeaderTimeout == 0 {
		cfg.Server.ReadHeaderTimeout = 5 * time.Second
	}

	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}

	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 5 * time.Second
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

func validate(cfg *Config) error {
	var errs []error

	if cfg.Env == "" {
		errs = append(errs, fmt.Errorf("env must not be empty"))
	}

	if cfg.Env != "local" && cfg.Env != "dev" && cfg.Env != "prod" {
		errs = append(errs, fmt.Errorf("env must be one of 'local', 'dev', or 'prod'"))
	}

	_, portSrv, errSrv := net.SplitHostPort(cfg.Server.Address)
	if errSrv != nil {
		errs = append(errs, fmt.Errorf("invalid server address format: %w", errSrv))
	}

	_, portMtr, errMtr := net.SplitHostPort(cfg.Metrics.Address)
	if errMtr != nil {
		errs = append(errs, fmt.Errorf("invalid metrics address format: %w", errMtr))
	}

	if errSrv == nil && errMtr == nil && portSrv == portMtr {
		errs = append(errs, errors.New("server and metrics must use different ports"))
	}

	if cfg.Server.ReadHeaderTimeout < 0 {
		errs = append(errs, fmt.Errorf("server.read_header_timeout must be positive"))
	}

	if cfg.Server.IdleTimeout < 0 {
		errs = append(errs, fmt.Errorf("server.idle_timeout must be positive"))
	}

	if cfg.Server.ShutdownTimeout < 0 {
		errs = append(errs, fmt.Errorf("server.shutdown_timeout must be positive"))
	}

	if cfg.Defaults.Timeout < 0 {
		errs = append(errs, fmt.Errorf("defaults.timeout must be positive"))
	}

	if cfg.Mirroring.QueueSize < 0 {
		errs = append(errs, fmt.Errorf("mirroring.queue_size must be positive"))
	}

	if cfg.Mirroring.Workers < 0 {
		errs = append(errs, fmt.Errorf("mirroring.workers must be positive"))
	}

	if cfg.Mirroring.Timeout < 0 {
		errs = append(errs, fmt.Errorf("mirroring.timeout must be positive"))
	}

	if cfg.Mirroring.MaxBodyBytes < 0 {
		errs = append(errs, fmt.Errorf("mirroring.max_body_bytes must be positive"))
	}

	if len(cfg.Routes) == 0 {
		errs = append(errs, fmt.Errorf("routes must not be empty"))
	}

	seenNames := make(map[string]struct{})
	for i, route := range cfg.Routes {
		for _, err := range validateRoute(route) {
			errs = append(errs, fmt.Errorf("route %d (%s): %w", i, route.Name, err))
		}

		if _, exists := seenNames[route.Name]; exists {
			errs = append(errs, fmt.Errorf("route %d (%s): duplicate name", i, route.Name))
		}
		seenNames[route.Name] = struct{}{}
	}

	return errors.Join(errs...)
}

func validateRoute(route Route) []error {
	var errs []error
	if route.Name == "" {
		errs = append(errs, fmt.Errorf("name must not be empty"))
	}

	set := 0
	for _, s := range []string{route.Exact, route.Prefix, route.Regex} {
		if s != "" {
			set++
		}
	}
	if set != 1 {
		errs = append(errs, errors.New("exactly one of exact, prefix, regex must be set"))
	}

	if route.Exact != "" && route.Exact[0] != '/' {
		errs = append(errs, fmt.Errorf("exact must start with '/'"))
	}

	if route.Prefix != "" && route.Prefix[0] != '/' {
		errs = append(errs, fmt.Errorf("prefix must start with '/'"))
	}

	if route.Regex != "" {
		if _, err := regexp.Compile(route.Regex); err != nil {
			errs = append(errs, fmt.Errorf("regex is invalid: %w", err))
		}
	}

	if route.Backend == "" {
		errs = append(errs, fmt.Errorf("backend must not be empty"))
	}

	if route.Backend != "" {
		err := validateURL(route.Backend)
		if err != nil {
			errs = append(errs, fmt.Errorf("backend is invalid: %w", err))
		}
	}

	if route.Fallback != "" {
		err := validateURL(route.Fallback)
		if err != nil {
			errs = append(errs, fmt.Errorf("fallback is invalid: %w", err))
		}
	}

	if route.Timeout < 0 {
		errs = append(errs, fmt.Errorf("timeout must be positive"))
	}

	if route.Limit != nil {
		if route.Limit.RPS <= 0 {
			errs = append(errs, fmt.Errorf("limit.rps must be positive"))
		}
		if route.Limit.Burst < 0 {
			errs = append(errs, fmt.Errorf("limit.burst must be positive"))
		}
	}

	if route.Mirror != nil {
		if route.Mirror.Backend == "" {
			errs = append(errs, fmt.Errorf("mirror.backend must not be empty"))
		}

		if route.Mirror.Backend != "" {
			err := validateURL(route.Mirror.Backend)
			if err != nil {
				errs = append(errs, fmt.Errorf("mirror.backend is invalid: %w", err))
			}
		}

		if route.Mirror.Percent < 0 || route.Mirror.Percent > 100 {
			errs = append(errs, fmt.Errorf("mirror.percent must be between 0 and 100"))
		}
	}

	return errs
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("host must not be empty")
	}
	return nil
}
