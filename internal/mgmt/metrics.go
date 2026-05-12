package mgmt

import (
	"fmt"
	"io"
	"slices"
	"sync"
	"sync/atomic"
)

// Metrics holds Prometheus-style counters using only the standard library.
type Metrics struct {
	total   atomic.Uint64
	active  atomic.Int64
	unknown atomic.Uint64
	mu      sync.RWMutex
	byHost  map[string]*atomic.Uint64
}

// NewMetrics preallocates per-host counters for known route hostnames.
func NewMetrics(hosts []string) *Metrics {
	m := &Metrics{byHost: make(map[string]*atomic.Uint64, len(hosts)+1)}
	for _, h := range hosts {
		if h == "" {
			continue
		}
		m.byHost[h] = new(atomic.Uint64)
	}
	return m
}

// SyncHosts replaces the per-host counter map while preserving existing counters when possible.
func (m *Metrics) SyncHosts(hosts []string) {
	next := make(map[string]*atomic.Uint64, len(hosts)+1)
	m.mu.Lock()
	for _, h := range hosts {
		if h == "" {
			continue
		}
		if old, ok := m.byHost[h]; ok {
			next[h] = old
		} else {
			next[h] = new(atomic.Uint64)
		}
	}
	m.byHost = next
	m.mu.Unlock()
}

// ActiveInc increments in-flight requests.
func (m *Metrics) ActiveInc() { m.active.Add(1) }

// ActiveDec decrements in-flight requests.
func (m *Metrics) ActiveDec() { m.active.Add(-1) }

// Record counts one completed request (matched is false for unknown-host 502).
func (m *Metrics) Record(normalizedHost string, matched bool) {
	m.total.Add(1)
	if !matched {
		m.unknown.Add(1)
		return
	}
	m.mu.RLock()
	c, ok := m.byHost[normalizedHost]
	m.mu.RUnlock()
	if ok {
		c.Add(1)
		return
	}
	m.unknown.Add(1)
}

// WritePrometheus writes a minimal text exposition to w.
func (m *Metrics) WritePrometheus(w io.Writer) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hosts := make([]string, 0, len(m.byHost))
	for host := range m.byHost {
		hosts = append(hosts, host)
	}
	slices.Sort(hosts)
	for _, host := range hosts {
		c := m.byHost[host]
		_, _ = fmt.Fprintf(w, "buick_requests_total{host=%q} %d\n", host, c.Load())
	}
	if u := m.unknown.Load(); u > 0 {
		_, _ = fmt.Fprintf(w, "buick_requests_total{host=\"unknown\"} %d\n", u)
	}
	_, _ = fmt.Fprintf(w, "buick_requests_grand_total %d\n", m.total.Load())
	_, _ = fmt.Fprintf(w, "buick_active_requests %d\n", m.active.Load())
}
