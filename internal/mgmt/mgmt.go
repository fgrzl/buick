package mgmt

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/fgrzl/buick/internal/config"
)

// RouteReader is implemented by the proxy router for introspection.
type RouteReader interface {
	Routes() []config.Resolved
}

// Wrap returns a handler that serves /_buick/* on loopback clients before delegating to inner.
func Wrap(inner http.Handler, routes RouteReader, start time.Time, version, httpAddr, httpsAddr string, m *Metrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsLoopbackClient(r.RemoteAddr) || !IsMgmtPath(r.URL.Path) {
			inner.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		switch r.URL.Path {
		case "/_buick/health":
			rs := routes.Routes()
			resp := map[string]any{
				"status":   "ok",
				"version":  version,
				"uptime_s": int64(time.Since(start).Seconds()),
				"routes":   len(rs),
				"http":     httpAddr,
				"https":    httpsAddr,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case "/_buick/routes":
			rs := routes.Routes()
			out := make([]map[string]any, 0, len(rs))
			for _, x := range rs {
				tgts := make([]string, len(x.Targets))
				for i, u := range x.Targets {
					tgts[i] = u.String()
				}
				row := map[string]any{
					"host":             x.Host,
					"targets":          tgts,
					"websocket":        x.WebSocket,
					"read_timeout_ms":  x.ReadTimeout.Milliseconds(),
					"write_timeout_ms": x.WriteTimeout.Milliseconds(),
				}
				if len(tgts) > 0 {
					row["target"] = tgts[0]
				}
				out = append(out, row)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		case "/_buick/metrics":
			if m == nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			m.WritePrometheus(w)
		default:
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	})
}
