# localserv

[![CI](https://github.com/xuqingfeng/localserv/actions/workflows/ci.yml/badge.svg)](https://github.com/xuqingfeng/localserv/actions/workflows/ci.yml)

A minimal static file server for local development, written in Go. It serves a
directory over HTTP, or over HTTPS with a certificate and key.

## Install

### From source (requires Go 1.26+)

```
go install github.com/xuqingfeng/localserv@latest
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`.

### From GitHub Releases

Download the binary for your platform from the
[latest release](https://github.com/xuqingfeng/localserv/releases/latest). For
example, on macOS (Apple Silicon):

```
curl -sSL -o localserv \
  https://github.com/xuqingfeng/localserv/releases/latest/download/localserv_darwin_arm64
chmod +x localserv
sudo mv localserv /usr/local/bin/
```

## Usage

```
Usage of localserv:
  -cert string
    	cert file (cert.pem)
  -dir string
    	directory (default "./")
  -host string
    	host (127.0.0.1 / 0.0.0.0) (default "127.0.0.1")
  -key string
    	key file (key.pem)
  -port int
    	port (default 8000)
```

## Examples

Serve the current directory over HTTP:

```
localserv -dir . -port 8000
```

Serve over HTTPS with a self-signed certificate:

```
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
localserv -dir . -cert cert.pem -key key.pem -port 8443
```
