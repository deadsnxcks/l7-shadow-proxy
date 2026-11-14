package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
)

func TestConfig(t *testing.T) {
	cfg, err := config.Load("../../configs/example.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	t.Logf("Loaded config: %+v", cfg)

	if len(cfg.Routes) != 5 {
		t.Fatalf("want routes = 5, get = %d", len(cfg.Routes))
	}

	apiRoute := findRoute(t, cfg, "api")

	if apiRoute.Timeout != 2*time.Second {
		t.Errorf("want timeout = 2s, get = %v", apiRoute.Timeout)
	}

	if apiRoute.Mirror == nil {
		t.Fatal("api: mirror is nil")
	}

	if apiRoute.Mirror.Percent != 10 {
		t.Errorf("want mirror percent = 10, got %d", apiRoute.Mirror.Percent)
	}
}

func TestLoad_RouteInheritsDefaultTimeout(t *testing.T) {
	path := writeConfig(t, `
env: "local"
defaults:
  timeout: 3s
routes:
  - name: api
    prefix: /api/
    backend: http://api:9001
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	r := findRoute(t, cfg, "api")
	if r.Timeout != 3*time.Second {
		t.Errorf("timeout: want 3s, got %v", r.Timeout)
	}
}

func TestLoad_UnknownFieldFails(t *testing.T) {
	path := writeConfig(t, `
env: "local"
routes:
  - name: api
    prefix: /api/
    backend: http://api:9001
    timout: 2s
`)

	if _, err := config.Load(path); err == nil {
		t.Fatal("want error for unknown field, got nil")
	}
}

func TestLoad_NotSetDefaults(t *testing.T) {
	path := writeConfig(t, `
env: "local"
routes:
  - name: api
    prefix: /api/
    backend: http://api:9001
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	r := findRoute(t, cfg, "api")
	if r.Timeout != 5*time.Second {
		t.Errorf("timeout: want 5s, got %v", r.Timeout)
	}
}

func TestLoad_NotSetMirroring(t *testing.T) {
	path := writeConfig(t, `
env: "local"
routes:
  - name: api
    prefix: /api/
    backend: http://api:9001
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if cfg.Mirroring.QueueSize != 1000 {
		t.Errorf("queue size: want 1000, got %d", cfg.Mirroring.QueueSize)
	}

	if cfg.Mirroring.Workers != 5 {
		t.Errorf("workers: want 5, got %d", cfg.Mirroring.Workers)
	}

	if cfg.Mirroring.Timeout != 5*time.Second {
		t.Errorf("timeout: want 5s, got %v", cfg.Mirroring.Timeout)
	}

	if cfg.Mirroring.MaxBodyBytes != 1024*1024 {
		t.Errorf("max body bytes: want 1MB, got %d", cfg.Mirroring.MaxBodyBytes)
	}
}

func findRoute(t *testing.T, cfg *config.Config, name string) config.Route {
	t.Helper()

	for _, route := range cfg.Routes {
		if route.Name == name {
			return route
		}
	}
	t.Fatalf("Route not found: %s", name)
	return config.Route{}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
