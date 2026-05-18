package trust

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func validatePEMPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("invalid path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", fmt.Errorf("path is not a file: %s", abs)
	}
	return abs, nil
}

func combinedOutput(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		_, _ = fmt.Fprint(os.Stderr, string(out))
	}
	return err
}

func certutilAddRoot(caPEM string) error {
	abs, err := validatePEMPath(caPEM)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(context.Background(), "certutil", "-user", "-addstore", "Root", abs)
	if err := combinedOutput(cmd); err != nil {
		return fmt.Errorf("certutil -addstore: %w", err)
	}
	return nil
}

func certutilDelRoot() error {
	cmd := exec.CommandContext(context.Background(), "certutil", "-user", "-delstore", "Root", "Buick Development CA")
	if err := combinedOutput(cmd); err != nil {
		return fmt.Errorf("certutil -delstore: %w", err)
	}
	return nil
}
