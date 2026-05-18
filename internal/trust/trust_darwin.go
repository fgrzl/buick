//go:build darwin

package trust

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func installDarwin(caPEM string) error {
	abs, err := validatePEMPath(caPEM)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	kc := filepath.Join(home, "Library", "Keychains", "login.keychain-db")
	if _, err := os.Stat(kc); err != nil {
		return fmt.Errorf("login keychain: %w", err)
	}
	cmd := exec.CommandContext(context.Background(), "security", "add-trusted-cert", "-r", "trustRoot", "-k", kc, abs)
	if err := combinedOutput(cmd); err != nil {
		return fmt.Errorf("security add-trusted-cert: %w", err)
	}
	return nil
}

func uninstallDarwin() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	kc := filepath.Join(home, "Library", "Keychains", "login.keychain-db")
	cmd := exec.CommandContext(context.Background(), "security", "delete-certificate", "-c", "Buick Development CA", kc)
	out, err := cmd.CombinedOutput()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 44 {
			return nil
		}
		return fmt.Errorf("security delete-certificate: %w: %s", err, out)
	}
	return nil
}
