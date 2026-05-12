// Package proxy implements the Host-based reverse proxy router.
package proxy

import (
	"bufio"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fgrzl/buick/internal/config"
	"github.com/fgrzl/buick/internal/mgmt"
)

// Router matches Host and forwards via httputil.ReverseProxy.
type Router struct {
	log       *slog.Logger
	accessLog *slog.Logger
	metrics   *mgmt.Metrics

	mu     sync.RWMutex
	routes map[string]config.Resolved
	pools  map[string]*upstreamPool
}

// upstreamPool round-robins across one or more reverse proxies (one per target URL).
type upstreamPool struct {
	proxies []*httputil.ReverseProxy
	targets []*url.URL
	next    atomic.Uint64
}

// NewRouter builds a handler table from validated routes.
func NewRouter(routes []config.Resolved, log *slog.Logger, opts ...Option) *Router {
	if log == nil {
		log = slog.Default()
	}
	rt := &Router{
		log:    log,
		routes: buildRouteMap(routes),
		pools:  make(map[string]*upstreamPool),
	}
	for _, o := range opts {
		o(rt)
	}
	return rt
}

func buildRouteMap(routes []config.Resolved) map[string]config.Resolved {
	m := make(map[string]config.Resolved, len(routes))
	for _, r := range routes {
		m[r.Host] = r
	}
	return m
}

// Routes returns a snapshot copy of configured routes (for management API).
func (rt *Router) Routes() []config.Resolved {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	out := make([]config.Resolved, 0, len(rt.routes))
	for _, r := range rt.routes {
		out = append(out, r)
	}
	return out
}

// Reload replaces the route table and clears the reverse-proxy cache.
func (rt *Router) Reload(routes []config.Resolved) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.routes = buildRouteMap(routes)
	rt.pools = make(map[string]*upstreamPool)
	if rt.metrics != nil {
		hosts := make([]string, 0, len(routes))
		for _, r := range routes {
			hosts = append(hosts, r.Host)
		}
		rt.metrics.SyncHosts(hosts)
	}
}

func poolKey(r config.Resolved) string {
	var b strings.Builder
	b.WriteString(r.Host)
	b.WriteByte('|')
	for i, u := range r.Targets {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(u.String())
	}
	return b.String()
}

func (rt *Router) poolFor(r config.Resolved) *upstreamPool {
	key := poolKey(r)
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if p, ok := rt.pools[key]; ok {
		return p
	}
	p := newUpstreamPool(r, rt.log)
	rt.pools[key] = p
	return p
}

func newUpstreamPool(r config.Resolved, log *slog.Logger) *upstreamPool {
	proxies := make([]*httputil.ReverseProxy, len(r.Targets))
	for i := range r.Targets {
		proxies[i] = newReverseProxy(r.Targets[i], r.WebSocket, log)
	}
	return &upstreamPool{proxies: proxies, targets: append([]*url.URL(nil), r.Targets...)}
}

func (p *upstreamPool) pick() int {
	n := uint64(len(p.proxies))
	if n == 0 {
		return 0
	}
	i := p.next.Add(1)
	return int((i - 1) % n)
}

func newReverseProxy(target *url.URL, websocket bool, log *slog.Logger) *httputil.ReverseProxy {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}
	if target.Scheme == "https" {
		transport.TLSClientConfig.InsecureSkipVerify = false
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.Transport = transport
	if websocket {
		rp.FlushInterval = -1
	}

	director := rp.Director
	rp.Director = func(req *http.Request) {
		origHost := req.Host
		if h := req.Header.Get("Host"); h != "" {
			origHost = h
		}
		director(req)
		req.Host = origHost
		req.Header.Set("Host", origHost)
		applyForwardedHeaders(req, origHost)
	}

	rp.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Warn("upstream unreachable", "err", err, "host", req.Host, "path", req.URL.Path, "target", target.String())
		w.WriteHeader(http.StatusBadGateway)
	}

	return rp
}

func applyForwardedHeaders(req *http.Request, forwardedHost string) {
	if forwardedHost == "" {
		forwardedHost = req.Host
	}

	proto := "http"
	if req.TLS != nil {
		proto = "https"
	}

	req.Header.Set("X-Forwarded-Host", forwardedHost)
	req.Header.Set("X-Forwarded-Proto", proto)

	clientIP, _, splitErr := net.SplitHostPort(req.RemoteAddr)
	if splitErr != nil {
		clientIP = req.RemoteAddr
	}
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}
	if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
		req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
	} else {
		req.Header.Set("X-Forwarded-For", clientIP)
	}
}

// ServeHTTP implements http.Handler.
func (rt *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if rt.metrics != nil {
		rt.metrics.ActiveInc()
		defer rt.metrics.ActiveDec()
	}

	host := config.NormalizeHost(req.Host)
	rt.mu.RLock()
	route, ok := rt.routes[host]
	rt.mu.RUnlock()
	if !ok {
		rt.log.Warn("unknown host", "host", req.Host, "normalized", host, "path", req.URL.Path)
		if rt.metrics != nil {
			rt.metrics.Record(host, false)
		}
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	start := time.Now()
	lw := &logResponseWriter{ResponseWriter: w, status: http.StatusOK}

	pool := rt.poolFor(route)
	ix := pool.pick()
	chosen := pool.targets[ix]
	pool.proxies[ix].ServeHTTP(lw, req)

	dur := time.Since(start)
	attrs := []any{
		"method", req.Method,
		"host", req.Host,
		"path", req.URL.Path,
		"target", chosen.String(),
		"status", lw.status,
		"duration_ms", dur.Milliseconds(),
	}
	if lw.err != nil {
		attrs = append(attrs, "err", lw.err.Error())
	}
	rt.log.Info("request", attrs...)

	if rt.accessLog != nil {
		rt.accessLog.Info("access",
			slog.Time("time", time.Now().UTC()),
			slog.String("method", req.Method),
			slog.String("host", req.Host),
			slog.String("path", req.URL.Path),
			slog.Int("status", lw.status),
			slog.Int64("duration_ms", dur.Milliseconds()),
			slog.Int64("bytes_out", lw.bytesOut),
		)
	}
	if rt.metrics != nil {
		rt.metrics.Record(host, true)
	}
}

type logResponseWriter struct {
	http.ResponseWriter
	status   int
	err      error
	bytesOut int64
}

func (lw *logResponseWriter) WriteHeader(code int) {
	lw.status = code
	lw.ResponseWriter.WriteHeader(code)
}

func (lw *logResponseWriter) Write(b []byte) (int, error) {
	n, err := lw.ResponseWriter.Write(b)
	if n > 0 {
		lw.bytesOut += int64(n)
	}
	if err != nil && lw.err == nil {
		lw.err = err
	}
	return n, err
}

// Unwrap for http.ResponseController and similar.
func (lw *logResponseWriter) Unwrap() http.ResponseWriter {
	return lw.ResponseWriter
}

func (lw *logResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := lw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacker not supported")
	}
	return h.Hijack()
}

func (lw *logResponseWriter) Flush() {
	if f, ok := lw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Ensure Router implements mgmt.RouteReader.
var _ mgmt.RouteReader = (*Router)(nil)
