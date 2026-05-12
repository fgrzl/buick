package initca

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fgrzl/buick/internal/config"
)

func TestGenerateAndTrustCreatesParentDirectoriesGivenNestedCertPaths(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "nested", "deep", "leaf.pem")
	key := filepath.Join(dir, "nested", "deep", "leaf-key.pem")
	cfg := &config.Root{
		Proxy: config.Proxy{
			HTTP:     ":0",
			HTTPS:    ":0",
			CertFile: cert,
			KeyFile:  key,
		},
		Services: map[string]config.Service{
			"x.localhost": {Target: "http://127.0.0.1:9"},
		},
	}
	if err := GenerateAndTrust(cfg, Options{SkipTrust: true}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{cert, key, filepath.Join(filepath.Dir(cert), RootCAFileName)} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %q: %v", p, err)
		}
	}
}
