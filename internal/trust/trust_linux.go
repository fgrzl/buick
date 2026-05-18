//go:build linux

package trust

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

const linuxCACertDest = "/usr/local/share/ca-certificates/buick-root-ca.crt"

func installLinux(src string) error {
	if _, err := os.Stat("/usr/local/share/ca-certificates"); err != nil {
		return fmt.Errorf("linux: %w (use -skip-trust and install %s via your distro docs)", err, src)
	}
	abs, err := validatePEMPath(src)
	if err != nil {
		return err
	}
	cp := exec.CommandContext(context.Background(), "sudo", "cp", abs, linuxCACertDest)
	if err := combinedOutput(cp); err != nil {
		return fmt.Errorf("linux install: %w (use -skip-trust if you cannot use sudo here)", err)
	}
	upd := exec.CommandContext(context.Background(), "sudo", "update-ca-certificates")
	if err := combinedOutput(upd); err != nil {
		return fmt.Errorf("linux update-ca-certificates: %w", err)
	}
	return nil
}

func uninstallLinux() error {
	rm := exec.CommandContext(context.Background(), "sudo", "rm", "-f", linuxCACertDest)
	_ = combinedOutput(rm)
	upd := exec.CommandContext(context.Background(), "sudo", "update-ca-certificates")
	if err := combinedOutput(upd); err != nil {
		return fmt.Errorf("linux update-ca-certificates: %w", err)
	}
	return nil
}
