package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/router"
)

// syntheticRoutes генерирует n правил (чётные — exact, нечётные — prefix) и маршрут по умолчанию.
func syntheticRoutes(n int) []config.Route {
	routes := make([]config.Route, 0, n+1)
	for i := 0; i < n; i++ {
		r := config.Route{Name: fmt.Sprintf("r%d", i), Backend: "http://backend:9000"}
		if i%2 == 0 {
			r.Exact = fmt.Sprintf("/exact/%d", i)
		} else {
			r.Prefix = fmt.Sprintf("/svc%d/", i)
		}
		routes = append(routes, r)
	}
	return append(routes, config.Route{Name: "default", Prefix: "/", Backend: "http://default:9000"})
}

func BenchmarkMatch(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		routes, err := router.Build(&config.Config{Routes: syntheticRoutes(n)})
		if err != nil {
			b.Fatal(err)
		}

		want := fmt.Sprintf("r%d", n-1)
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/svc%d/items/42", n-1), nil)

		for _, s := range strategies {
			m := s.newMatcher(routes)

			if got := m.Match(req); got == nil || got.Name != want {
				b.Fatalf("%s: want %s, got %+v", s.name, want, got)
			}

			b.Run(fmt.Sprintf("%s/rules=%d", s.name, n), func(b *testing.B) {
				for b.Loop() {
					m.Match(req)
				}
			})
		}
	}
}
