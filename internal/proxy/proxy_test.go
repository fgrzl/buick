package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/fgrzl/buick/internal/config"
)

func TestShouldReturn502GivenUnmatchedHostWhenServingRequest(t *testing.T) {
	rt := NewRouter([]config.Resolved{
		{Host: "ok.localhost", Targets: []*url.URL{mustURL(t, "http://example.invalid")}},
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

func TestShouldPropagateUpstreamStatusGivenMatchedHostWhenUpstreamResponds(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(up.Close)

	u, err := url.Parse(up.URL)
	if err != nil {
		t.Fatal(err)
	}
	rt := routerToTestUpstream(t, u)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://app.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "app.test"
	rr := httptest.NewRecorder()
	rt.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTeapot)
	}
}

func TestShouldPreserveOriginalHostOnUpstreamRequestGivenClientSendsHostWithPortWhenProxying(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Host", r.Host)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)

	u, err := url.Parse(up.URL)
	if err != nil {
		t.Fatal(err)
	}
	rt := routerToTestUpstream(t, u)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://app.test/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "app.test:9999"
	rr := httptest.NewRecorder()
	rt.ServeHTTP(rr, req)
	if got := rr.Header().Get("X-Echo-Host"); got != "app.test:9999" {
		t.Fatalf("Host forwarded: got %q, want %q", got, "app.test:9999")
	}
}

func TestShouldSetXForwardedHostToClientHostGivenMatchedRouteWhenProxying(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Forwarded-Host", r.Header.Get("X-Forwarded-Host"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)

	u, err := url.Parse(up.URL)
	if err != nil {
		t.Fatal(err)
	}
	rt := routerToTestUpstream(t, u)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://app.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "app.test:9999"
	rr := httptest.NewRecorder()
	rt.ServeHTTP(rr, req)
	if got := rr.Header().Get("X-Echo-Forwarded-Host"); got != "app.test:9999" {
		t.Fatalf("X-Forwarded-Host: got %q, want %q", got, "app.test:9999")
	}
}

func TestShouldSetXForwardedProtoToHTTPGivenPlainHTTPRequestWhenProxying(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Forwarded-Proto", r.Header.Get("X-Forwarded-Proto"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)

	u, err := url.Parse(up.URL)
	if err != nil {
		t.Fatal(err)
	}
	rt := routerToTestUpstream(t, u)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://app.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "app.test"
	rr := httptest.NewRecorder()
	rt.ServeHTTP(rr, req)
	if got := rr.Header().Get("X-Echo-Forwarded-Proto"); got != "http" {
		t.Fatalf("X-Forwarded-Proto: got %q, want http", got)
	}
}

func TestShouldSetXForwardedForGivenMatchedRouteWhenProxying(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Forwarded-For", r.Header.Get("X-Forwarded-For"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)

	u, err := url.Parse(up.URL)
	if err != nil {
		t.Fatal(err)
	}
	rt := routerToTestUpstream(t, u)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://app.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "app.test"
	req.RemoteAddr = "192.0.2.1:12345"
	rr := httptest.NewRecorder()
	rt.ServeHTTP(rr, req)
	if got := rr.Header().Get("X-Echo-Forwarded-For"); got == "" {
		t.Fatal("expected X-Forwarded-For to be set on upstream request")
	}
}

func TestShouldRoundRobinPeersGivenMultipleTargetsWhenSequentialRequests(t *testing.T) {
	s0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "A")
	}))
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "B")
	}))
	t.Cleanup(s0.Close)
	t.Cleanup(s1.Close)

	u0, err := url.Parse(s0.URL)
	if err != nil {
		t.Fatal(err)
	}
	u1, err := url.Parse(s1.URL)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRouter([]config.Resolved{
		{Host: "lb.test", Targets: []*url.URL{u0, u1}},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for i := 0; i < 8; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://lb.test/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "lb.test"
		rr := httptest.NewRecorder()
		rt.ServeHTTP(rr, req)
		want := "ABABABAB"[i : i+1]
		if got := rr.Body.String(); got != want {
			t.Fatalf("req %d: body %q want %q", i, got, want)
		}
	}
}

func routerToTestUpstream(t *testing.T, u *url.URL) *Router {
	t.Helper()
	return NewRouter([]config.Resolved{
		{Host: "app.test", Targets: []*url.URL{u}, WebSocket: false, ReadTimeout: time.Second, WriteTimeout: time.Second},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
