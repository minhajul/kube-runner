// Package main implements a tiny HTTP server whose /healthz endpoint
// satisfies the verification contract defined in ADR-0007:
//   - HTTP 200
//   - body equals the literal string "ok"
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// rootHandler responds to GET / with a static hello page.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello from kube-runner")
}

// healthzHandler responds to GET /healthz with the verification contract.
// Anything other than this exact response is a deployment failure
// (see ADR-0007).
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/healthz", healthzHandler)

	log.Printf("kube-runner app listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}