package reloader

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

const testDebounce = 50 * time.Millisecond

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// startReloader запускает Reloader на временном файле.
// Возвращает путь к файлу и счётчик вызовов reload.
// При завершении теста отменяет контекст и проверяет, что Run остановился.
func startReloader(t *testing.T, reloadErr error) (string, *atomic.Int32) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, "v1")

	var calls atomic.Int32
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := New(log, path, func() error {
		calls.Add(1)
		return reloadErr
	})
	r.debounce = testDebounce

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("Run did not stop after context cancel")
		}
	})

	return path, &calls
}

// warmUp пишет в файл, пока не случится первая перезагрузка:
// так мы убеждаемся, что Run уже следит за файлом.
// Возвращает число вызовов после затишья — базу для дальнейших проверок.
func warmUp(t *testing.T, path string, calls *atomic.Int32) int32 {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("reloader did not react to file change")
		}
		writeFile(t, path, "warm-up")
		time.Sleep(4 * testDebounce)
	}
	time.Sleep(4 * testDebounce) // даём долететь перезагрузкам от последних записей
	return calls.Load()
}

// waitFor ждёт, пока счётчик станет равен want.
func waitFor(t *testing.T, calls *atomic.Int32, want int32) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if calls.Load() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("want %d reload calls, got %d", want, calls.Load())
}

func TestReloader_BurstOfWritesReloadsOnce(t *testing.T) {
	path, calls := startReloader(t, nil)
	base := warmUp(t, path, calls)

	for i := range 5 {
		writeFile(t, path, string(rune('a'+i)))
	}

	waitFor(t, calls, base+1)

	time.Sleep(4 * testDebounce) // убеждаемся, что лишних перезагрузок не будет
	if got := calls.Load(); got != base+1 {
		t.Errorf("want exactly one reload for a burst, got %d", got-base)
	}
}

func TestReloader_OtherFileIgnored(t *testing.T) {
	path, calls := startReloader(t, nil)
	base := warmUp(t, path, calls)

	writeFile(t, filepath.Join(filepath.Dir(path), "other.yaml"), "x")

	time.Sleep(4 * testDebounce)
	if got := calls.Load(); got != base {
		t.Errorf("change of another file should not reload, got %d reloads", got-base)
	}
}

func TestReloader_KeepsRunningAfterError(t *testing.T) {
	path, calls := startReloader(t, errors.New("broken config"))
	base := warmUp(t, path, calls)

	writeFile(t, path, "again")

	waitFor(t, calls, base+1)
}
