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

// Tipos de mensajes WebSocket
const (
	msgPing            = "ping"
	msgPong            = "pong"
	msgCreate          = "create"
	msgJoin            = "join"
	msgStart           = "start"
	msgRoll            = "roll"
	msgToggleSelect    = "toggle_select"
	msgSetAside        = "set_aside"
	msgBank            = "bank"
	msgError           = "error"
	msgGameCreated     = "game_created"
	msgGameJoined      = "game_joined"
	msgGameStarted     = "game_started"
	msgGameState       = "game_state"
	msgGameOver        = "game_over"
	msgPlayerJoined    = "player_joined"
	msgPlayerDisconnected = "player_disconnected"
	msgRollResult      = "roll_result"
	msgFarkle          = "farkle"
	msgHotDice         = "hot_dice"
	msgTurnChanged     = "turn_changed"
	msgFinalRound      = "final_round"
)

const (
	jsonKeyType   = "type"
	jsonKeyMsg    = "message"
	jsonKeyGameCode = "gameCode"
	jsonKeyWinner = "winner"
)

// Mensajes de error
const (
	errNoGame             = "You are not in any game"
	errGameNotFound       = "Game not found"
	errNotYourTurn        = "Not your turn"
	errGameFinished       = "The game has ended"
	errGameFull           = "Game is full"
	errGameCodeRequired   = "Game code required"
	errInvalidJSON        = "Invalid JSON"
	errInvalidIndex       = "Invalid index"
	errRollWithoutApartar = "You must set aside at least one scoring die before rolling again"
	errSelectHeldDie      = "You cannot select a die that is already set aside"
	errRollFirst          = "You must roll the dice first"
	errSelectBeforeApart  = "You must select dice before setting aside"
	errSelectNotHeld      = "Select dice that are not already set aside"
	errInvalidSelection   = "Invalid selection: all dice must score"
	errBankNoPoints       = "You have no points to bank"
	errBankMustApartar    = "You must set aside at least one combination before banking"
)

const (
	invalidIndex = -1
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
	finalRoundPlayedExtra []bool
	winnerIndex           int        // -1 si la partida sigue
	finishedAt            time.Time  // cuándo terminó la partida
	mu                    sync.RWMutex
}

// nextActivePlayerIndex devuelve el siguiente índice de jugador con cliente activo
// empezando después de from, recorriendo de forma circular. Devuelve -1 si no hay ninguno.
func (g *Game) nextActivePlayerIndex(from int) int {
	n := len(g.clients)
	if n == 0 {
		return -1
	}
	for step := 1; step <= n; step++ {
		idx := (from + step) % n
		if g.clients[idx] != nil {
			return idx
		}
	}
	return -1
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
	b := make([]byte, Cfg.GameCodeLength)
	for i := range b {
		b[i] = Cfg.GameCodeChars[rand.Intn(len(Cfg.GameCodeChars))]
	}
	return string(b)
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			wsConnections.Inc()
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.handleClientDisconnect(client)
				delete(h.clients, client)
				close(client.send)
				wsConnections.Dec()
			}
		}
	}
}

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

	// Calcular jugadores restantes todavía en la partida
	remaining := make([]int, 0, len(g.clients))
	for i, other := range g.clients {
		if other != nil {
			remaining = append(remaining, i)
		}
	}

	// Si no queda nadie, eliminamos la partida
	if len(remaining) == 0 {
		g.mu.Unlock()
		h.mu.Lock()
		delete(h.games, client.gameCode)
		activeGames.Dec()
		h.mu.Unlock()
		return
	}

	gameCode := client.gameCode

	// Si solo queda un jugador, ese jugador gana por desconexión del resto
	if len(remaining) == 1 {
		winnerIndex := remaining[0]
		g.winnerIndex = winnerIndex
		g.finishedAt = time.Now()
		g.mu.Unlock()

		h.broadcastToGame(gameCode, map[string]any{
			jsonKeyType:  msgPlayerDisconnected,
			jsonKeyMsg:   "El otro jugador se ha desconectado. Ganas la partida.",
			jsonKeyWinner: winnerIndex,
		})
		h.broadcastGameState(gameCode)
		return
	}

	// Si quedan varios jugadores, la partida continúa sin el jugador desconectado
	g.mu.Unlock()
	h.broadcastGameState(gameCode)
}

// cleanupFinishedGames elimina partidas terminadas hace más de FinishedGameRetention.
func (h *Hub) cleanupFinishedGames() {
	for range time.Tick(Cfg.CleanupInterval) {
		h.mu.Lock()
		now := time.Now()
		for code, g := range h.games {
			g.mu.RLock()
			finished := g.winnerIndex >= 0 && !g.finishedAt.IsZero() && now.Sub(g.finishedAt) > Cfg.FinishedGameRetention
			g.mu.RUnlock()
			if finished {
				delete(h.games, code)
				activeGames.Dec()
				log.Printf("Partida %s eliminada (terminada hace >%v)", code, Cfg.FinishedGameRetention)
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
			c.sendError(errInvalidJSON)
			continue
		}

		switch msg.Type {
		case msgPing:
			c.sendJSON(map[string]string{jsonKeyType: msgPong})
		case msgCreate:
			c.handleCreate(msg)
		case msgJoin:
			c.handleJoin(msg)
		case msgStart:
			c.handleStartGame(msg)
		case msgRoll:
			c.handleRoll()
		case msgToggleSelect:
			c.handleToggleSelect(msg)
		case msgSetAside:
			c.handleApartar()
		case msgBank:
			c.handleBank()
		default:
			c.sendError("tipo desconocido: " + msg.Type)
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

func (c *Client) sendError(msg string) {
	c.sendJSON(map[string]string{jsonKeyType: msgError, jsonKeyMsg: msg})
}

func (c *Client) handleCreate(msg InMessage) {
	code := generateGameCode()
	name := msg.PlayerName
	if name == "" {
		name = "Jugador 1"
	}
	victoryScore := msg.VictoryScore
	if victoryScore < Cfg.MinVictoryScore || victoryScore > Cfg.MaxVictoryScore {
		victoryScore = Cfg.DefaultVictoryScore
	}
	g := &Game{
		code:                   code,
		clients:                make([]*Client, Cfg.NumPlayers),
		playerNames:            make([]string, Cfg.NumPlayers),
		totals:                 make([]int, Cfg.NumPlayers),
		victoryScore:           victoryScore,
		finalRoundTriggerIndex: invalidIndex,
		finalRoundPlayedExtra:  make([]bool, Cfg.NumPlayers),
		winnerIndex:            invalidIndex,
	}
	g.clients[0] = c
	g.playerNames[0] = name
	c.gameCode = code
	c.playerIndex = 0

	c.hub.mu.Lock()
	c.hub.games[code] = g
	c.hub.mu.Unlock()

	gamesCreatedTotal.Inc()
	activeGames.Inc()

	c.sendJSON(map[string]any{jsonKeyType: msgGameCreated, jsonKeyGameCode: code})
}

func (c *Client) handleJoin(msg InMessage) {
	if msg.GameCode == "" {
		c.sendError(errGameCodeRequired)
		return
	}

	c.hub.mu.Lock()
	g, ok := c.hub.games[msg.GameCode]
	c.hub.mu.Unlock()

	if !ok {
		c.sendError(errGameNotFound)
		return
	}

	// Buscar el primer hueco libre para este jugador
	slot := -1
	for i := 0; i < len(g.clients); i++ {
		if g.clients[i] == nil {
			slot = i
			break
		}
	}
	if slot == -1 {
		c.sendError(errGameFull)
		return
	}

	g.clients[slot] = c
	c.gameCode = msg.GameCode
	c.playerIndex = slot

	name := msg.PlayerName
	if name == "" {
		name = "Jugador " + strconv.Itoa(slot+1)
	}
	g.playerNames[slot] = name

	gamesJoinedTotal.Inc()

	c.sendJSON(map[string]any{
		jsonKeyType:     msgGameJoined,
		jsonKeyGameCode: msg.GameCode,
		"playerIndex":   slot,
	})

	c.hub.broadcastToGame(msg.GameCode, map[string]any{
		jsonKeyType:      msgPlayerJoined,
		"playerIndex":    slot,
		"playerName":     name,
	})

	// Actualizar el estado para todos los jugadores tras la incorporación
	c.hub.broadcastGameState(msg.GameCode)
}

// handleStartGame marca el inicio de la partida a nivel de lobby,
// notificando a todos los jugadores que pueden abandonar el lobby.
func (c *Client) handleStartGame(msg InMessage) {
	if c.gameCode == "" {
		c.sendError(errNoGame)
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendError(errGameNotFound)
		return
	}

	g.mu.RLock()
	isCreator := c.playerIndex == 0
	g.mu.RUnlock()

	if !isCreator {
		c.sendError("Only the creator can start the game")
		return
	}

	// Notificar a todos los jugadores en la partida que el juego ha empezado
	c.hub.broadcastToGame(c.gameCode, map[string]any{
		jsonKeyType: msgGameStarted,
	})
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

	players := make([]map[string]any, Cfg.NumPlayers)
	for i := 0; i < Cfg.NumPlayers; i++ {
		active := i < len(g.clients) && g.clients[i] != nil
		name := ""
		total := 0
		if active {
			if i < len(g.playerNames) && g.playerNames[i] != "" {
				name = g.playerNames[i]
			} else {
				name = "Jugador " + strconv.Itoa(i+1)
			}
			if i < len(g.totals) {
				total = g.totals[i]
			}
		}
		players[i] = map[string]any{
			"name":   name,
			"total":  total,
			"active": active,
		}
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
		jsonKeyType:               msgGameState,
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
	start := time.Now()
	defer func() {
		rollDuration.Observe(time.Since(start).Seconds())
	}()

	if c.gameCode == "" {
		c.sendError(errNoGame)
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendError(errGameNotFound)
		return
	}

	g.mu.Lock()
	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		c.sendError(errGameFinished)
		return
	}
	if g.currentPlayerIndex != c.playerIndex {
		g.mu.Unlock()
		c.sendError(errNotYourTurn)
		return
	}
	if len(g.dice) > 0 && !g.hasApartadoThisRoll {
		g.mu.Unlock()
		c.sendError(errRollWithoutApartar)
		return
	}

	var activeValues []int
	if len(g.dice) == 0 {
		dice := make([]Die, Cfg.NumDice)
		activeValues = make([]int, Cfg.NumDice)
		for i := range dice {
			v := rand.Intn(Cfg.NumDice) + 1
			dice[i] = Die{Value: v, Held: false}
			activeValues[i] = v
		}
		g.dice = dice
	} else {
		// Solo re-roll de los dados no held; los held mantienen su valor y estado
		activeValues = make([]int, 0, len(g.dice))
		for i := range g.dice {
			if !g.dice[i].Held {
				v := rand.Intn(Cfg.NumDice) + 1
				g.dice[i].Value = v
				activeValues = append(activeValues, v)
			}
			// Los held no se incluyen en activeValues: no se re-rollan ni cuentan para Farkle
		}
	}
	g.selectedIndices = nil
	g.hasApartadoThisRoll = false
	g.mu.Unlock()

	c.hub.broadcastToGame(c.gameCode, map[string]any{jsonKeyType: msgRollResult, "dice": g.dice})

	// Farkle: si no hay ninguna combinación puntuable en los dados activos, pierde los puntos del turno
	if !HasAnyScoringOption(activeValues) {
		farklesTotal.Inc()
		g.mu.Lock()
		finishedIndex := g.currentPlayerIndex
		g.turnPoints = 0
		g.turnMoves = nil
		g.dice = nil
		g.selectedIndices = nil

		// Pasar turno al siguiente jugador activo, si lo hay
		next := g.nextActivePlayerIndex(g.currentPlayerIndex)
		g.currentPlayerIndex = next

		// Si la ronda final estaba activa, marcar el turno extra del jugador y comprobar si termina la partida
		finalFinished := false
		if g.finalRoundTriggerIndex >= 0 {
			// El jugador que disparó la ronda final no cuenta como turno extra
			if finishedIndex != g.finalRoundTriggerIndex && finishedIndex >= 0 && finishedIndex < len(g.finalRoundPlayedExtra) {
				g.finalRoundPlayedExtra[finishedIndex] = true
			}

			allDone := true
			for i := 0; i < len(g.clients); i++ {
				if i == g.finalRoundTriggerIndex {
					continue
				}
				if g.clients[i] != nil && !g.finalRoundPlayedExtra[i] {
					allDone = false
					break
				}
			}

			if allDone {
				// Calcular ganador por puntuación; en empate, favorece al jugador que disparó la ronda final
				winner := -1
				best := -1
				for i := 0; i < len(g.totals); i++ {
					if g.clients[i] == nil {
						continue
					}
					if g.totals[i] > best {
						best = g.totals[i]
						winner = i
					} else if g.totals[i] == best && best >= 0 {
						if winner != g.finalRoundTriggerIndex && i == g.finalRoundTriggerIndex {
							winner = i
						}
					}
				}
				if winner >= 0 {
					g.winnerIndex = winner
					g.finishedAt = time.Now()
					g.turnMoves = nil
					finalFinished = true
				}
			}
		}

		if finalFinished {
			g.mu.Unlock()
			c.hub.broadcastToGame(c.gameCode, map[string]any{jsonKeyType: msgFarkle, jsonKeyMsg: "Farkle: pierdes los puntos del turno"})
			c.hub.broadcastToGame(c.gameCode, map[string]any{jsonKeyType: msgGameOver, jsonKeyWinner: g.winnerIndex, jsonKeyMsg: "Partida terminada"})
			c.hub.broadcastGameState(c.gameCode)
		} else {
			g.mu.Unlock()
			c.hub.broadcastToGame(c.gameCode, map[string]any{jsonKeyType: msgFarkle, jsonKeyMsg: "Farkle: pierdes los puntos del turno"})
		}
	}

	c.hub.broadcastGameState(c.gameCode)
}

func (c *Client) handleToggleSelect(msg InMessage) {
	if c.gameCode == "" {
		c.sendError(errNoGame)
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendError(errGameNotFound)
		return
	}

	g.mu.Lock()
	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		c.sendError(errGameFinished)
		return
	}
	if g.currentPlayerIndex != c.playerIndex {
		g.mu.Unlock()
		c.sendError(errNotYourTurn)
		return
	}
	if msg.Index < 0 || msg.Index >= len(g.dice) {
		g.mu.Unlock()
		c.sendError(errInvalidIndex)
		return
	}
	if g.dice[msg.Index].Held {
		g.mu.Unlock()
		c.sendError(errSelectHeldDie)
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
		c.sendError(errNoGame)
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendError(errGameNotFound)
		return
	}

	g.mu.Lock()
	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		c.sendError(errGameFinished)
		return
	}
	if g.currentPlayerIndex != c.playerIndex {
		g.mu.Unlock()
		c.sendError(errNotYourTurn)
		return
	}
	if len(g.dice) == 0 {
		g.mu.Unlock()
		c.sendError(errRollFirst)
		return
	}
	if len(g.selectedIndices) == 0 {
		g.mu.Unlock()
		c.sendError(errSelectBeforeApart)
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
		c.sendError(errSelectNotHeld)
		return
	}

	valid, points := ScoreSelection(pickedValues)
	if !valid {
		g.mu.Unlock()
		c.sendError(errInvalidSelection)
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

	// Mano limpia (hot dice): si todos están held, volver a tener 6 para tirar
	allHeld := true
	for _, d := range g.dice {
		if !d.Held {
			allHeld = false
			break
		}
	}
	if allHeld {
		g.dice = nil
		g.mu.Unlock()
		c.hub.broadcastToGame(c.gameCode, map[string]any{
			jsonKeyType: msgHotDice,
			jsonKeyMsg:  "¡Mano limpia! Puedes volver a tirar los 6 dados",
		})
		c.hub.broadcastGameState(c.gameCode)
		return
	}

	g.mu.Unlock()
	c.hub.broadcastGameState(c.gameCode)
}

func (c *Client) handleBank() {
	if c.gameCode == "" {
		c.sendError(errNoGame)
		return
	}

	c.hub.mu.RLock()
	g, ok := c.hub.games[c.gameCode]
	c.hub.mu.RUnlock()
	if !ok {
		c.sendError(errGameNotFound)
		return
	}

	g.mu.Lock()
	if g.winnerIndex >= 0 {
		g.mu.Unlock()
		c.sendError(errGameFinished)
		return
	}
	if g.currentPlayerIndex != c.playerIndex {
		g.mu.Unlock()
		c.sendError(errNotYourTurn)
		return
	}
	if g.turnPoints <= 0 {
		g.mu.Unlock()
		c.sendError(errBankNoPoints)
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
		c.sendError(errBankMustApartar)
		return
	}

	finishedIndex := c.playerIndex
	g.totals[c.playerIndex] += g.turnPoints
	g.turnPoints = 0
	g.turnMoves = nil
	g.dice = nil
	g.selectedIndices = nil
	g.hasApartadoThisRoll = false

	if g.finalRoundTriggerIndex == invalidIndex && g.totals[c.playerIndex] >= g.victoryScore {
		g.finalRoundTriggerIndex = c.playerIndex
		g.finalRoundPlayedExtra = make([]bool, len(g.clients))
		g.currentPlayerIndex = g.nextActivePlayerIndex(g.currentPlayerIndex)
		g.mu.Unlock()
		c.hub.broadcastToGame(c.gameCode, map[string]any{
			jsonKeyType:  msgFinalRound,
			jsonKeyMsg:   "Ronda final para el otro jugador",
		})
		c.hub.broadcastGameState(c.gameCode)
		return
	}

	nextPlayer := g.nextActivePlayerIndex(g.currentPlayerIndex)
	g.currentPlayerIndex = nextPlayer
	nextName := "Jugador " + strconv.Itoa(nextPlayer+1)
	if nextPlayer < len(g.playerNames) && g.playerNames[nextPlayer] != "" {
		nextName = g.playerNames[nextPlayer]
	}

	// Si la ronda final ya estaba activa, marcar el turno extra del jugador y comprobar si termina la partida
	finalFinished := false
	if g.finalRoundTriggerIndex >= 0 {
		if finishedIndex != g.finalRoundTriggerIndex && finishedIndex >= 0 && finishedIndex < len(g.finalRoundPlayedExtra) {
			g.finalRoundPlayedExtra[finishedIndex] = true
		}

		allDone := true
		for i := 0; i < len(g.clients); i++ {
			if i == g.finalRoundTriggerIndex {
				continue
			}
			if g.clients[i] != nil && !g.finalRoundPlayedExtra[i] {
				allDone = false
				break
			}
		}

		if allDone {
			winner := -1
			best := -1
			for i := 0; i < len(g.totals); i++ {
				if g.clients[i] == nil {
					continue
				}
				if g.totals[i] > best {
					best = g.totals[i]
					winner = i
				} else if g.totals[i] == best && best >= 0 {
					if winner != g.finalRoundTriggerIndex && i == g.finalRoundTriggerIndex {
						winner = i
					}
				}
			}
			if winner >= 0 {
				g.winnerIndex = winner
				g.finishedAt = time.Now()
				finalFinished = true
			}
		}
	}

	if finalFinished {
		g.mu.Unlock()
		c.hub.broadcastToGame(c.gameCode, map[string]any{
			jsonKeyType:  msgGameOver,
			jsonKeyWinner: g.winnerIndex,
			jsonKeyMsg:   "Partida terminada",
		})
		c.hub.broadcastGameState(c.gameCode)
		return
	}

	g.mu.Unlock()
	c.hub.broadcastToGame(c.gameCode, map[string]any{
		jsonKeyType: msgTurnChanged,
		jsonKeyMsg:  "Turno de " + nextName,
	})
	c.hub.broadcastGameState(c.gameCode)
}
