package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
		send: make(chan []byte, Cfg.SendBufferSize),
	}
	client.hub.register <- client

	go client.writePump()
	client.readPump()
}

func main() {
	hub := newHub()
	go hub.run()
	go hub.cleanupFinishedGames()

	addr := ":" + Cfg.Port
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})
	log.Println("Servidor WebSocket escuchando en", fmt.Sprintf("ws://localhost%s/ws", addr))
	log.Fatal(http.ListenAndServe(addr, nil))
}
