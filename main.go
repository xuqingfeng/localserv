package main
import (
	"net/http"
	"flag"
	"strconv"
	"fmt"
	"os"
)

func main() {

	var (
		port *int = flag.Int(
			"p", 8888, "port",
		)
		directory *string = flag.String(
			"d", "./", "directory",
		)
	)

	flag.Parse()

	fmt.Printf("Server running at localhost:%d on %s\n", *port, *directory)
	fmt.Println("Ctrl-C to exit.")

	err := http.ListenAndServe(":" + strconv.Itoa(*port), http.FileServer(http.Dir(*directory)))
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
