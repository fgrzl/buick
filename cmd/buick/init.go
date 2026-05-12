package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fgrzl/buick/internal/config"
	"github.com/fgrzl/buick/internal/initca"
)

const defaultInitConfigName = "buick.yml"

// findInitConfigFile returns ./buick.yml under wd when it exists as a regular file.
func findInitConfigFile(wd string) (abs string, ok bool) {
	p := filepath.Join(wd, defaultInitConfigName)
	st, err := os.Stat(p)
	if err == nil && !st.IsDir() {
		return filepath.Clean(p), true
	}
	return "", false
}

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: buick init [flags]\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Generate a local Buick CA, issue a leaf TLS certificate for hostnames in the\n")
		_, _ = fmt.Fprintf(fs.Output(), "config, write PEMs under certs.path (see buick.yml), and install the CA\n")
		_, _ = fmt.Fprintf(fs.Output(), "into this machine's trust store where supported (Windows / macOS login keychain /\n")
		_, _ = fmt.Fprintf(fs.Output(), "Debian-like Linux with sudo). Firefox NSS is not modified; use -skip-trust to write PEMs only.\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Run on the host before docker compose up so browsers trust HTTPS to *.localhost.\n\n")
		fs.PrintDefaults()
	}

	configPath := fs.String("config", "", "path to buick YAML (if omitted: ./"+defaultInitConfigName+")")
	uninstall := fs.Bool("uninstall", false, "remove the Buick CA from trust stores (uses buick-root-ca.pem next to the leaf cert)")
	skipTrust := fs.Bool("skip-trust", false, "write PEM files only; do not install the CA")
	noFirefox := fs.Bool("no-firefox", false, "no-op (reserved); Firefox NSS is never modified by buick init")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	chosen := strings.TrimSpace(*configPath)
	if chosen == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "buick init: working directory: %v\n", err)
			return 2
		}
		var ok bool
		chosen, ok = findInitConfigFile(wd)
		if !ok {
			fmt.Fprintf(os.Stderr, "buick init: no %s in current directory; use --config\n", defaultInitConfigName)
			fs.Usage()
			return 2
		}
	}
	chosen = filepath.Clean(chosen)

	root, err := config.Load(chosen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buick init: load config: %v\n", err)
		return 1
	}
	if _, err := config.Validate(root); err != nil {
		fmt.Fprintf(os.Stderr, "buick init: invalid config: %v\n", err)
		return 1
	}

	opts := initca.Options{
		SkipTrust: *skipTrust,
		NoFirefox: *noFirefox,
	}

	if *uninstall {
		if err := initca.Uninstall(root, opts); err != nil {
			fmt.Fprintf(os.Stderr, "buick init: uninstall: %v\n", err)
			return 1
		}
		fmt.Println("Buick CA removed from trust stores (PEM files on disk were not deleted).")
		return 0
	}

	if err := initca.GenerateAndTrust(root, opts); err != nil {
		fmt.Fprintf(os.Stderr, "buick init: %v\n", err)
		return 1
	}

	fmt.Println("Buick TLS material written and CA installed for local development.")
	leafCert, _, err := config.InitWritePEMPaths(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buick init: %v\n", err)
		return 1
	}
	fmt.Printf("CA certificate (backup / audit): %s\n", filepath.Join(filepath.Dir(filepath.Clean(leafCert)), initca.RootCAFileName))
	return 0
}
