package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"server/internal/server"
	clients "server/internal/server/Clients"
)

var (
	port = flag.Int("port", 8080, "port to listen on")
)

func main() {
	flag.Parse()

	// Define the game hub
	hub := server.NewHub()

	// Define handler for WebSocket connections
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.Server(clients.NewWebSocketClient, w, r)
	})

	go hub.Run()

	addr := fmt.Sprintf(":%d", *port)

	log.Printf("Starting server on %s", addr)

	err := http.ListenAndServe(addr, nil)

	if err != nil {
		log.Fatalf("ListenAndServer %v", err)
	}
}
