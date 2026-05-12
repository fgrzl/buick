package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
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
	u := echoUpstream(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	rt := routerToTestUpstream(t, u)
	rr := getThroughRouter(t, rt, "app.test", "/", "")
	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTeapot)
	}
}

func TestShouldPreserveOriginalHostOnUpstreamRequestGivenClientSendsHostWithPortWhenProxying(t *testing.T) {
	u := echoUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Host", r.Host)
		w.WriteHeader(http.StatusOK)
	})
	rt := routerToTestUpstream(t, u)
	rr := getThroughRouter(t, rt, "app.test:9999", "/foo", "")
	if got := rr.Header().Get("X-Echo-Host"); got != "app.test:9999" {
		t.Fatalf("Host forwarded: got %q, want %q", got, "app.test:9999")
	}
}

func TestShouldSetXForwardedHostToClientHostGivenMatchedRouteWhenProxying(t *testing.T) {
	u := echoUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Forwarded-Host", r.Header.Get("X-Forwarded-Host"))
		w.WriteHeader(http.StatusOK)
	})
	rt := routerToTestUpstream(t, u)
	rr := getThroughRouter(t, rt, "app.test:9999", "/", "")
	if got := rr.Header().Get("X-Echo-Forwarded-Host"); got != "app.test:9999" {
		t.Fatalf("X-Forwarded-Host: got %q, want %q", got, "app.test:9999")
	}
}

func TestShouldSetXForwardedProtoToHTTPGivenPlainHTTPRequestWhenProxying(t *testing.T) {
	u := echoUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Forwarded-Proto", r.Header.Get("X-Forwarded-Proto"))
		w.WriteHeader(http.StatusOK)
	})
	rt := routerToTestUpstream(t, u)
	rr := getThroughRouter(t, rt, "app.test", "/", "")
	if got := rr.Header().Get("X-Echo-Forwarded-Proto"); got != "http" {
		t.Fatalf("X-Forwarded-Proto: got %q, want http", got)
	}
}

func TestShouldSetXForwardedForGivenMatchedRouteWhenProxying(t *testing.T) {
	u := echoUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Forwarded-For", r.Header.Get("X-Forwarded-For"))
		w.WriteHeader(http.StatusOK)
	})
	rt := routerToTestUpstream(t, u)
	rr := getThroughRouter(t, rt, "app.test", "/", "192.0.2.1:12345")
	if got := rr.Header().Get("X-Echo-Forwarded-For"); got == "" {
		t.Fatal("expected X-Forwarded-For to be set on upstream request")
	}
}

func TestShouldRoundRobinPeersGivenMultipleTargetsWhenSequentialRequests(t *testing.T) {
	u0 := echoUpstream(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "A")
	})
	u1 := echoUpstream(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "B")
	})

	rt := NewRouter([]config.Resolved{
		{Host: "lb.test", Targets: []*url.URL{u0, u1}},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for i := 0; i < 8; i++ {
		rr := getThroughRouter(t, rt, "lb.test", "/", "")
		want := "ABABABAB"[i : i+1]
		if got := rr.Body.String(); got != want {
			t.Fatalf("req %d: body %q want %q", i, got, want)
		}
	}
}

func TestShouldReturnRoutesSortedByHostGivenUnorderedRouteSliceWhenRoutesCalled(t *testing.T) {
	u := mustURL(t, "http://127.0.0.1:9")
	rt := NewRouter([]config.Resolved{
		{Host: "z.h", Targets: []*url.URL{u}},
		{Host: "a.h", Targets: []*url.URL{u}},
		{Host: "m.h", Targets: []*url.URL{u}},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	rs := rt.Routes()
	if len(rs) != 3 {
		t.Fatalf("len = %d", len(rs))
	}
	got := []string{rs[0].Host, rs[1].Host, rs[2].Host}
	want := []string{"a.h", "m.h", "z.h"}
	if !slices.Equal(got, want) {
		t.Fatalf("Routes() = %v, want %v", got, want)
	}
}

func TestShouldMatchRouteCountGivenRouterWithMultipleHostsWhenRouteCountCalled(t *testing.T) {
	u := mustURL(t, "http://127.0.0.1:9")
	rt := NewRouter([]config.Resolved{
		{Host: "z.h", Targets: []*url.URL{u}},
		{Host: "a.h", Targets: []*url.URL{u}},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if n, m := rt.RouteCount(), len(rt.Routes()); n != m {
		t.Fatalf("RouteCount=%d, len(Routes())=%d", n, m)
	}
}

func echoUpstream(t *testing.T, echo http.HandlerFunc) *url.URL {
	t.Helper()
	srv := httptest.NewServer(echo)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func getThroughRouter(t *testing.T, rt *Router, host, urlPath, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	if !strings.HasPrefix(urlPath, "/") {
		t.Fatalf("urlPath must start with /, got %q", urlPath)
	}
	reqURL := "http://" + host + urlPath
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	rr := httptest.NewRecorder()
	rt.ServeHTTP(rr, req)
	return rr
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
