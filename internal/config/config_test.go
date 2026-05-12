package config

import (
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestShouldNormalizeHostForRoutingGivenRepresentativeHostStringsWhenNormalizeHostCalled(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"service1.localhost", "service1.localhost"},
		{"Service1.Localhost", "service1.localhost"},
		{"service1.localhost:80", "service1.localhost"},
		{"service1.localhost:443", "service1.localhost"},
		{"[::1]:8443", "::1"},
		{"127.0.0.1:3000", "127.0.0.1"},
	}
	for _, tc := range tests {
		if got := NormalizeHost(tc.in); got != tc.want {
			t.Errorf("NormalizeHost(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestShouldResolveSingleTargetGivenYAMLWithTargetWhenLoadAndValidate(t *testing.T) {
	routes := mustLoadValidateYAML(t, `
proxy:
  http: ":0"

services:
  app.localhost:
    target: "http://127.0.0.1:9"
`)
	if len(routes) != 1 {
		t.Fatalf("routes: got %d want 1", len(routes))
	}
	if routes[0].Host != "app.localhost" {
		t.Fatalf("host: got %q", routes[0].Host)
	}
	if routes[0].Targets[0].String() != "http://127.0.0.1:9" {
		t.Fatalf("target: got %q", routes[0].Targets[0].String())
	}
}

func TestShouldRejectConfigGivenEmptyServicesWhenValidate(t *testing.T) {
	_, err := Validate(&Root{Proxy: Proxy{HTTP: ":80"}, Services: map[string]Service{}})
	if err == nil {
		t.Fatal("expected error for empty services")
	}
}

func TestShouldApplyDefaultHTTPGivenEmptyListenersAndNoCertsWhenValidate(t *testing.T) {
	root := &Root{
		Proxy:    Proxy{},
		Services: map[string]Service{"a": {Target: "http://127.0.0.1:1"}},
	}
	if _, err := Validate(root); err != nil {
		t.Fatal(err)
	}
	if root.Proxy.HTTP != defaultListenHTTP {
		t.Fatalf("http: got %q want %q", root.Proxy.HTTP, defaultListenHTTP)
	}
	if strings.TrimSpace(root.Proxy.HTTPS) != "" {
		t.Fatalf("https: got %q want empty", root.Proxy.HTTPS)
	}
}

func TestShouldApplyDefaultHTTPAndHTTPSGivenEmptyListenersAndCertPathsWhenValidate(t *testing.T) {
	root := &Root{
		Proxy: Proxy{CertFile: "c.pem", KeyFile: "k.pem"},
		Services: map[string]Service{
			"a": {Target: "http://127.0.0.1:1"},
		},
	}
	if _, err := Validate(root); err != nil {
		t.Fatal(err)
	}
	if root.Proxy.HTTP != defaultListenHTTP || root.Proxy.HTTPS != defaultListenHTTPS {
		t.Fatalf("proxy: %+v", root.Proxy)
	}
}

func TestShouldRejectConfigGivenHTTPSWithoutCertPathsWhenValidate(t *testing.T) {
	_, err := Validate(&Root{
		Proxy: Proxy{HTTP: ":80", HTTPS: ":443"},
		Services: map[string]Service{
			"a": {Target: "http://x"},
		},
	})
	if err == nil {
		t.Fatal("expected error when https without cert paths")
	}
}

func TestShouldRejectConfigGivenMalformedTargetURLWhenValidate(t *testing.T) {
	_, err := Validate(&Root{
		Proxy: Proxy{HTTP: ":80"},
		Services: map[string]Service{
			"a": {Target: "not-a-url"},
		},
	})
	if err == nil {
		t.Fatal("expected error for bad target URL")
	}
}

func TestShouldRejectConfigGivenBothTargetAndTargetsWhenValidate(t *testing.T) {
	_, err := Validate(&Root{
		Proxy: Proxy{HTTP: ":80"},
		Services: map[string]Service{
			"a": {Target: "http://x", Targets: []string{"http://y"}},
		},
	})
	if err == nil {
		t.Fatal("expected error when both target and targets set")
	}
}

func TestShouldRejectConfigGivenDuplicateNormalizedHostsWhenValidate(t *testing.T) {
	_, err := Validate(&Root{
		Proxy: Proxy{HTTP: ":80"},
		Services: map[string]Service{
			"foo":      {Target: "http://x"},
			"foo:8080": {Target: "http://y"},
		},
	})
	if err == nil {
		t.Fatal("expected error for duplicate normalized hosts")
	}
}

func TestShouldReturnRoutesSortedByHostGivenUnorderedYAMLMapKeysWhenValidate(t *testing.T) {
	routes := mustLoadValidateYAML(t, `
proxy:
  http: ":0"

services:
  zzz.localhost:
    target: "http://127.0.0.1:3"
  aaa.localhost:
    target: "http://127.0.0.1:1"
  mmm.localhost:
    target: "http://127.0.0.1:2"
`)
	if len(routes) != 3 {
		t.Fatalf("len = %d", len(routes))
	}
	hosts := []string{routes[0].Host, routes[1].Host, routes[2].Host}
	if !slices.IsSorted(hosts) {
		t.Fatalf("routes not sorted by host: %v", hosts)
	}
	if hosts[0] != "aaa.localhost" || hosts[1] != "mmm.localhost" || hosts[2] != "zzz.localhost" {
		t.Fatalf("unexpected order: %v", hosts)
	}
}

func TestShouldReturnSortedHostnamesForCertGivenServiceMapWhenHostnamesForCertCalled(t *testing.T) {
	root := &Root{
		Proxy: Proxy{HTTP: ":80", HTTPS: ":443", CertFile: "c.pem", KeyFile: "k.pem"},
		Services: map[string]Service{
			"zebra.test": {Target: "http://x"},
			"apple.test": {Target: "http://y"},
		},
	}
	got := HostnamesForCert(root)
	if !slices.IsSorted(got) {
		t.Fatalf("not sorted: %v", got)
	}
	wantSubset := []string{"127.0.0.1", "::1", "localhost", "apple.test", "zebra.test"}
	for _, w := range wantSubset {
		if !slices.Contains(got, w) {
			t.Fatalf("missing %q in %v", w, got)
		}
	}
}

func TestShouldParseMultipleTargetsGivenYAMLWithTargetsListWhenLoadAndValidate(t *testing.T) {
	routes := mustLoadValidateYAML(t, `
proxy:
  http: ":0"

services:
  app.localhost:
    targets:
      - "http://127.0.0.1:1"
      - "http://127.0.0.1:2"
`)
	if len(routes) != 1 || len(routes[0].Targets) != 2 {
		t.Fatalf("routes: %+v", routes)
	}
	if routes[0].Targets[0].String() != "http://127.0.0.1:1" || routes[0].Targets[1].String() != "http://127.0.0.1:2" {
		t.Fatalf("targets: %+v", routes[0].Targets)
	}
}

func TestShouldApplyLongLivedFloorGivenHTTPDefaultTimeoutsWhenMaxEffectiveTimeoutsCalled(t *testing.T) {
	u := mustParseURL(t, "http://127.0.0.1:1")
	routes := []Resolved{
		{Host: "a", Targets: []*url.URL{u}, ReadTimeout: defaultHTTPReadTimeout, WriteTimeout: defaultHTTPWriteTimeout},
	}
	read, write := MaxEffectiveTimeouts(routes)
	if read != defaultWSReadTimeout || write != defaultWSWriteTimeout {
		t.Fatalf("read=%v write=%v", read, write)
	}
}

func TestShouldHonorExplicitReadAboveWebSocketDefaultWhenMaxEffectiveTimeoutsCalled(t *testing.T) {
	u := mustParseURL(t, "http://127.0.0.1:1")
	r200h := 200 * time.Hour
	routes := []Resolved{{Host: "a", Targets: []*url.URL{u}, ReadTimeout: r200h, WriteTimeout: defaultHTTPWriteTimeout}}
	read, write := MaxEffectiveTimeouts(routes)
	if read != r200h {
		t.Fatalf("read=%v want %v", read, r200h)
	}
	if write != defaultWSWriteTimeout {
		t.Fatalf("write=%v", write)
	}
}

func TestShouldReturnHostsInOrderGivenResolvedSliceWhenHostsFromResolvedCalled(t *testing.T) {
	routes := []Resolved{
		{Host: "b", Targets: []*url.URL{mustParseURL(t, "http://127.0.0.1:1")}},
		{Host: "a", Targets: []*url.URL{mustParseURL(t, "http://127.0.0.1:2")}},
	}
	got := HostsFromResolved(routes)
	if !slices.Equal(got, []string{"b", "a"}) {
		t.Fatalf("got %v", got)
	}
}

func TestShouldApplyListenDefaultsGivenYAMLWithOnlyCertsAndServicesWhenLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "buick.yml")
	raw := `
proxy:
  cert_file: "c.pem"
  key_file: "k.pem"
services:
  a.localhost:
    target: "http://127.0.0.1:1"
`
	if err := writeFile(path, raw); err != nil {
		t.Fatal(err)
	}
	root, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := Validate(root); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if root.Proxy.HTTP != defaultListenHTTP || root.Proxy.HTTPS != defaultListenHTTPS {
		t.Fatalf("proxy: %+v", root.Proxy)
	}
}

func mustParseURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func mustLoadValidateYAML(t *testing.T, raw string) []Resolved {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "buick.yml")
	if err := writeFile(path, raw); err != nil {
		t.Fatal(err)
	}
	root, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	routes, err := Validate(root)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	return routes
}

func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}
