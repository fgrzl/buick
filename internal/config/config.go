// Package config loads and validates Buick YAML configuration.
package config

import (
	"cmp"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Root is the top-level configuration document.
type Root struct {
	Proxy    Proxy              `yaml:"proxy"`
	Services map[string]Service `yaml:"services"`
}

// Proxy holds listener and TLS file paths.
type Proxy struct {
	HTTP     string `yaml:"http"`
	HTTPS    string `yaml:"https"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// Service describes one hostname route.
type Service struct {
	Target       string   `yaml:"target"`
	Targets      []string `yaml:"targets"`
	ReadTimeout  string   `yaml:"read_timeout"`
	WriteTimeout string   `yaml:"write_timeout"`
}

// Resolved bundles parsed fields used at runtime.
type Resolved struct {
	Host         string
	Targets      []*url.URL // non-empty; round-robin when len > 1
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

const (
	defaultHTTPReadTimeout  = 60 * time.Second
	defaultHTTPWriteTimeout = 60 * time.Second
	defaultWSReadTimeout    = 168 * time.Hour
	defaultWSWriteTimeout   = 168 * time.Hour

	defaultListenHTTP  = ":80"
	defaultListenHTTPS = ":443"
)

// Load reads and parses a YAML config file.
func Load(path string) (*Root, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var root Root
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	return &root, nil
}

// applyProxyDefaults sets standard listen addresses when http/https are omitted.
// If both cert_file and key_file are set, https defaults to :443; http defaults
// to :80 whenever it was omitted. Call before validating listener fields.
func applyProxyDefaults(root *Root) {
	if root == nil {
		return
	}
	httpEmpty := strings.TrimSpace(root.Proxy.HTTP) == ""
	httpsEmpty := strings.TrimSpace(root.Proxy.HTTPS) == ""
	if !httpEmpty && !httpsEmpty {
		return
	}
	hasCerts := strings.TrimSpace(root.Proxy.CertFile) != "" &&
		strings.TrimSpace(root.Proxy.KeyFile) != ""

	if httpEmpty && httpsEmpty {
		if hasCerts {
			root.Proxy.HTTP = defaultListenHTTP
			root.Proxy.HTTPS = defaultListenHTTPS
		} else {
			root.Proxy.HTTP = defaultListenHTTP
		}
		return
	}
	if httpsEmpty && hasCerts {
		root.Proxy.HTTPS = defaultListenHTTPS
	}
}

// Validate checks proxy and service entries and returns resolved routes.
func Validate(root *Root) ([]Resolved, error) {
	if root == nil {
		return nil, errors.New("config is nil")
	}
	applyProxyDefaults(root)
	if strings.TrimSpace(root.Proxy.HTTP) == "" && strings.TrimSpace(root.Proxy.HTTPS) == "" {
		return nil, errors.New("proxy: at least one of http or https must be set")
	}
	if len(root.Services) == 0 {
		return nil, errors.New("services: at least one hostname mapping is required")
	}

	https := strings.TrimSpace(root.Proxy.HTTPS) != ""
	if https {
		if strings.TrimSpace(root.Proxy.CertFile) == "" {
			return nil, errors.New("proxy: cert_file is required when https is set")
		}
		if strings.TrimSpace(root.Proxy.KeyFile) == "" {
			return nil, errors.New("proxy: key_file is required when https is set")
		}
	}

	var resolved []Resolved
	seen := make(map[string]struct{})
	for host, svc := range root.Services {
		nh := NormalizeHost(host)
		if nh == "" {
			return nil, fmt.Errorf("services: invalid empty hostname for entry %q", host)
		}
		if _, dup := seen[nh]; dup {
			return nil, fmt.Errorf("services: duplicate hostname after normalization: %q", nh)
		}
		seen[nh] = struct{}{}

		rawSingle := strings.TrimSpace(svc.Target)
		var rawList []string
		for _, s := range svc.Targets {
			s = strings.TrimSpace(s)
			if s != "" {
				rawList = append(rawList, s)
			}
		}
		if rawSingle != "" && len(rawList) > 0 {
			return nil, fmt.Errorf("services[%q]: specify either target or targets, not both", host)
		}
		if rawSingle == "" && len(rawList) == 0 {
			return nil, fmt.Errorf("services[%q]: target or targets is required", host)
		}

		var urls []*url.URL
		if len(rawList) > 0 {
			for _, t := range rawList {
				u, err := parseServiceTargetURL(host, t)
				if err != nil {
					return nil, err
				}
				urls = append(urls, u)
			}
		} else {
			u, err := parseServiceTargetURL(host, rawSingle)
			if err != nil {
				return nil, err
			}
			urls = []*url.URL{u}
		}

		rt, wt, err := parseTimeouts(svc)
		if err != nil {
			return nil, fmt.Errorf("services[%q]: %w", host, err)
		}

		resolved = append(resolved, Resolved{
			Host:         nh,
			Targets:      urls,
			ReadTimeout:  rt,
			WriteTimeout: wt,
		})
	}
	slices.SortFunc(resolved, func(a, b Resolved) int {
		return cmp.Compare(a.Host, b.Host)
	})
	return resolved, nil
}

func parseServiceTargetURL(serviceHost, t string) (*url.URL, error) {
	u, err := url.Parse(t)
	if err != nil {
		return nil, fmt.Errorf("services[%q]: target parse: %w", serviceHost, err)
	}
	if !u.IsAbs() || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("services[%q]: target must be an absolute URL with host (got %q)", serviceHost, t)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("services[%q]: target scheme must be http or https (got %q)", serviceHost, u.Scheme)
	}
	return u, nil
}

func parseTimeouts(s Service) (read, write time.Duration, err error) {
	defaultRead, defaultWrite := defaultHTTPReadTimeout, defaultHTTPWriteTimeout

	read = defaultRead
	write = defaultWrite

	if strings.TrimSpace(s.ReadTimeout) != "" {
		read, err = time.ParseDuration(strings.TrimSpace(s.ReadTimeout))
		if err != nil {
			return 0, 0, fmt.Errorf("read_timeout: %w", err)
		}
		if read <= 0 {
			return 0, 0, errors.New("read_timeout must be positive")
		}
	}
	if strings.TrimSpace(s.WriteTimeout) != "" {
		write, err = time.ParseDuration(strings.TrimSpace(s.WriteTimeout))
		if err != nil {
			return 0, 0, fmt.Errorf("write_timeout: %w", err)
		}
		if write <= 0 {
			return 0, 0, errors.New("write_timeout must be positive")
		}
	}
	return read, write, nil
}

// NormalizeHost strips bracketed IPv6 ports and trailing :port for matching.
func NormalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	// IPv6 with port: [::1]:8080
	if strings.HasPrefix(host, "[") {
		if i := strings.Index(host, "]"); i >= 0 {
			return host[1:i]
		}
		return strings.TrimPrefix(host, "[")
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(h)
	}
	return host
}

// HostsFromResolved returns each route's Host in slice order (matches Validate output order).
func HostsFromResolved(routes []Resolved) []string {
	hosts := make([]string, 0, len(routes))
	for _, r := range routes {
		if r.Host != "" {
			hosts = append(hosts, r.Host)
		}
	}
	return hosts
}

// MaxEffectiveTimeouts returns server-wide read/write caps from resolved routes.
// A floor of defaultWSReadTimeout / defaultWSWriteTimeout is applied so WebSocket
// and other long-lived upgrades are not cut off by HTTP-only per-route defaults.
func MaxEffectiveTimeouts(routes []Resolved) (read, write time.Duration) {
	read, write = defaultHTTPReadTimeout, defaultHTTPWriteTimeout
	for _, r := range routes {
		if r.ReadTimeout > read {
			read = r.ReadTimeout
		}
		if r.WriteTimeout > write {
			write = r.WriteTimeout
		}
	}
	if read < defaultWSReadTimeout {
		read = defaultWSReadTimeout
	}
	if write < defaultWSWriteTimeout {
		write = defaultWSWriteTimeout
	}
	return read, write
}

// HostnamesForCert returns unique hostnames (including normalized keys) for TLS SANs.
func HostnamesForCert(root *Root) []string {
	if root == nil || root.Services == nil {
		return []string{"localhost"}
	}
	set := map[string]struct{}{"localhost": {}, "127.0.0.1": {}, "::1": {}}
	for h := range root.Services {
		n := NormalizeHost(h)
		if n != "" {
			set[n] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for h := range set {
		out = append(out, h)
	}
	slices.Sort(out)
	return out
}
