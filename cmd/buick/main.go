// Package main is the buick developer CLI (init, check, routes, status).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fgrzl/buick/internal/buildinfo"
	"github.com/fgrzl/buick/internal/config"
)

func main() {
	if hasVersionFlag() {
		fmt.Printf("buick %s (commit %s, built %s)\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)
		os.Exit(0)
	}
	if len(os.Args) >= 2 && os.Args[1] == "init" {
		os.Exit(runInit(os.Args[2:]))
	}
	os.Exit(run())
}

func run() int {
	var (
		configPath  = flag.String("config", "", "path to buick YAML config")
		checkOnly   = flag.Bool("check", false, "validate config and exit")
		printRoutes = flag.Bool("print-routes", false, "print resolved host→target table and exit")
	)
	flag.Parse()

	if strings.TrimSpace(*configPath) == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "buick: working directory: %v\n", err)
			return 2
		}
		chosen, ok := findInitConfigFile(wd)
		if !ok {
			fmt.Fprintf(os.Stderr, "buick: no %s in current directory; use --config\n", defaultInitConfigName)
			return 2
		}
		*configPath = chosen
	} else {
		*configPath = filepath.Clean(strings.TrimSpace(*configPath))
	}

	root, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buick: load config: %v\n", err)
		return 1
	}
	routes, err := config.Validate(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buick: invalid config: %v\n", err)
		return 1
	}

	if *checkOnly {
		fmt.Println("config OK")
		return 0
	}
	if *printRoutes {
		printRouteTable(routes)
		return 0
	}

	fmt.Fprintln(os.Stderr, "buick: nothing to do (use --check, --print-routes, or buick init; run the proxy with buickd)")
	return 2
}

func printRouteTable(routes []config.Resolved) {
	sort.Slice(routes, func(i, j int) bool { return routes[i].Host < routes[j].Host })
	for _, r := range routes {
		tgts := make([]string, len(r.Targets))
		for i, u := range r.Targets {
			tgts[i] = u.String()
		}
		fmt.Printf("%s -> %s\n", r.Host, strings.Join(tgts, ", "))
	}
}

func hasVersionFlag() bool {
	for _, a := range os.Args[1:] {
		if a == "--version" || a == "-version" {
			return true
		}
	}
	return false
}
