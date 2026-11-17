package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
)

const configA = `
env: local
routes:
  - name: a
    prefix: /
    backend: http://a:9000
`

const configB = `
env: local
routes:
  - name: b
    prefix: /
    backend: http://b:9000
`

// Сломан: префикс без слэша не пройдёт проверку.
const configBroken = `
env: local
routes:
  - name: broken
    prefix: no-leading-slash
    backend: http://c:9000
`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// matchName возвращает имя маршрута, который сейчас выбирает роутер.
func matchName(a *App) string {
	route := a.rtr.Match(httptest.NewRequest(http.MethodGet, "/x", nil))
	if route == nil {
		return ""
	}
	return route.Name
}

func TestReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, configA)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil)) // логи в тесте не нужны
	a, err := New(log, cfg, path)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if got := matchName(a); got != "a" {
		t.Fatalf("before reload: want a, got %q", got)
	}

	// правильный новый конфиг применяется
	writeFile(t, path, configB)
	if err := a.reloadConfig(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := matchName(a); got != "b" {
		t.Fatalf("after reload: want b, got %q", got)
	}

	// сломанный конфиг не применяется, работаем на прежнем
	writeFile(t, path, configBroken)
	if err := a.reloadConfig(); err == nil {
		t.Fatal("want error for broken config, got nil")
	}
	if got := matchName(a); got != "b" {
		t.Fatalf("after broken reload: want b, got %q", got)
	}
}
