package router_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/router"
)

// matcherFor собирает Indexed из одного правила-ловушки с заданным именем.
func matcherFor(t *testing.T, name string) router.Matcher {
	t.Helper()

	routes, err := router.Build(&config.Config{Routes: []config.Route{
		{Name: name, Prefix: "/", Backend: "http://" + name + ":9000"},
	}})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return router.NewIndexed(routes)
}

func TestRouter_Update(t *testing.T) {
	rtr := router.New(matcherFor(t, "a"))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	if got := rtr.Match(req); got == nil || got.Name != "a" {
		t.Fatalf("before update: want a, got %+v", got)
	}

	rtr.Update(matcherFor(t, "b"))

	if got := rtr.Match(req); got == nil || got.Name != "b" {
		t.Fatalf("after update: want b, got %+v", got)
	}
}

// Читатели ищут маршрут, пока писатель постоянно переключает таблицы.
// Запускать с -race: детектор гонок проверит, что доступ безопасен.
func TestRouter_ConcurrentUpdate(t *testing.T) {
	ma, mb := matcherFor(t, "a"), matcherFor(t, "b")
	rtr := router.New(ma)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	done := make(chan struct{})
	var writer sync.WaitGroup
	writer.Add(1)
	go func() {
		defer writer.Done()
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			default:
			}
			if i%2 == 0 {
				rtr.Update(mb)
			} else {
				rtr.Update(ma)
			}
		}
	}()

	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for range 10000 {
				got := rtr.Match(req)
				if got == nil || (got.Name != "a" && got.Name != "b") {
					// из горутин можно t.Errorf, но не t.Fatal
					t.Errorf("unexpected route: %+v", got)
					return
				}
			}
		}()
	}

	readers.Wait()
	close(done)
	writer.Wait()
}
