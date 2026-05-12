package mgmt

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fgrzl/buick/internal/config"
)

type stubRoutes struct {
	n int
}

func (s stubRoutes) Routes() []config.Resolved {
	out := make([]config.Resolved, s.n)
	for i := 0; i < s.n; i++ {
		out[i] = config.Resolved{Host: "h"}
	}
	return out
}

func (s stubRoutes) RouteCount() int { return s.n }

func TestWrapShouldServeHealthOnLoopbackGivenGETWhenPathIsBuickHealth(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("inner should not run for mgmt path, got %s", r.URL.Path)
	})
	h := Wrap(inner, stubRoutes{n: 2}, time.Now(), "0.0.1", ":8080", "", nil)

	req := httptest.NewRequest(http.MethodGet, "/_buick/health", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestWrapShouldDelegateToInnerGivenNonLoopbackWhenPathIsBuickHealth(t *testing.T) {
	var innerHit bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerHit = true
		w.WriteHeader(http.StatusTeapot)
	})
	h := Wrap(inner, stubRoutes{}, time.Now(), "0.0.1", ":8080", "", nil)

	req := httptest.NewRequest(http.MethodGet, "/_buick/health", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !innerHit {
		t.Fatal("expected inner handler for non-loopback client")
	}
	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestWrapShouldReturn404GivenUnknownBuickPathWhenLoopbackGET(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("inner should not run, path %s", r.URL.Path)
	})
	h := Wrap(inner, stubRoutes{}, time.Now(), "0.0.1", ":8080", "", nil)

	req := httptest.NewRequest(http.MethodGet, "/_buick/nope", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
