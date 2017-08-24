package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/handlers"
)

func main() {

	var (
		host *string = flag.String(
			"h", "127.0.0.1", "host",
		)
		port *int = flag.Int(
			"p", 8000, "port",
		)
		directory *string = flag.String(
			"d", "./", "directory",
		)
	)

	flag.Parse()

	absolutePath, err := filepath.Abs(*directory)
	if err != nil {
		absolutePath = *directory
	}

	fmt.Printf("Serving %s at http://%s:%d\n", absolutePath, *host, *port)
	fmt.Println("Ctrl-C to exit.")

	log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), handlers.LoggingHandler(os.Stdout, http.FileServer(http.Dir(*directory)))))
}
