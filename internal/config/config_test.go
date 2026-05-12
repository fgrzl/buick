package config

import (
	"os"
	"path/filepath"
	"testing"
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
	dir := t.TempDir()
	path := filepath.Join(dir, "buick.yml")
	content := `
proxy:
  http: ":0"

services:
  app.localhost:
    target: "http://127.0.0.1:9"
`
	if err := writeFile(path, content); err != nil {
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

func TestShouldRejectConfigGivenNoListenersWhenValidate(t *testing.T) {
	_, err := Validate(&Root{
		Proxy: Proxy{HTTP: ""},
		Services: map[string]Service{
			"a": {Target: "http://x"},
		},
	})
	if err == nil {
		t.Fatal("expected error when no listeners")
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

func TestShouldParseMultipleTargetsGivenYAMLWithTargetsListWhenLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "buick.yml")
	content := `
proxy:
  http: ":0"

services:
  app.localhost:
    targets:
      - "http://127.0.0.1:1"
      - "http://127.0.0.1:2"
`
	if err := writeFile(path, content); err != nil {
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
	if len(routes) != 1 || len(routes[0].Targets) != 2 {
		t.Fatalf("routes: %+v", routes)
	}
	if routes[0].Targets[0].String() != "http://127.0.0.1:1" || routes[0].Targets[1].String() != "http://127.0.0.1:2" {
		t.Fatalf("targets: %+v", routes[0].Targets)
	}
}

func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}
