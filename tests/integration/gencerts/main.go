// Command gencerts (maintainers) writes fixture PEMs under tests/integration/certs.
//
//	go run ./tests/integration/gencerts
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fgrzl/buick/internal/certs"
)

func main() {
	dir := filepath.Join("tests", "integration", "certs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatal(err)
	}
	cert := filepath.Join(dir, "localhost.pem")
	key := filepath.Join(dir, "localhost-key.pem")
	_ = os.Remove(cert)
	_ = os.Remove(key)
	names := []string{
		"localhost", "127.0.0.1", "::1",
		"service1.localhost", "service2.localhost", "service3.localhost",
	}
	gen, err := certs.EnsureDevPair(cert, key, names)
	if err != nil {
		log.Fatal(err)
	}
	if !gen {
		log.Fatal("expected new PEMs")
	}
	log.Printf("wrote %s and %s", cert, key)
}
