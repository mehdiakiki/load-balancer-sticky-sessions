package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections for demo
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, serverName string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[%s] WebSocket upgrade error: %v", serverName, err)
		return
	}
	defer conn.Close()

	log.Printf("[%s] WebSocket connection established from %s", serverName, r.RemoteAddr)

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[%s] WebSocket read error: %v", serverName, err)
			break
		}

		log.Printf("[%s] Received: %s", serverName, string(message))

		response := fmt.Sprintf("[%s] Echo: %s", serverName, string(message))
		if err := conn.WriteMessage(messageType, []byte(response)); err != nil {
			log.Printf("[%s] WebSocket write error: %v", serverName, err)
			break
		}
	}

	log.Printf("[%s] WebSocket connection closed", serverName)
}

func main() {
	port := flag.Int("port", 8081, "Port to run the backend server on")
	name := flag.String("name", "backend-1", "Server name identifier")
	flag.Parse()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, *name)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Server", *name)
		fmt.Fprintf(w, "Backend %s is running\n", *name)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	log.Printf("Starting backend server %s on port %d with WebSocket support", *name, *port)
	log.Printf("WebSocket endpoint: ws://localhost:%d/ws", *port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
