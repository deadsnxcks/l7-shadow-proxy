package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() *Config {
	return &Config{
		Server:    Server{Address: ":8080"},
		Metrics:   Metrics{Address: ":9090"},
		Defaults:  Defaults{Timeout: 5 * time.Second},
		Mirroring: Mirroring{QueueSize: 1000, Workers: 5, Timeout: time.Second, MaxBodyBytes: 1 << 20},
		Routes: []Route{
			{Name: "api", Prefix: "/api/", Backend: "http://api:9001", Timeout: 2 * time.Second},
			{Name: "api-v2", Prefix: "/api-v2/", Backend: "http://api-v2:9001", Timeout: 2 * time.Second},
		},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{
			name:    "valid",
			mutate:  func(c *Config) {},
			wantErr: "",
		},
		{
			name:    "backend without scheme",
			mutate:  func(c *Config) { c.Routes[0].Backend = "api:9001" },
			wantErr: "scheme",
		},
		{
			name: "mirror percent over 100",
			mutate: func(c *Config) {
				c.Routes[0].Mirror = &Mirror{Backend: "http://v2:9002", Percent: 150}
			},
			wantErr: "mirror.percent",
		},
		{
			name: "empty name",
			mutate: func(c *Config) {
				c.Routes[0].Name = ""
			},
			wantErr: "name must not be empty",
		},
		{
			name: "duplicate route",
			mutate: func(c *Config) {
				c.Routes[1].Name = "api"
			},
			wantErr: "duplicate",
		},
		{
			name: "no set prefix",
			mutate: func(c *Config) {
				c.Routes[0].Prefix = ""
			},
			wantErr: "exactly one of",
		},
		{
			name: "set more one prefix/exact/regex",
			mutate: func(c *Config) {
				c.Routes[0].Exact = "/api"
			},
			wantErr: "exactly one of",
		},
		{
			name: "prefix without '/'",
			mutate: func(c *Config) {
				c.Routes[0].Prefix = "api"
			},
			wantErr: "prefix must start with '/'",
		},
		{
			name: "invalid regex",
			mutate: func(c *Config) {
				c.Routes[0].Prefix = ""
				c.Routes[0].Regex = "[0-9"
			},
			wantErr: "regex is invalid",
		},
		{
			name: "empty backend",
			mutate: func(c *Config) {
				c.Routes[0].Backend = ""
			},
			wantErr: "backend must not be empty",
		},
		{
			name: "invalid fallback",
			mutate: func(c *Config) {
				c.Routes[0].Fallback = "http:api-v1-backup:9001"
			},
			wantErr: "fallback is invalid",
		},
		{
			name: "limit rps equals zero",
			mutate: func(c *Config) {
				c.Routes[0].Limit = &Limit{RPS: 0}
			},
			wantErr: "limit.rps",
		},
		{
			name: "negative percent",
			mutate: func(c *Config) {
				c.Routes[0].Mirror = &Mirror{Backend: "http://v2:9002", Percent: -10}
			},
			wantErr: "mirror.percent",
		},
		{
			name: "equals ports",
			mutate: func(c *Config) {
				c.Metrics.Address = ":8080"
			},
			wantErr: "server and metrics",
		},
		{
			name: "empty routes",
			mutate: func(c *Config) {
				c.Routes = []Route{}
			},
			wantErr: "routes must not be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			tc.mutate(cfg)

			err := validate(cfg)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("want error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}
