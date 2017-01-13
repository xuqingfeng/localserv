package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
)

func main() {

	var (
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

	fmt.Printf("Serving %s at http://127.0.0.1:%d\n", absolutePath, *port)
	fmt.Println("Ctrl-C to exit.")

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), http.FileServer(http.Dir(*directory))))
}
