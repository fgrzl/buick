package proxy

import (
	"io"
	"log/slog"

	"github.com/fgrzl/buick/internal/mgmt"
)

// Option configures Router construction.
type Option func(*Router)

// WithAccessLog enables one JSON object per line written to w.
func WithAccessLog(w io.Writer) Option {
	return func(rt *Router) {
		if w == nil {
			return
		}
		rt.accessLog = slog.New(slog.NewJSONHandler(w, nil))
	}
}

// WithMetrics attaches Prometheus-style counters updated per request.
func WithMetrics(m *mgmt.Metrics) Option {
	return func(rt *Router) {
		rt.metrics = m
	}
}
