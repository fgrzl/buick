// Package initca creates a local CA and leaf certificates for Buick TLS.
package initca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/smallstep/truststore"

	"github.com/fgrzl/buick/internal/config"
)

const (
	// RootCAFileName is the PEM filename written next to the leaf certificate.
	RootCAFileName = "buick-root-ca.pem"
	rootCAKeyFile  = "buick-root-ca-key.pem"
)

// Options controls trust installation for GenerateAndTrust.
type Options struct {
	SkipTrust bool // only write PEM files
	NoFirefox bool // omit Firefox/NSS trust store
}

// GenerateAndTrust creates a local CA, issues a leaf TLS certificate for the
// hostnames derived from cfg, writes leaf PEMs to cfg's cert_file/key_file,
// writes the CA next to the leaf cert, and installs the CA into system (and
// optionally Firefox) trust stores.
func GenerateAndTrust(cfg *config.Root, o Options) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	certPath := filepath.Clean(strings.TrimSpace(cfg.Proxy.CertFile))
	keyPath := filepath.Clean(strings.TrimSpace(cfg.Proxy.KeyFile))
	if certPath == "" || keyPath == "" {
		return errors.New("proxy.cert_file and proxy.key_file are required")
	}

	caDir := filepath.Dir(certPath)
	if caDir == "" || caDir == "." {
		caDir = "."
	}
	caCertPath := filepath.Join(caDir, RootCAFileName)
	caKeyPath := filepath.Join(caDir, rootCAKeyFile)

	if err := uninstallExistingCA(caCertPath, o); err != nil {
		return err
	}

	names := config.HostnamesForCert(cfg)
	dns, ips := splitNames(names)

	caKey, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return fmt.Errorf("generate CA key: %w", err)
	}
	caSerial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("CA serial: %w", err)
	}
	caTmpl := x509.Certificate{
		SerialNumber:          caSerial,
		Subject:               pkix.Name{Organization: []string{"Buick"}, CommonName: "Buick Development CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, &caTmpl, &caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create CA cert: %w", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(caCertPath), 0o755); err != nil && filepath.Dir(caCertPath) != "." {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := writePEM(caCertPath, "CERTIFICATE", caDER, 0o644); err != nil {
		return err
	}
	if err := writePEM(caKeyPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(caKey), 0o600); err != nil {
		return err
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate leaf key: %w", err)
	}
	leafSerial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("leaf serial: %w", err)
	}
	leafTmpl := x509.Certificate{
		SerialNumber:          leafSerial,
		Subject:               pkix.Name{Organization: []string{"Buick"}, CommonName: pickCN(dns, ips)},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(825 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dns,
		IPAddresses:           ips,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, &leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create leaf cert: %w", err)
	}

	cdir := filepath.Dir(certPath)
	if cdir != "" && cdir != "." {
		if err := os.MkdirAll(cdir, 0o755); err != nil {
			return fmt.Errorf("mkdir leaf dir: %w", err)
		}
	}
	kdir := filepath.Dir(keyPath)
	if kdir != "" && kdir != "." && kdir != cdir {
		if err := os.MkdirAll(kdir, 0o755); err != nil {
			return fmt.Errorf("mkdir key dir: %w", err)
		}
	}
	if err := writePEM(certPath, "CERTIFICATE", leafDER, 0o644); err != nil {
		return err
	}
	if err := writePEM(keyPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(leafKey), 0o600); err != nil {
		return err
	}

	if o.SkipTrust {
		return nil
	}

	var opts []truststore.Option
	if !o.NoFirefox {
		opts = append(opts, truststore.WithFirefox())
	}
	if err := truststore.Install(caCert, opts...); err != nil {
		return fmt.Errorf("install CA in trust stores: %w (leaf PEMs were written; you can retry or use -skip-trust)", err)
	}
	return nil
}

// Uninstall removes a previously installed Buick CA from trust stores using
// buick-root-ca.pem next to the configured leaf cert.
func Uninstall(cfg *config.Root, o Options) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	certPath := filepath.Clean(strings.TrimSpace(cfg.Proxy.CertFile))
	if certPath == "" {
		return errors.New("proxy.cert_file is required")
	}
	caCertPath := filepath.Join(filepath.Dir(certPath), RootCAFileName)
	return uninstallCAFile(caCertPath, o)
}

func uninstallExistingCA(caCertPath string, o Options) error {
	if _, err := os.Stat(caCertPath); err != nil {
		return nil
	}
	return uninstallCAFile(caCertPath, o)
}

func uninstallCAFile(caCertPath string, o Options) error {
	data, err := os.ReadFile(caCertPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("invalid CA PEM at " + caCertPath)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	var opts []truststore.Option
	if !o.NoFirefox {
		opts = append(opts, truststore.WithFirefox())
	}
	if err := truststore.Uninstall(cert, opts...); err != nil {
		return fmt.Errorf("uninstall CA: %w", err)
	}
	return nil
}

func writePEM(path, blockType string, der []byte, mode os.FileMode) error {
	b := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	return os.WriteFile(path, b, mode)
}

func splitNames(names []string) (dns []string, ips []net.IP) {
	setD := map[string]struct{}{}
	ipSet := map[string]net.IP{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if ip := net.ParseIP(name); ip != nil {
			ipSet[ip.String()] = ip
			continue
		}
		setD[name] = struct{}{}
	}
	for d := range setD {
		dns = append(dns, d)
	}
	for _, ip := range ipSet {
		ips = append(ips, ip)
	}
	return dns, ips
}

func pickCN(dns []string, ips []net.IP) string {
	for _, d := range dns {
		if d != "" {
			return d
		}
	}
	if len(ips) > 0 {
		return ips[0].String()
	}
	return "localhost"
}
