package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindInitConfigFileGivenNoFileReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	if _, ok := findInitConfigFile(dir); ok {
		t.Fatal("expected no file")
	}
}

func TestFindInitConfigFileGivenBuickYmlReturnsIt(t *testing.T) {
	dir := t.TempDir()
	b := filepath.Join(dir, "buick.yml")
	if err := os.WriteFile(b, []byte("proxy: {}\nservices: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, ok := findInitConfigFile(dir)
	want := filepath.Clean(b)
	if !ok || got != want {
		t.Fatalf("got %q want %q ok=%v", got, want, ok)
	}
}
