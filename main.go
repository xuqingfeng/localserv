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
            "p", 8000, "port",
        )
        directory *string = flag.String(
            "d", "./", "directory",
        )
    )

    flag.Parse()

    fmt.Printf("Server running at 127.0.0.1:%d on %s\n", *port, *directory)
    fmt.Println("Ctrl-C to exit.")

    err := http.ListenAndServe(":" + strconv.Itoa(*port), http.FileServer(http.Dir(*directory)))
    if err != nil {
        fmt.Printf("Error: %v", err)
        os.Exit(1)
    }
}
