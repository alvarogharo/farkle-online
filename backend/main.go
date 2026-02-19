package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Permite conexiones desde cualquier origen (solo para desarrollo)
	},
}

func handleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error al hacer upgrade:", err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	client.hub.register <- client

	go client.writePump()
	client.readPump()
}

func main() {
	hub := newHub()
	go hub.run()
	go hub.cleanupFinishedGames()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})
	log.Println("Servidor WebSocket escuchando en ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
