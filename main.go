package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	counts   = make(map[string]int)
	mu       sync.Mutex
	clients  = make(map[*websocket.Conn]bool)
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func main() {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", handleWS)

	// Use Render's PORT env var or fallback to 8080 locally
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Server started on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			delete(clients, conn)
			return
		}

		mu.Lock()
		if string(msg) == "__CLEAR__" {
			counts = make(map[string]int) // Reset leaderboard
			mu.Unlock()
			broadcastCounts()
			continue
		}

		// Normal counting
		counts[string(msg)]++
		mu.Unlock()

		broadcastCounts()
	}
}

func broadcastCounts() {
	mu.Lock()
	defer mu.Unlock()

	for client := range clients {
		err := client.WriteJSON(counts)
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
}
