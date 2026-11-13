package router_test

import (
	"testing"
	"time"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/router"
)

func testConfig() *config.Config {
	return &config.Config{
		Routes: []config.Route{
			{Name: "health", Exact: "/health", Backend: "http://health:9000"},
			{
				Name:     "api-beta",
				Prefix:   "/api/",
				Headers:  map[string]string{"x-version": "beta"},
				Backend:  "http://api-v2:9002",
				Fallback: "http://api-backup:9001",
				Mirror:   &config.Mirror{Backend: "http://shadow:9003", Percent: 10},
				Timeout:  2 * time.Second,
			},
			{Name: "users", Regex: "^/users/[0-9]+$", Backend: "http://users:9001"},
		},
	}
}

func TestBuild(t *testing.T) {
	cfg := testConfig()

	routes, err := router.Build(cfg)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("want 3 routes, got %d", len(routes))
	}

	t.Run("exact route without optional parts", func(t *testing.T) {
		r := routes[0]
		if r.Kind != router.MatchExact || r.Path != "/health" {
			t.Errorf("want exact /health, got kind=%v path=%q", r.Kind, r.Path)
		}
		if r.Fallback != nil {
			t.Errorf("fallback: want nil, got %v", r.Fallback)
		}
		if r.MirrorTo != nil {
			t.Errorf("mirror: want nil, got %v", r.MirrorTo)
		}
		if r.Index != 0 {
			t.Errorf("index: want 0, got %d", r.Index)
		}
	})

	t.Run("prefix route with headers, fallback and mirror", func(t *testing.T) {
		r := routes[1]
		if r.Kind != router.MatchPrefix || r.Path != "/api/" {
			t.Errorf("want prefix /api/, got kind=%v path=%q", r.Kind, r.Path)
		}
		if r.Backend == nil || r.Backend.Host != "api-v2:9002" {
			t.Errorf("backend: want host api-v2:9002, got %v", r.Backend)
		}
		if r.Fallback == nil || r.Fallback.Host != "api-backup:9001" {
			t.Errorf("fallback: want host api-backup:9001, got %v", r.Fallback)
		}
		if r.MirrorTo == nil || r.MirrorTo.Host != "shadow:9003" {
			t.Errorf("mirror: want host shadow:9003, got %v", r.MirrorTo)
		}
		if r.MirrorPercent != 10 {
			t.Errorf("mirror percent: want 10, got %d", r.MirrorPercent)
		}
		if r.Timeout != 2*time.Second {
			t.Errorf("timeout: want 2s, got %v", r.Timeout)
		}
		if r.Headers["X-Version"] != "beta" {
			t.Errorf("headers: want X-Version=beta, got %v", r.Headers)
		}
		if r.Index != 1 {
			t.Errorf("index: want 1, got %d", r.Index)
		}
	})

	t.Run("regex route", func(t *testing.T) {
		r := routes[2]
		if r.Kind != router.MatchRegex || r.Re == nil {
			t.Fatalf("want compiled regex, got kind=%v re=%v", r.Kind, r.Re)
		}
		if !r.Re.MatchString("/users/42") {
			t.Error("regex should match /users/42")
		}
		if r.Re.MatchString("/users/abc") {
			t.Error("regex should not match /users/abc")
		}
		if r.Index != 2 {
			t.Errorf("index: want 2, got %d", r.Index)
		}
	})

	t.Run("config is not modified", func(t *testing.T) {
		h := cfg.Routes[1].Headers
		if len(h) != 1 || h["x-version"] != "beta" {
			t.Errorf("config headers changed: %v", h)
		}
	})
}
