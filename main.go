package main

import (
	"net/http"
	"flag"
	"strconv"
	"fmt"
	"os"
	"bitbucket.org/xuqingfeng/localserv/vendor/github.com/abbot/go-http-auth"
)

func main() {

	var (
		err error
		port *int = flag.Int(
			"p", 8888, "port",
		)
		directory *string = flag.String(
			"d", "./", "directory",
		)
		basicAuth *bool = flag.Bool(
			"a", false, "basic auth",
		)
	)

	flag.Parse()

	fmt.Printf("Server running at localhost:%d on %s\n", *port, *directory)
	fmt.Println("Ctrl-C to exit.")

	if (!basicAuth) {
		err = http.ListenAndServe(":" + strconv.Itoa(*port), http.FileServer(http.Dir(*directory)))
		if err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
	}else {
		authenticator := auth.NewBasicAuthenticator("localhost", secret)
		http.HandleFunc("/", authenticator.Wrap(func() http.Handler {
			return http.FileServer(http.Dir(*directory))
		})())
		err = http.ListenAndServe(":" + strconv.Itoa(*port), http.FileServer(http.Dir(*directory)))
	}

}

func secret(user, realm string) string {

	if ("xqf" == user) {
		return "localserv"
	}
	return ""
}
