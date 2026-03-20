package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := flag.Int("port", 8081, "Port to run the backend server on")
	name := flag.String("name", "backend-1", "Name identifier for this backend")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] Request from %s: %s %s", *name, r.RemoteAddr, r.Method, r.URL.Path)
		w.Header().Set("X-Backend-Server", *name)
		fmt.Fprintf(w, "Hello from %s!\nRequest path: %s\nSession cookie: %v\n", *name, r.URL.Path, r.Cookies())
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	log.Printf("Starting backend server %s on port %d", *name, *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatalf("Failed to start backend server: %v", err)
	}
}
