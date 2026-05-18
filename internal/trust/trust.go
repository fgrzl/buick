package trust

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Options controls optional behavior for Install / Uninstall.
type Options struct {
	// NoFirefox is reserved; Firefox NSS is not modified by this package.
	NoFirefox bool
}

// Install adds the CA PEM at caCertPath to the platform trust store where supported.
func Install(caCertPath string, _ Options) error {
	abs, err := filepath.Abs(caCertPath)
	if err != nil {
		return fmt.Errorf("ca path: %w", err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("ca pem not found: %s", abs)
		}
		return err
	}
	if st.IsDir() {
		return fmt.Errorf("ca path is not a file: %s", abs)
	}
	switch runtime.GOOS {
	case "windows":
		return installWindows(abs)
	case "darwin":
		return installDarwin(abs)
	case "linux":
		return installLinux(abs)
	default:
		return fmt.Errorf("automatic CA install is not implemented on %s; use -skip-trust and import %s manually", runtime.GOOS, abs)
	}
}

// Uninstall removes the CA installed by Install where the platform supports it.
func Uninstall(caCertPath string, _ Options) error {
	abs, err := filepath.Abs(caCertPath)
	if err != nil {
		return fmt.Errorf("ca path: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return uninstallWindows()
	case "darwin":
		return uninstallDarwin()
	case "linux":
		return uninstallLinux()
	default:
		return fmt.Errorf("automatic CA uninstall is not implemented on %s; remove the CA manually from your trust store", runtime.GOOS)
	}
}

