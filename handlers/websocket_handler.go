package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aminearbi/ludo-nadwa-server/models"
	"github.com/gorilla/websocket"
)

// WebSocket configuration
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	gameCode string
	playerID string
}

// Hub maintains active clients and broadcasts refresh signals
type Hub struct {
	games      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *GameMessage
	mu         sync.RWMutex
}

// GameMessage represents a message to broadcast
type GameMessage struct {
	GameCode string
	Message  []byte
}

// RefreshEvent is the simplified event - just tells clients to fetch new state
type RefreshEvent struct {
	Type string `json:"type"` // Always "refresh"
	Hint string `json:"hint"` // What changed: "dice_rolled", "piece_moved", "player_joined", etc.
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		games:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *GameMessage),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.games[client.gameCode] == nil {
				h.games[client.gameCode] = make(map[*Client]bool)
			}
			h.games[client.gameCode][client] = true
			h.mu.Unlock()
			log.Printf("WS: %s connected to game %s", client.playerID, client.gameCode)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.games[client.gameCode]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.games, client.gameCode)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WS: %s disconnected from game %s", client.playerID, client.gameCode)

		case message := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.games[message.GameCode]; ok {
				for client := range clients {
					select {
					case client.send <- message.Message:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastRefresh sends a simple refresh signal to all clients in a game
func (h *Hub) BroadcastRefresh(gameCode string, hint string) {
	event := RefreshEvent{
		Type: "refresh",
		Hint: hint,
	}
	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling refresh event: %v", err)
		return
	}

	h.broadcast <- &GameMessage{
		GameCode: gameCode,
		Message:  message,
	}
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub         *Hub
	gameManager *models.GameManager
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *Hub, gm *models.GameManager) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		gameManager: gm,
	}
}

// HandleWebSocket handles WebSocket upgrade and connection
func (wsh *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	gameCode := r.URL.Query().Get("code")
	playerID := r.URL.Query().Get("player_id")

	if gameCode == "" || playerID == "" {
		http.Error(w, "code and player_id are required", http.StatusBadRequest)
		return
	}

	// Verify game exists and player is in it
	game, err := wsh.gameManager.GetGame(gameCode)
	if err != nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if _, exists := game.Players[playerID]; !exists {
		// Also allow spectators
		if _, specExists := game.Spectators[playerID]; !specExists {
			http.Error(w, "Player not in game", http.StatusForbidden)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:      wsh.hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		gameCode: gameCode,
		playerID: playerID,
	}

	wsh.hub.register <- client

	// Notify others that someone connected (they should refresh)
	wsh.hub.BroadcastRefresh(gameCode, "player_connected")

	go client.writePump()
	go client.readPump(wsh)
}

// readPump handles incoming messages (just ping/pong)
func (c *Client) readPump(wsh *WebSocketHandler) {
	defer func() {
		// Notify others on disconnect
		wsh.hub.BroadcastRefresh(c.gameCode, "player_disconnected")
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle ping from client
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			if msg["type"] == "ping" {
				response, _ := json.Marshal(map[string]string{"type": "pong"})
				c.send <- response
			}
		}
	}
}

// writePump sends messages to the client
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
