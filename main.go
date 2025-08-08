package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	counts   = make(map[string]int)
	mu       sync.Mutex
	clients  = make(map[*websocket.Conn]bool)
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

func main() {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", handleWS)

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
			// Reset leaderboard
			counts = make(map[string]int)
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
