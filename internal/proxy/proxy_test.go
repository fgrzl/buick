package proxy

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/fgrzl/buick/internal/config"
)

func TestUnknownHost502(t *testing.T) {
	rt := NewRouter([]config.Resolved{
		{Host: "ok.localhost", Target: mustURL(t, "http://example.invalid")},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://nope.localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "nope.localhost"
	rr := httptest.NewRecorder()
	rt.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rr.Code)
	}
}

func TestForwardsAndHeaders(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Host", r.Host)
		w.Header().Set("X-Echo-Forwarded-Host", r.Header.Get("X-Forwarded-Host"))
		w.Header().Set("X-Echo-Forwarded-Proto", r.Header.Get("X-Forwarded-Proto"))
		w.Header().Set("X-Echo-Forwarded-For", r.Header.Get("X-Forwarded-For"))
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(up.Close)

	u, err := url.Parse(up.URL)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRouter([]config.Resolved{
		{Host: "app.test", Target: u, WebSocket: false, ReadTimeout: time.Second, WriteTimeout: time.Second},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://app.test/foo?x=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "app.test:9999"
	rr := httptest.NewRecorder()
	rt.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d", rr.Code)
	}
	if got := rr.Header().Get("X-Echo-Host"); got != "app.test:9999" {
		t.Fatalf("Host forwarded: got %q", got)
	}
	if got := rr.Header().Get("X-Echo-Forwarded-Host"); got != "app.test:9999" {
		t.Fatalf("X-Forwarded-Host: got %q", got)
	}
	if got := rr.Header().Get("X-Echo-Forwarded-Proto"); got != "http" {
		t.Fatalf("X-Forwarded-Proto: got %q", got)
	}
	if got := rr.Header().Get("X-Echo-Forwarded-For"); got == "" {
		t.Fatal("expected X-Forwarded-For to be set")
	}
}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
