package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseFlags(t *testing.T) {
	cfg := parseFlags([]string{"-host", "0.0.0.0", "-port", "9000", "-dir", "/tmp/www", "-cert", "cert.pem", "-key", "key.pem"})

	if cfg.host != "0.0.0.0" {
		t.Errorf("host = %q, want %q", cfg.host, "0.0.0.0")
	}
	if cfg.port != 9000 {
		t.Errorf("port = %d, want %d", cfg.port, 9000)
	}
	if cfg.directory != "/tmp/www" {
		t.Errorf("directory = %q, want %q", cfg.directory, "/tmp/www")
	}
	if cfg.certFile != "cert.pem" {
		t.Errorf("certFile = %q, want %q", cfg.certFile, "cert.pem")
	}
	if cfg.keyFile != "key.pem" {
		t.Errorf("keyFile = %q, want %q", cfg.keyFile, "key.pem")
	}
}

func TestParseFlagsDefaults(t *testing.T) {
	cfg := parseFlags(nil)

	if cfg.host != "127.0.0.1" {
		t.Errorf("host = %q, want %q", cfg.host, "127.0.0.1")
	}
	if cfg.port != 8000 {
		t.Errorf("port = %d, want %d", cfg.port, 8000)
	}
	if cfg.directory != "./" {
		t.Errorf("directory = %q, want %q", cfg.directory, "./")
	}
	if cfg.certFile != "" || cfg.keyFile != "" {
		t.Errorf("certFile/keyFile = %q/%q, want empty", cfg.certFile, cfg.keyFile)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config
		wantErr bool
	}{
		{"both set", config{certFile: "cert.pem", keyFile: "key.pem"}, false},
		{"neither set", config{}, false},
		{"only cert", config{certFile: "cert.pem"}, true},
		{"only key", config{keyFile: "key.pem"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// writeTestCert writes a self-signed certificate and its key to dir and
// returns their paths.
func writeTestCert(t *testing.T, dir string, dnsNames []string, ips []net.IP) (string, string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}

func TestCertName(t *testing.T) {
	dir := t.TempDir()

	t.Run("dns name", func(t *testing.T) {
		certPath, _ := writeTestCert(t, dir, []string{"localhost", "example.com"}, nil)
		name, err := certName(certPath)
		if err != nil {
			t.Fatal(err)
		}
		if name != "localhost" {
			t.Errorf("name = %q, want %q", name, "localhost")
		}
	})

	t.Run("ip fallback", func(t *testing.T) {
		certPath, _ := writeTestCert(t, dir, nil, []net.IP{net.ParseIP("127.0.0.1")})
		name, err := certName(certPath)
		if err != nil {
			t.Fatal(err)
		}
		if name != "127.0.0.1" {
			t.Errorf("name = %q, want %q", name, "127.0.0.1")
		}
	})

	t.Run("no names", func(t *testing.T) {
		certPath, _ := writeTestCert(t, dir, nil, nil)
		name, err := certName(certPath)
		if err != nil {
			t.Fatal(err)
		}
		if name != "" {
			t.Errorf("name = %q, want empty", name)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := certName(filepath.Join(dir, "missing.pem")); err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("bad pem", func(t *testing.T) {
		bad := filepath.Join(dir, "bad.pem")
		if err := os.WriteFile(bad, []byte("not a pem"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := certName(bad); err == nil {
			t.Error("expected error for bad PEM")
		}
	})
}

// newTestHandler builds the same middleware + file server chain used by
// run() over a temp directory containing an index.html.
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	return loggingMiddleware(http.FileServer(http.Dir(dir)))
}

func TestHandlerServesFile(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "hello" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "hello")
	}
}

func TestHandlerNotFound(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/missing.txt", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandlerLogsStatus(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(old)

	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/missing.txt", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := buf.String(); !strings.Contains(got, "status=404") {
		t.Errorf("request log = %q, want it to contain status=404", got)
	}
}

// freePort returns a currently free TCP port on the loopback interface.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// waitForServer polls url until the client gets a response or timeout elapses.
func waitForServer(t *testing.T, client *http.Client, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server at %s did not respond within %v", url, timeout)
}

// signalStop sends SIGINT to the current process, which run() handles via
// signal.NotifyContext, and asserts that run returns cleanly.
func signalStop(t *testing.T, done chan error) {
	t.Helper()
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() returned error after shutdown: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not exit after SIGINT")
	}
}

func TestRunServesAndShutsDown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	port := freePort(t)
	cfg := config{host: "127.0.0.1", port: port, directory: dir}
	done := make(chan error, 1)
	go func() { done <- run(cfg) }()

	url := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/"
	client := &http.Client{}
	waitForServer(t, client, url, 5*time.Second)

	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || string(body) != "hi" {
		t.Errorf("GET %s = status %d body %q, want 200 %q", url, resp.StatusCode, body, "hi")
	}

	signalStop(t, done)
}

func TestRunTLS(t *testing.T) {
	// keep the cert outside the served directory, as the README recommends
	certDir := t.TempDir()
	certPath, keyPath := writeTestCert(t, certDir, []string{"localhost"}, nil)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("secure"), 0o644); err != nil {
		t.Fatal(err)
	}

	port := freePort(t)
	cfg := config{host: "127.0.0.1", port: port, directory: dir, certFile: certPath, keyFile: keyPath}
	done := make(chan error, 1)
	go func() { done <- run(cfg) }()

	// InsecureSkipVerify is intentional: the test uses a throwaway self-signed cert.
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	url := "https://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/"
	waitForServer(t, client, url, 5*time.Second)

	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || string(body) != "secure" {
		t.Errorf("GET %s = status %d body %q, want 200 %q", url, resp.StatusCode, body, "secure")
	}

	signalStop(t, done)
}
