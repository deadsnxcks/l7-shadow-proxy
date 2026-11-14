package pipeline_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/pipeline"
	"github.com/deadsnxcks/l7-shadow-proxy/internal/router"
)

// fakeRouter всегда возвращает заданный маршрут.
type fakeRouter struct{ route *router.Route }

func (f fakeRouter) Match(*http.Request) *router.Route { return f.route }

// fakeForwarder запоминает, был ли вызов и на какой бэкенд.
type fakeForwarder struct {
	called bool
	target *url.URL
}

func (f *fakeForwarder) Forward(w http.ResponseWriter, _ *http.Request, target *url.URL) {
	f.called = true
	f.target = target
	w.WriteHeader(http.StatusOK)
}

func TestPipeline_NoRoute(t *testing.T) {
	fw := &fakeForwarder{}
	p := pipeline.New(fakeRouter{route: nil}, fw)

	w := httptest.NewRecorder()
	p.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", w.Code)
	}
	if fw.called {
		t.Error("forwarder should not be called when no route matched")
	}
}

func TestPipeline_ForwardsToRouteBackend(t *testing.T) {
	backend, _ := url.Parse("http://api-v1:9001")
	fw := &fakeForwarder{}
	p := pipeline.New(fakeRouter{route: &router.Route{Name: "api", Backend: backend}}, fw)

	w := httptest.NewRecorder()
	p.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/orders", nil))

	if !fw.called {
		t.Fatal("forwarder should be called")
	}
	if fw.target != backend {
		t.Errorf("target: want %v, got %v", backend, fw.target)
	}
}
