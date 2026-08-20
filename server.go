package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// run serves the configured directory over HTTP or HTTPS and returns an error
// if the configuration is invalid or the server fails.
func run(cfg config) error {
	absolutePath, err := filepath.Abs(cfg.directory)
	if err != nil {
		absolutePath = cfg.directory
	}

	handler := loggingMiddleware(http.FileServer(http.Dir(cfg.directory)))
	addr := net.JoinHostPort(cfg.host, strconv.Itoa(cfg.port))

	switch {
	case cfg.certFile != "" && cfg.keyFile != "":
		name, err := certName(filepath.Join(absolutePath, cfg.certFile))
		if err != nil {
			return err
		}
		if name == "" {
			name = cfg.host
		}
		fmt.Printf("Serving %s at https://%s\n", absolutePath, net.JoinHostPort(name, strconv.Itoa(cfg.port)))
		fmt.Println("Ctrl-C to exit.")
		return http.ListenAndServeTLS(addr, cfg.certFile, cfg.keyFile, handler)
	case cfg.certFile == "" && cfg.keyFile == "":
		fmt.Printf("Serving %s at http://%s\n", absolutePath, addr)
		fmt.Println("Ctrl-C to exit.")
		return http.ListenAndServe(addr, handler)
	default:
		return errors.New("cert and key file must be used together")
	}
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

// loggingMiddleware logs each request (method, path, status, duration),
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
