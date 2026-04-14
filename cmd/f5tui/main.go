package main

import (
	"flag"
	"fmt"
	"os"

	"f5tui/internal/config"
	"f5tui/internal/f5"
	"f5tui/internal/mock"
	"f5tui/internal/ui"
)

func main() {
	var (
		configPath = flag.String("config", "", "path to YAML config (default: $XDG_CONFIG_HOME/f5tui/config.yaml)")
		host       = flag.String("host", "", "BIG-IP host (e.g. https://bigip.example.com)")
		user       = flag.String("user", "", "BIG-IP username (basic auth)")
		pass       = flag.String("pass", "", "BIG-IP password (basic auth)")
		partition  = flag.String("partition", "", "initial partition filter (empty = all)")
		insecure   = flag.Bool("insecure", false, "skip TLS verification")
		useMock    = flag.Bool("mock", false, "run against an in-process mock BIG-IP")
	)
	flag.Parse()

	path := *configPath
	explicit := path != ""
	if path == "" {
		path = config.DefaultPath()
	}
	cfg, err := config.Load(path, explicit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	merged := *cfg
	if *host != "" {
		merged.Host = *host
	}
	if *user != "" {
		merged.User = *user
	}
	if *pass != "" {
		merged.Pass = *pass
	}
	if *partition != "" {
		merged.Partition = *partition
	}
	if *insecure {
		merged.Insecure = true
	}

	if *useMock {
		srv := mock.Start()
		defer srv.Close()
		merged.Host = srv.URL
		merged.User = "admin"
		merged.Pass = "admin"
		merged.Insecure = true
	}

	if merged.Host == "" {
		fmt.Fprintln(os.Stderr, "host is required (config, --host, or --mock)")
		os.Exit(2)
	}

	client := f5.New(merged.Host, merged.User, merged.Pass, merged.Insecure)
	if err := ui.Run(client, merged.Partition); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
