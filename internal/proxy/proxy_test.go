package proxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRemoveHopByHop(t *testing.T) {
	h := http.Header{}
	h.Set("Connection", "keep-alive, X-Secret")
	h.Set("Keep-Alive", "timeout=5")
	h.Set("X-Secret", "1")
	h.Set("Upgrade", "websocket")
	h.Set("X-Keep", "yes")

	removeHopByHop(h)

	for _, name := range []string{"Connection", "Keep-Alive", "X-Secret", "Upgrade"} {
		if h.Get(name) != "" {
			t.Errorf("%s should be removed, got %q", name, h.Get(name))
		}
	}
	if h.Get("X-Keep") != "yes" {
		t.Errorf("X-Keep should stay, got %q", h.Get("X-Keep"))
	}
}

func TestRemoveHopByHop_ConnectionOnSeveralLines(t *testing.T) {
	h := http.Header{}
	h.Add("Connection", "X-A")
	h.Add("Connection", "X-B")
	h.Set("X-A", "1")
	h.Set("X-B", "2")

	removeHopByHop(h)

	if h.Get("X-A") != "" || h.Get("X-B") != "" {
		t.Errorf("headers listed in Connection should be removed: %v", h)
	}
}

func TestAppendForwardedFor(t *testing.T) {
	tests := []struct {
		name  string
		prior []string
		want  string
	}{
		{name: "no prior", prior: nil, want: "10.0.0.1"},
		{name: "one prior", prior: []string{"1.2.3.4"}, want: "1.2.3.4, 10.0.0.1"},
		{name: "prior on two lines", prior: []string{"1.2.3.4", "5.6.7.8"}, want: "1.2.3.4, 5.6.7.8, 10.0.0.1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			for _, v := range tc.prior {
				h.Add("X-Forwarded-For", v)
			}

			appendForwardedFor(h, "10.0.0.1")

			if got := h.Values("X-Forwarded-For"); len(got) != 1 || got[0] != tc.want {
				t.Errorf("want single line %q, got %q", tc.want, got)
			}
		})
	}
}

// fakeTransport запоминает исходящий запрос и отвечает заранее заданным ответом. Сети нет.
type fakeTransport struct {
	got  *http.Request
	resp *http.Response
	err  error
}

func (f *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.got = r
	return f.resp, f.err
}

func TestForward(t *testing.T) {
	ft := &fakeTransport{resp: &http.Response{
		StatusCode: http.StatusTeapot, // необычный код: так видно, что статус передан, а не поставлен по умолчанию
		Header:     http.Header{"X-Backend": {"b1"}, "Keep-Alive": {"timeout=5"}},
		Body:       io.NopCloser(strings.NewReader("hello")),
	}}
	f := New(ft)

	r := httptest.NewRequest(http.MethodPost, "/api/orders?x=1", strings.NewReader("body"))
	r.Header.Set("X-Test", "123")
	r.Header.Set("Connection", "X-Secret")
	r.Header.Set("X-Secret", "1")
	w := httptest.NewRecorder()

	target, _ := url.Parse("http://api-v1:9001")
	f.Forward(w, r, target)

	// что ушло на бэкенд
	out := ft.got
	if out == nil {
		t.Fatal("transport was not called")
	}
	if got := out.URL.String(); got != "http://api-v1:9001/api/orders?x=1" {
		t.Errorf("url: got %q", got)
	}
	if out.Method != http.MethodPost {
		t.Errorf("method: got %q", out.Method)
	}
	if out.Header.Get("X-Test") != "123" {
		t.Errorf("X-Test should be forwarded")
	}
	if out.Header.Get("Connection") != "" || out.Header.Get("X-Secret") != "" {
		t.Errorf("hop-by-hop headers should be removed: %v", out.Header)
	}
	if got := out.Header.Get("X-Forwarded-For"); got != "192.0.2.1" {
		t.Errorf("X-Forwarded-For: got %q", got)
	}
	if body, _ := io.ReadAll(out.Body); string(body) != "body" {
		t.Errorf("body: got %q", body)
	}

	// что вернулось клиенту
	if w.Code != http.StatusTeapot {
		t.Errorf("status: want 418, got %d", w.Code)
	}
	if w.Header().Get("X-Backend") != "b1" {
		t.Errorf("X-Backend should be copied to client")
	}
	if w.Header().Get("Keep-Alive") != "" {
		t.Errorf("Keep-Alive should not reach client")
	}
	if w.Body.String() != "hello" {
		t.Errorf("body: got %q", w.Body.String())
	}
}

func TestForward_BackendError(t *testing.T) {
	f := New(&fakeTransport{err: errors.New("connection refused")})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	target, _ := url.Parse("http://api-v1:9001")

	f.Forward(w, r, target)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status: want 502, got %d", w.Code)
	}
}
