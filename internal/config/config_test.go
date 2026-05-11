package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeHost(t *testing.T) {
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

func TestLoadAndValidate(t *testing.T) {
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
	if routes[0].Target.String() != "http://127.0.0.1:9" {
		t.Fatalf("target: got %q", routes[0].Target.String())
	}
}

func TestValidateErrors(t *testing.T) {
	_, err := Validate(&Root{Proxy: Proxy{HTTP: ":80"}, Services: map[string]Service{}})
	if err == nil {
		t.Fatal("expected error for empty services")
	}

	_, err = Validate(&Root{
		Proxy: Proxy{HTTP: ""},
		Services: map[string]Service{
			"a": {Target: "http://x"},
		},
	})
	if err == nil {
		t.Fatal("expected error when no listeners")
	}

	_, err = Validate(&Root{
		Proxy: Proxy{HTTP: ":80", HTTPS: ":443"},
		Services: map[string]Service{
			"a": {Target: "http://x"},
		},
	})
	if err == nil {
		t.Fatal("expected error when https without cert paths")
	}

	_, err = Validate(&Root{
		Proxy: Proxy{HTTP: ":80"},
		Services: map[string]Service{
			"a": {Target: "not-a-url"},
		},
	})
	if err == nil {
		t.Fatal("expected error for bad target URL")
	}

	_, err = Validate(&Root{
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

func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}
