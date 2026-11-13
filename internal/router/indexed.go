package router

import "net/http"

type Indexed struct {
	exact  map[string][]*Route
	prefix *trieNode
	regex  []*Route
}

func NewIndexed(routes []*Route) *Indexed {
	exactMap := make(map[string][]*Route)
	prefixTrie := newTrieNode()
	var regexRoutes []*Route

	for _, route := range routes {
		switch route.Kind {
		case MatchExact:
			exactMap[route.Path] = append(exactMap[route.Path], route)
		case MatchPrefix:
			prefixTrie.insert(route.Path, route)
		case MatchRegex:
			regexRoutes = append(regexRoutes, route)
		}
	}

	return &Indexed{
		exact:  exactMap,
		prefix: prefixTrie,
		regex:  regexRoutes,
	}
}

func (x *Indexed) Match(r *http.Request) *Route {
	var best *Route

	findBest := func(rt *Route) {
		if rt.matches(r) && (best == nil || moreSpecific(rt, best)) {
			best = rt
		}
	}

	for _, rt := range x.exact[r.URL.Path] {
		findBest(rt)
	}

	x.prefix.walk(r.URL.Path, findBest)

	for _, rt := range x.regex {
		findBest(rt)
	}

	return best
}
