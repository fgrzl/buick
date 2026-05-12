package certs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldGenerateDevPairGivenMissingCertFilesWhenEnsureDevPairCalled(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "cert.pem")
	key := filepath.Join(dir, "key.pem")

	gen, err := EnsureDevPair(cert, key, []string{"dev.test", "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	if !gen {
		t.Fatal("expected generated=true on first run")
	}
	if _, err := os.Stat(cert); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(key); err != nil {
		t.Fatal(err)
	}
}

func TestShouldReuseExistingPairGivenCertFilesAlreadyExistWhenEnsureDevPairCalled(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "cert.pem")
	key := filepath.Join(dir, "key.pem")

	if _, err := EnsureDevPair(cert, key, []string{"dev.test", "127.0.0.1"}); err != nil {
		t.Fatal(err)
	}

	gen2, err := EnsureDevPair(cert, key, []string{"other.test"})
	if err != nil {
		t.Fatal(err)
	}
	if gen2 {
		t.Fatal("expected generated=false when files exist")
	}
}
