package proxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

type Forwarder struct {
	transport http.RoundTripper
}

func New(transport http.RoundTripper) *Forwarder {
	return &Forwarder{
		transport: transport,
	}
}

func (f *Forwarder) Forward(w http.ResponseWriter, r *http.Request, target *url.URL) {
	out := &url.URL{
		Scheme:   target.Scheme,
		Host:     target.Host,
		Path:     r.URL.Path,
		RawPath:  r.URL.RawPath,
		RawQuery: r.URL.RawQuery,
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, out.String(), r.Body)
	if err != nil {
		log.Printf("failed to create request: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	req.ContentLength = r.ContentLength

	copyHeaders(req.Header, r.Header)
	removeHopByHop(req.Header)

	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		appendForwardedFor(req.Header, clientIP)
	} else {
		log.Printf("failed to parse remote addr %q: %v", r.RemoteAddr, err)
	}

	resp, err := f.transport.RoundTrip(req)
	if err != nil {
		log.Printf("backend error: %v", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	removeHopByHop(w.Header())
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("failed to copy response body: %v", err)
	}
}

func removeHopByHop(h http.Header) {
	for _, line := range h.Values("Connection") {
		for _, token := range strings.Split(line, ",") {
			h.Del(strings.TrimSpace(token))
		}
	}
	for _, header := range hopHeaders {
		h.Del(header)
	}
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func appendForwardedFor(h http.Header, clientIP string) {
	if prior := h.Values("X-Forwarded-For"); len(prior) > 0 {
		clientIP = strings.Join(prior, ", ") + ", " + clientIP
	}
	h.Set("X-Forwarded-For", clientIP)
}
