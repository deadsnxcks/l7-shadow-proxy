package pipeline

import (
	"net/http"
	"net/url"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/router"
)

type routeMatcher interface {
	Match(r *http.Request) *router.Route
}

type forwarder interface {
	Forward(w http.ResponseWriter, r *http.Request, target *url.URL)
}

type Pipeline struct {
	router routeMatcher
	proxy  forwarder
}

func New(rt routeMatcher, fw forwarder) *Pipeline {
	return &Pipeline{
		router: rt,
		proxy:  fw,
	}
}

func (p *Pipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route := p.router.Match(r)
	if route == nil {
		http.Error(w, "no route", http.StatusNotFound)
		return
	}

	p.proxy.Forward(w, r, route.Backend)
}
