//go:build tools
// +build tools

package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
)

func main() {
	var port int
	flag.IntVar(&port, "p", 5555, "listen port")
	flag.Parse()
	http.HandleFunc("/", botHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func botHandler(w http.ResponseWriter, r *http.Request) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	fmt.Printf("%s\n", dump)

	w.WriteHeader(http.StatusNoContent)
}
