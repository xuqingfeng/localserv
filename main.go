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

	cfg := parseFlags(os.Args[1:])
	if err := run(cfg); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func parseFlags(args []string) config {
	var cfg config
	fs := flag.NewFlagSet("localserv", flag.ExitOnError)
	fs.StringVar(&cfg.host, "host", "127.0.0.1", "host (127.0.0.1 / 0.0.0.0)")
	fs.IntVar(&cfg.port, "port", 8000, "port")
	fs.StringVar(&cfg.directory, "dir", "./", "directory")
	fs.StringVar(&cfg.certFile, "cert", "", "cert file, relative to -dir (cert.pem)")
	fs.StringVar(&cfg.keyFile, "key", "", "key file, relative to -dir (key.pem)")
	fs.Parse(args)
	return cfg
}
