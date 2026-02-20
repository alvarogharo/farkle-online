package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type InMessage struct {
	Type         string `json:"type"`
	GameCode     string `json:"gameCode"`
	PlayerName   string `json:"playerName"`
	Values       []int  `json:"values"`
	Index        int    `json:"index"`
	VictoryScore int    `json:"victoryScore"`
}

type Hub struct {
	clients    map[*Client]bool
	games      map[string]*Game
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	gameCode    string
	playerIndex int
}

type Die struct {
	Value int  `json:"value"`
	Held  bool `json:"held"`
}

type TurnMove struct {
	ID     int   `json:"id"`
	Values []int `json:"values"`
	Points int   `json:"points"`
}

const (
	defaultVictoryScore   = 2000
	finishedGameRetention = 5 * time.Minute
	cleanupInterval       = 1 * time.Minute
)

type Game struct {
	code                  string
	clients               []*Client
	playerNames           []string
	totals                []int
	currentPlayerIndex    int
	dice                  []Die
	selectedIndices       []int
	turnPoints            int
	turnMoves             []TurnMove
	hasApartadoThisRoll   bool
	victoryScore          int
	finalRoundTriggerIndex int       // -1 si no ha pasado
	winnerIndex           int        // -1 si la partida sigue
	finishedAt            time.Time  // cuándo terminó la partida
	mu                    sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		games:      make(map[string]*Game),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func generateGameCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 5)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.handleClientDisconnect(client)
				delete(h.clients, client)
				close(client.send)
			}
		}
	}
}

// handleClientDisconnect quita al cliente de su partida y notifica al otro jugador.
func (h *Hub) handleClientDisconnect(client *Client) {
	if client.gameCode == "" {
		return
	}

	h.mu.RLock()
	g, ok := h.games[client.gameCode]
	h.mu.RUnlock()
	if !ok {
		return
	}

	g.mu.Lock()
	if client.playerIndex < 0 || client.playerIndex >= len(g.clients) || g.clients[client.playerIndex] != client {
		g.mu.Unlock()
		return
	}

	g.clients[client.playerIndex] = nil

	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		return
	}

	otherIndex := 1 - client.playerIndex
	other := g.clients[otherIndex]
	if other == nil {
		g.mu.Unlock()
		h.mu.Lock()
		delete(h.games, client.gameCode)
		h.mu.Unlock()
		return
	}

	g.winnerIndex = otherIndex
	g.finishedAt = time.Now()
	gameCode := client.gameCode
	g.mu.Unlock()

	h.broadcastToGame(gameCode, map[string]any{
		"type":    "player_disconnected",
		"message": "El otro jugador se ha desconectado. Ganas la partida.",
		"winner":  otherIndex,
	})
	h.broadcastGameState(gameCode)
}

// cleanupFinishedGames elimina partidas terminadas hace más de finishedGameRetention.
func (h *Hub) cleanupFinishedGames() {
	for range time.Tick(cleanupInterval) {
		h.mu.Lock()
		now := time.Now()
		for code, g := range h.games {
			g.mu.RLock()
			finished := g.winnerIndex >= 0 && !g.finishedAt.IsZero() && now.Sub(g.finishedAt) > finishedGameRetention
			g.mu.RUnlock()
			if finished {
				delete(h.games, code)
				log.Printf("Partida %s eliminada (terminada hace >%v)", code, finishedGameRetention)
			}
		}
		h.mu.Unlock()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg InMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendJSON(map[string]string{"type": "error", "message": "JSON inválido"})
			continue
		}

		switch msg.Type {
		case "ping":
			c.sendJSON(map[string]string{"type": "pong"})
		case "create":
			c.handleCreate(msg)
		case "join":
			c.handleJoin(msg)
		case "roll":
			c.handleRoll()
		case "toggle_select":
			c.handleToggleSelect(msg)
		case "apartar":
			c.handleApartar()
		case "bank":
			c.handleBank()
		default:
			c.sendJSON(map[string]string{"type": "error", "message": "tipo desconocido: " + msg.Type})
		}
	}
}

func (c *Client) writePump() {
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func (c *Client) sendJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		log.Println("Canal lleno, no se pudo enviar")
	}
}

func (c *Client) handleCreate(msg InMessage) {
	code := generateGameCode()
	name := msg.PlayerName
	if name == "" {
		name = "Jugador 1"
	}
	victoryScore := msg.VictoryScore
	if victoryScore < 100 || victoryScore > 100000 {
		victoryScore = defaultVictoryScore
	}
	g := &Game{
		code:                   code,
		clients:                make([]*Client, 2),
		playerNames:            make([]string, 2),
		totals:                 make([]int, 2),
		victoryScore:           victoryScore,
		finalRoundTriggerIndex: -1,
		winnerIndex:            -1,
	}
	g.clients[0] = c
	g.playerNames[0] = name
	c.gameCode = code
	c.playerIndex = 0

	c.hub.mu.Lock()
	c.hub.games[code] = g
	c.hub.mu.Unlock()

	c.sendJSON(map[string]any{"type": "game_created", "gameCode": code})
}

func (c *Client) handleJoin(msg InMessage) {
	if msg.GameCode == "" {
		c.sendJSON(map[string]string{"type": "error", "message": "gameCode requerido"})
		return
	}

	c.hub.mu.Lock()
	g, ok := c.hub.games[msg.GameCode]
	c.hub.mu.Unlock()

	if !ok {
		c.sendJSON(map[string]string{"type": "error", "message": "partida no encontrada"})
		return
	}

	// Asignar como jugador 1 (índice 1)
	if g.clients[1] != nil {
		c.sendJSON(map[string]string{"type": "error", "message": "partida llena"})
		return
	}

	g.clients[1] = c
	c.gameCode = msg.GameCode
	c.playerIndex = 1

	name := msg.PlayerName
	if name == "" {
		name = "Jugador 2"
	}
	g.playerNames[1] = name

	c.sendJSON(map[string]any{"type": "game_joined", "gameCode": msg.GameCode, "playerIndex": 1})

	// Notificar al creador que alguien se unió
	c.hub.broadcastToGame(msg.GameCode, map[string]any{
		"type":        "player_joined",
		"playerIndex": 1,
		"playerName":  name,
	})

	// Broadcast game_state a ambos (partida completa, empieza el juego)
	c.hub.broadcastGameState(msg.GameCode)
}

func (h *Hub) broadcastToGame(gameCode string, payload any) {
	h.mu.RLock()
	g, ok := h.games[gameCode]
	h.mu.RUnlock()
	if !ok {
		return
	}

	data, _ := json.Marshal(payload)
	for _, client := range g.clients {
		if client != nil {
			select {
			case client.send <- data:
			default:
				log.Println("No se pudo enviar a un jugador, canal lleno")
			}
		}
	}
}

func (h *Hub) broadcastGameState(gameCode string) {
	h.mu.RLock()
	g, ok := h.games[gameCode]
	h.mu.RUnlock()
	if !ok {
		return
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	players := make([]map[string]any, 2)
	for i := 0; i < 2; i++ {
		name := "Jugador " + strconv.Itoa(i+1)
		if i < len(g.playerNames) && g.playerNames[i] != "" {
			name = g.playerNames[i]
		}
		total := 0
		if i < len(g.totals) {
			total = g.totals[i]
		}
		players[i] = map[string]any{"name": name, "total": total}
	}

	remainingCount := 0
	for _, d := range g.dice {
		if !d.Held {
			remainingCount++
		}
	}

	status := "playing"
	if g.winnerIndex >= 0 {
		status = "finished"
	}
	turnMoves := g.turnMoves
	if turnMoves == nil {
		turnMoves = []TurnMove{}
	}
	state := map[string]any{
		"type":                   "game_state",
		"players":                players,
		"currentPlayerIndex":     g.currentPlayerIndex,
		"dice":                   g.dice,
		"selectedIndices":        g.selectedIndices,
		"remainingDiceCount":     remainingCount,
		"turnPoints":             g.turnPoints,
		"turnMoves":              turnMoves,
		"victoryScore":           g.victoryScore,
		"finalRoundTriggerIndex": g.finalRoundTriggerIndex,
		"winnerIndex":            g.winnerIndex,
		"status":                 status,
	}
	h.broadcastToGame(gameCode, state)
}

func (c *Client) handleRoll() {
	if c.gameCode == "" {
		c.sendJSON(map[string]string{"type": "error", "message": "no estás en ninguna partida"})
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendJSON(map[string]string{"type": "error", "message": "partida no encontrada"})
		return
	}

	g.mu.Lock()
	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "la partida ha terminado"})
		return
	}
	if g.currentPlayerIndex != c.playerIndex {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "no es tu turno"})
		return
	}
	// Regla: no puedes tirar de nuevo hasta haber apartado al menos una combinación puntuable
	if len(g.dice) > 0 && !g.hasApartadoThisRoll {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "debes apartar al menos un dado que puntúe antes de volver a tirar"})
		return
	}

	var activeValues []int
	if len(g.dice) == 0 {
		dice := make([]Die, 6)
		activeValues = make([]int, 6)
		for i := range dice {
			v := rand.Intn(6) + 1
			dice[i] = Die{Value: v, Held: false}
			activeValues[i] = v
		}
		g.dice = dice
	} else {
		// Solo re-roll de los dados no held; los held mantienen su valor y estado
		activeValues = make([]int, 0, len(g.dice))
		for i := range g.dice {
			if !g.dice[i].Held {
				v := rand.Intn(6) + 1
				g.dice[i].Value = v
				activeValues = append(activeValues, v)
			}
			// Los held no se incluyen en activeValues: no se re-rollan ni cuentan para Farkle
		}
	}
	g.selectedIndices = nil
	g.hasApartadoThisRoll = false
	g.mu.Unlock()

	c.hub.broadcastToGame(c.gameCode, map[string]any{"type": "roll_result", "dice": g.dice})

	// Farkle: si no hay ninguna combinación puntuable en los dados activos, pierde los puntos del turno
	if !HasAnyScoringOption(activeValues) {
		g.mu.Lock()
		finishedIndex := g.currentPlayerIndex
			g.turnPoints = 0
			g.turnMoves = nil
			g.dice = nil
			g.selectedIndices = nil
			g.currentPlayerIndex = (g.currentPlayerIndex + 1) % 2

		// Si la ronda final estaba activa y el otro jugador acaba de Farklear
		if g.finalRoundTriggerIndex >= 0 && finishedIndex != g.finalRoundTriggerIndex {
			t0, t1 := g.totals[0], g.totals[1]
			winner := 0
			if t1 > t0 {
				winner = 1
			} else if t0 == t1 {
				winner = g.finalRoundTriggerIndex
			}
			g.winnerIndex = winner
			g.finishedAt = time.Now()
			g.turnMoves = nil
			g.mu.Unlock()
			c.hub.broadcastToGame(c.gameCode, map[string]any{"type": "farkle", "message": "Farkle: pierdes los puntos del turno"})
			c.hub.broadcastToGame(c.gameCode, map[string]any{"type": "game_over", "winner": winner, "message": "Partida terminada"})
			c.hub.broadcastGameState(c.gameCode)
		} else {
			g.mu.Unlock()
			c.hub.broadcastToGame(c.gameCode, map[string]any{"type": "farkle", "message": "Farkle: pierdes los puntos del turno"})
		}
	}

	c.hub.broadcastGameState(c.gameCode)
}

func (c *Client) handleToggleSelect(msg InMessage) {
	if c.gameCode == "" {
		c.sendJSON(map[string]string{"type": "error", "message": "no estás en ninguna partida"})
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendJSON(map[string]string{"type": "error", "message": "partida no encontrada"})
		return
	}

	g.mu.Lock()
	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "la partida ha terminado"})
		return
	}
	if g.currentPlayerIndex != c.playerIndex {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "no es tu turno"})
		return
	}
	if msg.Index < 0 || msg.Index >= len(g.dice) {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "índice inválido"})
		return
	}
	if g.dice[msg.Index].Held {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "no puedes seleccionar un dado ya apartado"})
		return
	}

	// Toggle: si ya está seleccionado, quitarlo; si no, añadirlo
	found := false
	for i, idx := range g.selectedIndices {
		if idx == msg.Index {
			g.selectedIndices = append(g.selectedIndices[:i], g.selectedIndices[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		g.selectedIndices = append(g.selectedIndices, msg.Index)
	}

	g.mu.Unlock()
	c.hub.broadcastGameState(c.gameCode)
}

func (c *Client) handleApartar() {
	if c.gameCode == "" {
		c.sendJSON(map[string]string{"type": "error", "message": "no estás en ninguna partida"})
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendJSON(map[string]string{"type": "error", "message": "partida no encontrada"})
		return
	}

	g.mu.Lock()
	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "la partida ha terminado"})
		return
	}
	if g.currentPlayerIndex != c.playerIndex {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "no es tu turno"})
		return
	}
	if len(g.dice) == 0 {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "primero debes tirar los dados"})
		return
	}
	if len(g.selectedIndices) == 0 {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "debes seleccionar dados antes de apartar"})
		return
	}

	// Obtener valores de los dados seleccionados (solo los no held)
	pickedValues := make([]int, 0, len(g.selectedIndices))
	for _, idx := range g.selectedIndices {
		if idx >= 0 && idx < len(g.dice) && !g.dice[idx].Held {
			pickedValues = append(pickedValues, g.dice[idx].Value)
		}
	}
	if len(pickedValues) == 0 {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "selecciona dados que no estén ya apartados"})
		return
	}

	valid, points := ScoreSelection(pickedValues)
	if !valid {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "selección inválida: todos los dados deben puntuar"})
		return
	}

	g.turnPoints += points
	g.hasApartadoThisRoll = true
	g.turnMoves = append(g.turnMoves, TurnMove{
		ID:     len(g.turnMoves) + 1,
		Values: pickedValues,
		Points: points,
	})

	// Marcar los dados seleccionados como held (apartados)
	for _, idx := range g.selectedIndices {
		if idx >= 0 && idx < len(g.dice) {
			g.dice[idx].Held = true
		}
	}
	g.selectedIndices = nil

	// Mano limpia: si todos están held, volver a tener 6 para tirar
	allHeld := true
	for _, d := range g.dice {
		if !d.Held {
			allHeld = false
			break
		}
	}
	if allHeld {
		g.dice = nil
	}

	g.mu.Unlock()
	c.hub.broadcastGameState(c.gameCode)
}

func (c *Client) handleBank() {
	if c.gameCode == "" {
		c.sendJSON(map[string]string{"type": "error", "message": "no estás en ninguna partida"})
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendJSON(map[string]string{"type": "error", "message": "partida no encontrada"})
		return
	}

	g.mu.Lock()
	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "la partida ha terminado"})
		return
	}
	if g.currentPlayerIndex != c.playerIndex {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "no es tu turno"})
		return
	}
	if g.turnPoints <= 0 {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "no tienes puntos que plantarte"})
		return
	}
	hasActiveDice := false
	for _, d := range g.dice {
		if !d.Held {
			hasActiveDice = true
			break
		}
	}
	if hasActiveDice && !g.hasApartadoThisRoll {
		g.mu.Unlock()
		c.sendJSON(map[string]string{"type": "error", "message": "debes apartar al menos una combinación antes de plantarte"})
		return
	}

	finishedIndex := c.playerIndex
	g.totals[c.playerIndex] += g.turnPoints
	g.turnPoints = 0
	g.turnMoves = nil
	g.dice = nil
	g.selectedIndices = nil
	g.hasApartadoThisRoll = false

	// Si alcanza la meta y aún no se había disparado la ronda final
	if g.finalRoundTriggerIndex < 0 && g.totals[c.playerIndex] >= g.victoryScore {
		g.finalRoundTriggerIndex = c.playerIndex
		g.currentPlayerIndex = (g.currentPlayerIndex + 1) % 2
		g.mu.Unlock()
		c.hub.broadcastToGame(c.gameCode, map[string]any{
			"type": "final_round",
			"message": "Ronda final para el otro jugador",
		})
		c.hub.broadcastGameState(c.gameCode)
		return
	}

	g.currentPlayerIndex = (g.currentPlayerIndex + 1) % 2

	// Si la ronda final ya estaba activa y el otro jugador acaba de terminar su turno
	if g.finalRoundTriggerIndex >= 0 && finishedIndex != g.finalRoundTriggerIndex {
		t0, t1 := g.totals[0], g.totals[1]
		winner := 0
		if t1 > t0 {
			winner = 1
		} else if t0 == t1 {
			winner = g.finalRoundTriggerIndex
		}
		g.winnerIndex = winner
		g.finishedAt = time.Now()
		g.mu.Unlock()
		c.hub.broadcastToGame(c.gameCode, map[string]any{
			"type":    "game_over",
			"winner":  winner,
			"message": "Partida terminada",
		})
		c.hub.broadcastGameState(c.gameCode)
		return
	}

	g.mu.Unlock()
	c.hub.broadcastGameState(c.gameCode)
}
