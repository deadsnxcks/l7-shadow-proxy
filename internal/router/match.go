package router

import (
	"net/http"
	"strings"
)

type Matcher interface {
	Match(r *http.Request) *Route
}

func (rt *Route) matches(r *http.Request) bool {
	switch rt.Kind {
	case MatchExact:
		if rt.Path != r.URL.Path {
			return false
		}
	case MatchPrefix:
		if !strings.HasPrefix(r.URL.Path, rt.Path) {
			return false
		}
	case MatchRegex:
		if !rt.Re.MatchString(r.URL.Path) {
			return false
		}
	}

	for key, value := range rt.Headers {
		if r.Header.Get(key) != value {
			return false
		}
	}

	return true
}

func kindRank(k MatchKind) int {
	if k == MatchExact {
		return 2
	}

	if k == MatchRegex {
		return 1
	}

	return 0
}

func moreSpecific(a, b *Route) bool {
	if ra, rb := kindRank(a.Kind), kindRank(b.Kind); ra != rb {
		return ra > rb
	}

	if a.Kind == MatchPrefix && len(a.Path) != len(b.Path) {
		return len(a.Path) > len(b.Path)
	}

	if len(a.Headers) != len(b.Headers) {
		return len(a.Headers) > len(b.Headers)
	}

	return a.Index < b.Index
}

type Linear struct{ routes []*Route }

func NewLinear(routes []*Route) *Linear {
	return &Linear{routes: routes}
}

func (l *Linear) Match(r *http.Request) *Route {
	var best *Route

	for _, route := range l.routes {
		if !route.matches(r) {
			continue
		}

		if best == nil || moreSpecific(route, best) {
			best = route
		}
	}

	return best
}
