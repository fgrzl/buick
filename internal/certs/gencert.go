// Package certs generates development TLS key pairs.
package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// EnsureDevPair creates certFile/keyFile if either is missing, using dnsNames for SANs.
// It returns generated=true when a new key pair was written.
func EnsureDevPair(certFile, keyFile string, dnsNames []string) (generated bool, err error) {
	certFile = filepath.Clean(certFile)
	keyFile = filepath.Clean(keyFile)

	needGen := false
	if _, err := os.Stat(certFile); err != nil {
		needGen = true
	}
	if _, err := os.Stat(keyFile); err != nil {
		needGen = true
	}
	if !needGen {
		return false, nil
	}

	cdir := filepath.Dir(certFile)
	if cdir != "" && cdir != "." {
		if err := os.MkdirAll(cdir, 0o750); err != nil {
			return false, fmt.Errorf("mkdir cert dir: %w", err)
		}
	}
	kdir := filepath.Dir(keyFile)
	if kdir != "" && kdir != "." && kdir != cdir {
		if err := os.MkdirAll(kdir, 0o750); err != nil {
			return false, fmt.Errorf("mkdir key dir: %w", err)
		}
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return false, fmt.Errorf("generate rsa key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return false, fmt.Errorf("serial: %w", err)
	}

	dnsSet := map[string]struct{}{}
	ipSet := map[string]struct{}{}
	for _, name := range dnsNames {
		name = trimHost(name)
		if name == "" {
			continue
		}
		if ip := net.ParseIP(name); ip != nil {
			ipSet[ip.String()] = struct{}{}
			continue
		}
		dnsSet[name] = struct{}{}
	}
	var dns []string
	for d := range dnsSet {
		dns = append(dns, d)
	}
	var ips []net.IP
	for i := range ipSet {
		ips = append(ips, net.ParseIP(i))
	}

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"buick dev"},
			CommonName:   pickCN(dns, ips),
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dns,
		IPAddresses:           ips,
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return false, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		return false, fmt.Errorf("write cert: %w", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		return false, fmt.Errorf("write key: %w", err)
	}
	return true, nil
}

func trimHost(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
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
