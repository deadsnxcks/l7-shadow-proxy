package router

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/deadsnxcks/l7-shadow-proxy/internal/config"
)

type MatchKind int

const (
	MatchExact  MatchKind = iota // 0
	MatchPrefix                  // 1
	MatchRegex                   // 2
)

type Route struct {
	Name          string
	Kind          MatchKind
	Path          string
	Re            *regexp.Regexp
	Headers       map[string]string
	Backend       *url.URL
	Fallback      *url.URL
	MirrorTo      *url.URL
	MirrorPercent int
	Timeout       time.Duration
	Index         int
}

func Build(cfg *config.Config) ([]*Route, error) {
	routes := make([]*Route, 0, len(cfg.Routes))

	for i, r := range cfg.Routes {
		var err error
		route := Route{}

		if r.Exact != "" {
			route.Path = r.Exact
			route.Kind = MatchExact
		}

		if r.Prefix != "" {
			route.Path = r.Prefix
			route.Kind = MatchPrefix
		}

		if r.Regex != "" {
			route.Re, err = regexp.Compile(r.Regex)
			if err != nil {
				return nil, fmt.Errorf("route %d (%s): compile regex: %w", i, r.Name, err)
			}

			route.Kind = MatchRegex
		}

		route.Name = r.Name

		route.Headers = make(map[string]string, len(r.Headers))
		for key, value := range r.Headers {
			route.Headers[http.CanonicalHeaderKey(key)] = value
		}

		route.Backend, err = url.Parse(r.Backend)
		if err != nil {
			return nil, fmt.Errorf("route %d (%s): parse backend: %w", i, r.Name, err)
		}

		route.Fallback, err = parseOptionalURL(r.Fallback)
		if err != nil {
			return nil, fmt.Errorf("route %d (%s): parse fallback: %w", i, r.Name, err)
		}

		if r.Mirror != nil {
			route.MirrorTo, err = parseOptionalURL(r.Mirror.Backend)
			if err != nil {
				return nil, fmt.Errorf("route %d (%s): parse mirror: %w", i, r.Name, err)
			}

			route.MirrorPercent = r.Mirror.Percent
		}

		route.Timeout = r.Timeout
		route.Index = i

		routes = append(routes, &route)
	}

	return routes, nil
}

func parseOptionalURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}
	return url.Parse(raw)
}
