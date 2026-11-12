package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/router"
)

// strategies — все стратегии поиска. Каждый тест прогоняется на каждой из них.
var strategies = []struct {
	name       string
	newMatcher func([]*router.Route) router.Matcher
}{
	{"linear", func(rs []*router.Route) router.Matcher { return router.NewLinear(rs) }},
	{"indexed", func(rs []*router.Route) router.Matcher { return router.NewIndexed(rs) }},
}

// buildMatcher собирает стратегию поиска из правил конфига.
func buildMatcher(t *testing.T, newMatcher func([]*router.Route) router.Matcher, routes []config.Route) router.Matcher {
	t.Helper()

	built, err := router.Build(&config.Config{Routes: routes})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return newMatcher(built)
}

// exampleRoutes — правила как в example.yaml плюс api-admin для проверки длины префикса.
func exampleRoutes() []config.Route {
	return []config.Route{
		{Name: "health", Exact: "/health", Backend: "http://health:9000"},
		{Name: "api", Prefix: "/api/", Backend: "http://api-v1:9001"},
		{Name: "api-beta", Prefix: "/api/", Headers: map[string]string{"X-Version": "beta"}, Backend: "http://api-v2:9002"},
		{Name: "api-admin", Prefix: "/api/admin/", Backend: "http://admin:9003"},
		{Name: "users", Regex: "^/users/[0-9]+$", Backend: "http://users:9001"},
		{Name: "default", Prefix: "/", Backend: "http://default:9000"},
	}
}

func TestMatch(t *testing.T) {
	beta := map[string]string{"X-Version": "beta"}

	tests := []struct {
		name    string
		method  string // пусто — GET
		path    string
		headers map[string]string
		want    string // имя маршрута; "" — ничего не подошло
	}{
		// уровень 1: вид условия
		{name: "exact beats prefix /", path: "/health", want: "health"},
		{name: "regex beats prefix /", path: "/users/123", want: "users"},

		// строгость exact и regex
		{name: "exact needs full match", path: "/health/db", want: "default"},
		{name: "regex needs digits", path: "/users/abc", want: "default"},

		// уровень 2: длина префикса
		{name: "longer prefix wins", path: "/api/admin/users", want: "api-admin"},
		{name: "prefix length beats headers", path: "/api/admin/users", headers: beta, want: "api-admin"},

		// уровень 3: условия на заголовки
		{name: "header condition wins", path: "/api/orders", headers: beta, want: "api-beta"},
		{name: "wrong header value", path: "/api/clients", headers: map[string]string{"X-Version": "alpha"}, want: "api"},
		{name: "no header", path: "/api/orders", want: "api"},

		// метод на выбор не влияет
		{name: "method does not matter", method: http.MethodPost, path: "/api/orders", want: "api"},

		// ловушки с префиксом
		{name: "prefix needs trailing slash", path: "/api", want: "default"},
		{name: "prefix is compared as a whole", path: "/apiv2", want: "default"},
	}

	for _, s := range strategies {
		t.Run(s.name, func(t *testing.T) {
			m := buildMatcher(t, s.newMatcher, exampleRoutes())

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					method := tc.method
					if method == "" {
						method = http.MethodGet
					}

					r := httptest.NewRequest(method, tc.path, nil)
					for k, v := range tc.headers {
						r.Header.Set(k, v)
					}

					got := ""
					if route := m.Match(r); route != nil {
						got = route.Name
					}
					if got != tc.want {
						t.Errorf("%s %s: want %q, got %q", method, tc.path, tc.want, got)
					}
				})
			}
		})
	}
}

// Уровень 4: при полном равенстве побеждает правило выше в файле.
func TestMatch_RegexOrder(t *testing.T) {
	for _, s := range strategies {
		t.Run(s.name, func(t *testing.T) {
			m := buildMatcher(t, s.newMatcher, []config.Route{
				{Name: "first", Regex: "^/a", Backend: "http://a:9000"},
				{Name: "second", Regex: "^/a/b", Backend: "http://b:9000"},
			})

			r := httptest.NewRequest(http.MethodGet, "/a/b", nil)
			if got := m.Match(r); got == nil || got.Name != "first" {
				t.Errorf("want route first, got %+v", got)
			}
		})
	}
}

// Без маршрута по умолчанию неподходящий запрос не находит ничего.
func TestMatch_NoRoute(t *testing.T) {
	for _, s := range strategies {
		t.Run(s.name, func(t *testing.T) {
			m := buildMatcher(t, s.newMatcher, []config.Route{
				{Name: "api", Prefix: "/api/", Backend: "http://api:9001"},
			})

			r := httptest.NewRequest(http.MethodGet, "/unknown", nil)
			if got := m.Match(r); got != nil {
				t.Errorf("want nil, got %q", got.Name)
			}
		})
	}
}
