package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultInitConfigPath(t *testing.T) {
	dir := t.TempDir()
	if _, ok := defaultInitConfigPath(dir); ok {
		t.Fatal("expected no file")
	}
	b := filepath.Join(dir, "buick.yml")
	if err := os.WriteFile(b, []byte("proxy: {}\nservices: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, ok := defaultInitConfigPath(dir)
	want := filepath.Clean(b)
	if !ok || got != want {
		t.Fatalf("got %q want %q ok=%v", got, want, ok)
	}
}
