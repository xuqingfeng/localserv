# localserv

A minimal static file server for local development, written in Go. It serves a
directory over HTTP, or over HTTPS with a certificate and key.

## Usage

```
Usage of localserv:
  -cert string
    	cert file, relative to -dir (cert.pem)
  -dir string
    	directory (default "./")
  -host string
    	host (127.0.0.1 / 0.0.0.0) (default "127.0.0.1")
  -key string
    	key file, relative to -dir (key.pem)
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

The startup banner shows the certificate's first DNS name or IP address. Cert
and key paths are resolved relative to `-dir`. HTTPS requires TLS 1.2 or newer.
