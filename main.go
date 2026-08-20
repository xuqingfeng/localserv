package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	host := flag.String(
		"host", "127.0.0.1", "host (127.0.0.1 / 0.0.0.0)",
	)
	port := flag.Int(
		"port", 8000, "port",
	)
	directory := flag.String(
		"dir", "./", "directory",
	)
	certFile := flag.String(
		"cert", "", "cert file (cert.pem)",
	)
	key := flag.String(
		"key", "", "key file (key.pem)",
	)

	flag.Parse()

	absolutePath, err := filepath.Abs(*directory)
	if err != nil {
		absolutePath = *directory
	}

	if len(*certFile) > 0 && len(*key) > 0 {
		// read cert to display its name
		certPem, err := os.ReadFile(filepath.Join(absolutePath, *certFile))
		if err != nil {
			log.Fatalf("E! read cert failed: %v", err)
		}
		block, _ := pem.Decode(certPem)
		if block == nil {
			log.Fatalf("E! fail to parse cert PEM")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Fatalf("E! fail to parse cert: %v", err)
		}
		name := certDisplayName(cert)
		if name == "" {
			name = *host
		}
		fmt.Printf("Serving %s at https://%s\n", absolutePath, net.JoinHostPort(name, strconv.Itoa(*port)))
		fmt.Println("Ctrl-C to exit.")
		// using https
		log.Fatal(http.ListenAndServeTLS(net.JoinHostPort(*host, strconv.Itoa(*port)), *certFile, *key, loggingMiddleware(http.FileServer(http.Dir(*directory)))))
	} else if len(*certFile) == 0 && len(*key) == 0 {
		fmt.Printf("Serving %s at http://%s\n", absolutePath, net.JoinHostPort(*host, strconv.Itoa(*port)))
		fmt.Println("Ctrl-C to exit.")
		// using http
		log.Fatal(http.ListenAndServe(net.JoinHostPort(*host, strconv.Itoa(*port)), loggingMiddleware(http.FileServer(http.Dir(*directory)))))
	} else {
		log.Fatal("E! cert and key file must be used together")
	}
}

// certDisplayName returns the first DNS name or IP address of the
// certificate for display, or "" if the certificate has neither.
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
// replacing the gorilla/handlers LoggingHandler.
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
