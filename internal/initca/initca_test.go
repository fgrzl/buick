package initca

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fgrzl/buick/internal/config"
)

func TestGenerateAndTrustCreatesParentDirectoriesGivenNestedCertPaths(t *testing.T) {
	dir := t.TempDir()
	writeDir := filepath.Join(dir, "nested", "deep")
	cfg := &config.Root{
		Certs: config.Certs{Path: writeDir},
		Proxy: config.Proxy{
			HTTP:      ":0",
			HTTPS:     ":0",
			CertsPath: writeDir,
		},
		Services: map[string]config.Service{
			"x.localhost": {Target: "http://127.0.0.1:9"},
		},
	}
	if _, err := config.Validate(cfg); err != nil {
		t.Fatal(err)
	}
	if err := GenerateAndTrust(cfg, Options{SkipTrust: true}); err != nil {
		t.Fatal(err)
	}
	cert := filepath.Join(writeDir, "localhost.pem")
	key := filepath.Join(writeDir, "localhost-key.pem")
	for _, p := range []string{cert, key, filepath.Join(filepath.Dir(cert), RootCAFileName)} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %q: %v", p, err)
		}
	}
}

func TestGenerateAndTrustWritesToCertsPathWhenDiffersFromProxyCertsPath(t *testing.T) {
	dir := t.TempDir()
	hostWrite := filepath.Join(dir, "host")
	runRead := filepath.Join(dir, "run")
	cfg := &config.Root{
		Certs: config.Certs{Path: hostWrite},
		Proxy: config.Proxy{
			HTTP:      ":0",
			HTTPS:     ":0",
			CertsPath: runRead,
		},
		Services: map[string]config.Service{
			"x.localhost": {Target: "http://127.0.0.1:9"},
		},
	}
	if _, err := config.Validate(cfg); err != nil {
		t.Fatal(err)
	}
	if err := GenerateAndTrust(cfg, Options{SkipTrust: true}); err != nil {
		t.Fatal(err)
	}
	cert := filepath.Join(hostWrite, "localhost.pem")
	key := filepath.Join(hostWrite, "localhost-key.pem")
	for _, p := range []string{cert, key, filepath.Join(hostWrite, RootCAFileName)} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %q: %v", p, err)
		}
	}
}
