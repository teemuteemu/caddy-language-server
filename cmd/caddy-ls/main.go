package main

import (
	"flag"
	"fmt"
	"os"

	"caddy-ls/internal/server"
)

var appVersion = "dev"

func main() {
	var (
		showVersion bool
		logLevel    string
	)

	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.StringVar(&logLevel, "log-level", "warning", "log level: debug, info, warning, error")
	flag.Parse()

	if showVersion {
		fmt.Printf("caddy-ls %s\n", appVersion)
		os.Exit(0)
	}

	if err := server.Run(logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "caddy-ls: %v\n", err)
		os.Exit(1)
	}
}
