//go:build integration

package integration

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

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

func TestBuickHTTPRoutesToEchoBackends(t *testing.T) {
	requireIntegration(t)
	client := &http.Client{Timeout: 10 * time.Second}
	base := httpAddr()

	for _, tc := range []struct {
		host string
		want string
	}{
		{"service1.localhost", "service1"},
		{"service2.localhost", "service2"},
		{"service3.localhost", "service3"},
	} {
		t.Run(tc.host, func(t *testing.T) {
			code, body := get(t, client, base+"/", tc.host)
			if code != http.StatusOK {
				t.Fatalf("status = %d, body = %q", code, body)
			}
			if !strings.Contains(body, tc.want) {
				t.Fatalf("body %q does not contain %q", body, tc.want)
			}
		})
	}
}

func TestBuickHTTPUnknownHost502(t *testing.T) {
	requireIntegration(t)
	client := &http.Client{Timeout: 10 * time.Second}
	code, _ := get(t, client, httpAddr()+"/", "unknown.localhost")
	if code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", code)
	}
}

func TestBuickHTTPSRoutes(t *testing.T) {
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
