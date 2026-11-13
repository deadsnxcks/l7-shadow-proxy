package router

import (
	"net/http"
	"sync/atomic"
)

type snapshot struct {
	matcher Matcher
}

type Router struct {
	current atomic.Pointer[snapshot]
}

func New(m Matcher) *Router {
	rtr := &Router{}
	rtr.current.Store(&snapshot{matcher: m})
	return rtr
}

func (rtr *Router) Match(r *http.Request) *Route {
	snap := rtr.current.Load()
	return snap.matcher.Match(r)
}

func (rtr *Router) Update(m Matcher) {
	rtr.current.Store(&snapshot{matcher: m})
}
