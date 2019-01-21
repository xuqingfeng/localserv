package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/handlers"
)

func main() {

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
		log.Fatal(http.ListenAndServeTLS(*host+":"+strconv.Itoa(*port), *ca, *key, handlers.LoggingHandler(os.Stdout, http.FileServer(http.Dir(*directory)))))
	} else if len(*ca) == 0 && len(*key) == 0 {
		fmt.Printf("Serving %s at http://%s:%d\n", absolutePath, *host, *port)
		fmt.Println("Ctrl-C to exit.")
		// using http
		log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), handlers.LoggingHandler(os.Stdout, http.FileServer(http.Dir(*directory)))))
	} else {
		log.Fatal("E! CA and key file must be used together")
	}
}
