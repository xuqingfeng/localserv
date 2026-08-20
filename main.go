package main

import (
	"flag"
	"log/slog"
	"os"
)

// config holds the command-line configuration for the server.
type config struct {
	host      string
	port      int
	directory string
	certFile  string
	keyFile   string
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg := parseFlags()
	if err := run(cfg); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.host, "host", "127.0.0.1", "host (127.0.0.1 / 0.0.0.0)")
	flag.IntVar(&cfg.port, "port", 8000, "port")
	flag.StringVar(&cfg.directory, "dir", "./", "directory")
	flag.StringVar(&cfg.certFile, "cert", "", "cert file (cert.pem)")
	flag.StringVar(&cfg.keyFile, "key", "", "key file (key.pem)")
	flag.Parse()
	return cfg
}
