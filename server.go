package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// run serves the configured directory over HTTP or HTTPS until interrupted,
// returning an error if the configuration is invalid or the server fails.
func run(cfg config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	absolutePath, err := filepath.Abs(cfg.directory)
	if err != nil {
		slog.Warn("failed to resolve absolute path", "dir", cfg.directory, "error", err)
		absolutePath = cfg.directory
	}

	handler := loggingMiddleware(http.FileServer(http.Dir(cfg.directory)))
	addr := net.JoinHostPort(cfg.host, strconv.Itoa(cfg.port))

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	var useTLS bool
	if cfg.certFile != "" {
		// cert and key paths are resolved relative to the working directory,
		// not the served directory
		name, err := certName(cfg.certFile)
		if err != nil {
			return err
		}
		if name == "" {
			name = cfg.host
		}
		server.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		fmt.Printf("Serving %s at https://%s\n", absolutePath, net.JoinHostPort(name, strconv.Itoa(cfg.port)))
		useTLS = true
	} else {
		fmt.Printf("Serving %s at http://%s\n", absolutePath, addr)
	}
	fmt.Println("Ctrl-C to exit.")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if useTLS {
			errCh <- server.ListenAndServeTLS(cfg.certFile, cfg.keyFile)
		} else {
			errCh <- server.ListenAndServe()
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

// validateConfig checks that the certificate and key flags are either
// both set or both empty.
func validateConfig(cfg config) error {
	if (cfg.certFile == "") != (cfg.keyFile == "") {
		return errors.New("cert and key file must be used together")
	}
	return nil
}

// certName reads and parses the certificate at path and returns its first
// DNS name or IP address, or "" if it has neither.
func certName(path string) (string, error) {
	certPem, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read cert failed: %w", err)
	}
	block, _ := pem.Decode(certPem)
	if block == nil {
		return "", errors.New("fail to parse cert PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("fail to parse cert: %w", err)
	}
	return certDisplayName(cert), nil
}

// certDisplayName returns the first DNS name or IP address of the
// certificate, or "" if the certificate has neither.
func certDisplayName(cert *x509.Certificate) string {
	if len(cert.DNSNames) > 0 {
		return cert.DNSNames[0]
	}
	if len(cert.IPAddresses) > 0 {
		return cert.IPAddresses[0].String()
	}
	return ""
}

// statusWriter records the response status code for the request log. The
// default is 200 because a handler may respond without calling WriteHeader.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// loggingMiddleware logs each request (method, path, status, duration)
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.RequestURI(),
			"status", sw.status,
			"duration", time.Since(start),
		)
	})
}
