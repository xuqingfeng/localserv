package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"log/slog"
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
	ca := flag.String(
		"ca", "", "CA file (ca.pem)",
	)
	key := flag.String(
		"key", "", "key file (key.pem)",
	)

	flag.Parse()

	absolutePath, err := filepath.Abs(*directory)
	if err != nil {
		absolutePath = *directory
	}

	if len(*ca) > 0 && len(*key) > 0 {
		// read DNS name in ca
		certPem, err := ioutil.ReadFile(filepath.Join(absolutePath, *ca))
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
		// FIXME
		fmt.Printf("Serving %s at https://%s:%d\n", absolutePath, cert.DNSNames[0], *port)
		fmt.Println("Ctrl-C to exit.")
		// using https
		log.Fatal(http.ListenAndServeTLS(*host+":"+strconv.Itoa(*port), *ca, *key, loggingMiddleware(http.FileServer(http.Dir(*directory)))))
	} else if len(*ca) == 0 && len(*key) == 0 {
		fmt.Printf("Serving %s at http://%s:%d\n", absolutePath, *host, *port)
		fmt.Println("Ctrl-C to exit.")
		// using http
		log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), loggingMiddleware(http.FileServer(http.Dir(*directory)))))
	} else {
		log.Fatal("E! CA and key file must be used together")
	}
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
