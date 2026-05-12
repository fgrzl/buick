//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("BUICK_INTEGRATION") == "" {
		os.Exit(m.Run())
	}
	if err := waitForBuickStack(90 * time.Second); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "integration: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// waitForBuickStack polls until buickd accepts HTTP for a routed host (covers CI right after docker compose up).
func waitForBuickStack(timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	base := httpAddr()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/", nil)
		if err != nil {
			cancel()
			lastErr = err
			time.Sleep(time.Second)
			continue
		}
		req.Host = "service1.localhost"
		res, err := client.Do(req)
		cancel()
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if res.StatusCode == http.StatusOK && strings.Contains(string(body), "service1") {
			return nil
		}
		lastErr = fmt.Errorf("status %d", res.StatusCode)
		time.Sleep(time.Second)
	}
	if lastErr != nil {
		return fmt.Errorf("timed out after %v waiting for buickd at %s (run: docker compose up -d --build from repo root): %w", timeout, base, lastErr)
	}
	return fmt.Errorf("timed out after %v waiting for buickd at %s", timeout, base)
}

func httpAddr() string {
	if v := os.Getenv("BUICK_HTTP_ADDR"); v != "" {
		return v
	}
	return "http://127.0.0.1:18080"
}

func httpsAddr() string {
	if v := os.Getenv("BUICK_HTTPS_ADDR"); v != "" {
		return v
	}
	return "https://127.0.0.1:18443"
}

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("BUICK_INTEGRATION") == "" {
		t.Skip("set BUICK_INTEGRATION=1 with docker compose up -d --build from the repo root")
	}
}

func get(t *testing.T, client *http.Client, url, host string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(body)
}

func TestShouldReturnService1EchoGivenService1HostWhenHTTPGetThroughBuick(t *testing.T) {
	requireIntegration(t)
	client := &http.Client{Timeout: 10 * time.Second}
	code, body := get(t, client, httpAddr()+"/", "service1.localhost")
	if code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", code, body)
	}
	if !strings.Contains(body, "service1") {
		t.Fatalf("body %q does not contain service1", body)
	}
}

func TestShouldReturnService2EchoGivenService2HostWhenHTTPGetThroughBuick(t *testing.T) {
	requireIntegration(t)
	client := &http.Client{Timeout: 10 * time.Second}
	code, body := get(t, client, httpAddr()+"/", "service2.localhost")
	if code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", code, body)
	}
	if !strings.Contains(body, "service2") {
		t.Fatalf("body %q does not contain service2", body)
	}
}

func TestShouldReturnService3EchoGivenService3HostWhenHTTPGetThroughBuick(t *testing.T) {
	requireIntegration(t)
	client := &http.Client{Timeout: 10 * time.Second}
	code, body := get(t, client, httpAddr()+"/", "service3.localhost")
	if code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", code, body)
	}
	if !strings.Contains(body, "service3") {
		t.Fatalf("body %q does not contain service3", body)
	}
}

func TestShouldReturn502GivenUnknownHostWhenHTTPGetThroughBuick(t *testing.T) {
	requireIntegration(t)
	client := &http.Client{Timeout: 10 * time.Second}
	code, _ := get(t, client, httpAddr()+"/", "unknown.localhost")
	if code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", code)
	}
}

func TestShouldReturnService1EchoGivenService1HostWhenHTTPSGetThroughBuick(t *testing.T) {
	requireIntegration(t)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	code, body := get(t, client, httpsAddr()+"/", "service1.localhost")
	if code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", code, body)
	}
	if !strings.Contains(body, "service1") {
		t.Fatalf("body %q", body)
	}
}
