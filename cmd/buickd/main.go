// Command buickd is the Buick reverse proxy daemon.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fgrzl/buick/internal/buildinfo"
	"github.com/fgrzl/buick/internal/certs"
	"github.com/fgrzl/buick/internal/config"
	"github.com/fgrzl/buick/internal/mgmt"
	"github.com/fgrzl/buick/internal/proxy"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Printf("buickd %s (commit %s, built %s)\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)
		return 0
	}

	var configPath = flag.String("config", "", "path to buick YAML config")
	flag.Parse()

	if strings.TrimSpace(*configPath) == "" {
		fmt.Fprintln(os.Stderr, "buickd: --config is required")
		return 2
	}
	configFile := filepath.Clean(strings.TrimSpace(*configPath))

	root, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buickd: load config: %v\n", err)
		return 1
	}
	routes, err := config.Validate(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buickd: invalid config: %v\n", err)
		return 1
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("starting", "version", buildinfo.Version, "commit", buildinfo.Commit, "build_date", buildinfo.BuildDate)

	if err := ensureTLSMaterial(root, log, true); err != nil {
		fmt.Fprintf(os.Stderr, "buickd: tls: %v\n", err)
		return 1
	}

	httpAddr := strings.TrimSpace(root.Proxy.HTTP)
	httpsAddr := strings.TrimSpace(root.Proxy.HTTPS)

	if httpAddr == "" && httpsAddr == "" {
		fmt.Fprintln(os.Stderr, "buickd: no listeners configured")
		return 1
	}

	startTime := time.Now()
	metrics := mgmt.NewMetrics(config.HostsFromResolved(routes))
	rt := proxy.NewRouter(routes, log, proxy.WithMetrics(metrics))
	handler := mgmt.Wrap(rt, rt, startTime, buildinfo.Version, httpAddr, httpsAddr, metrics)

	readTO, writeTO := config.MaxEffectiveTimeouts(routes)

	var servers []*http.Server

	if httpAddr != "" {
		servers = append(servers, &http.Server{
			Addr:              httpAddr,
			Handler:           handler,
			ReadHeaderTimeout: 30 * time.Second,
			ReadTimeout:       readTO,
			WriteTimeout:      writeTO,
			IdleTimeout:       10 * time.Minute,
		})
	}
	if httpsAddr != "" {
		servers = append(servers, &http.Server{
			Addr:              httpsAddr,
			Handler:           handler,
			ReadHeaderTimeout: 30 * time.Second,
			ReadTimeout:       readTO,
			WriteTimeout:      writeTO,
			IdleTimeout:       10 * time.Minute,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
		})
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sigHup := make(chan os.Signal, 1)
	signal.Notify(sigHup, syscall.SIGHUP)
	go func() {
		defer signal.Stop(sigHup)
		for {
			select {
			case <-ctx.Done():
				return
			case <-sigHup:
				if err := reloadConfig(configFile, log, rt); err != nil {
					log.Warn("reload skipped", "err", err)
				}
			}
		}
	}()

	errCh := make(chan error, len(servers))

	httpIdx, httpsIdx := -1, -1
	n := 0
	if httpAddr != "" {
		httpIdx = n
		n++
	}
	if httpsAddr != "" {
		httpsIdx = n
	}

	if httpIdx >= 0 {
		s := servers[httpIdx]
		go func() {
			log.Info("listening", "proto", "http", "addr", s.Addr)
			if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("http %s: %w", s.Addr, err)
			}
		}()
	}
	if httpsIdx >= 0 {
		s := servers[httpsIdx]
		certPath := filepath.Clean(root.Proxy.CertFile)
		keyPath := filepath.Clean(root.Proxy.KeyFile)
		go func() {
			log.Info("listening", "proto", "https", "addr", s.Addr, "cert", certPath, "key", keyPath)
			if err := s.ListenAndServeTLS(certPath, keyPath); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("https %s: %w", s.Addr, err)
			}
		}()
	}

	select {
	case <-ctx.Done():
		log.Info("shutdown", "reason", ctx.Err())
	case err := <-errCh:
		log.Error("server error", "err", err)
		stop()
		shutdownServers(log, servers)
		return 1
	}

	shutdownServers(log, servers)
	log.Info("stopped")
	return 0
}

func reloadConfig(path string, log *slog.Logger, rt *proxy.Router) error {
	root, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	routes, err := config.Validate(root)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	if err := ensureTLSMaterial(root, log, false); err != nil {
		return fmt.Errorf("tls: %w", err)
	}
	rt.Reload(routes)
	log.Info("config reloaded", "routes", len(routes),
		"note", "listener addresses and server read/write timeouts unchanged until process restart")
	return nil
}

func shutdownServers(log *slog.Logger, servers []*http.Server) {
	shCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, s := range servers {
		if s == nil {
			continue
		}
		if err := s.Shutdown(shCtx); err != nil {
			log.Warn("shutdown", "addr", s.Addr, "err", err)
		}
	}
}

func ensureTLSMaterial(root *config.Root, log *slog.Logger, logExistingMaterial bool) error {
	if strings.TrimSpace(root.Proxy.HTTPS) == "" {
		return nil
	}
	if err := certs.CheckLeafMaterial(root.Proxy.CertFile, root.Proxy.KeyFile); err != nil {
		return err
	}
	if logExistingMaterial {
		log.Info("using tls material", "cert", root.Proxy.CertFile, "key", root.Proxy.KeyFile)
	}
	return nil
}
