//go:build linux

package trust

import (
	"fmt"
	"os"
)

const linuxCACertDest = "/usr/local/share/ca-certificates/buick-root-ca.crt"

func installLinux(src string) error {
	if _, err := os.Stat("/usr/local/share/ca-certificates"); err != nil {
		return fmt.Errorf("linux: %w (use -skip-trust and install %s via your distro docs)", err, src)
	}
	if err := run("sudo", "cp", src, linuxCACertDest); err != nil {
		return fmt.Errorf("linux install: %w (use -skip-trust if you cannot use sudo here)", err)
	}
	return run("sudo", "update-ca-certificates")
}

func uninstallLinux() error {
	_ = run("sudo", "rm", "-f", linuxCACertDest)
	return run("sudo", "update-ca-certificates")
}
