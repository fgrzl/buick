package certs

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckLeafMaterial returns nil when certFile and keyFile are existing regular
// files. It does not read or validate PEM contents.
func CheckLeafMaterial(certFile, keyFile string) error {
	certFile = filepath.Clean(certFile)
	keyFile = filepath.Clean(keyFile)
	for _, p := range []struct {
		label, path string
	}{
		{"cert_file", certFile},
		{"key_file", keyFile},
	} {
		fi, err := os.Stat(p.path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s %q: not found (use buick init, mkcert, or your own PEMs)", p.label, p.path)
			}
			return fmt.Errorf("%s %q: %w", p.label, p.path, err)
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%s %q: expected a regular file", p.label, p.path)
		}
	}
	return nil
}
