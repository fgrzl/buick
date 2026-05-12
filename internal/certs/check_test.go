package certs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckLeafMaterialGivenBothFilesExistReturnsNil(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "c.pem")
	key := filepath.Join(dir, "k.pem")
	if err := os.WriteFile(cert, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("y"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckLeafMaterial(cert, key); err != nil {
		t.Fatal(err)
	}
}

func TestCheckLeafMaterialGivenMissingCertReturnsError(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "k.pem")
	if err := os.WriteFile(key, []byte("y"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckLeafMaterial(filepath.Join(dir, "missing.pem"), key); err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckLeafMaterialGivenPathIsDirectoryReturnsError(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "c.pem")
	if err := os.Mkdir(certDir, 0o755); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(dir, "k.pem")
	if err := os.WriteFile(key, []byte("y"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckLeafMaterial(certDir, key); err == nil {
		t.Fatal("expected error")
	}
}
