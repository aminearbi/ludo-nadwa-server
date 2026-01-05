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
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
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

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients grouped by game code
	games map[string]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast message to all clients in a game
	broadcast chan *GameMessage

	mu sync.RWMutex
}

// GameMessage represents a message to broadcast to a game
type GameMessage struct {
	GameCode string
	Message  []byte
}

// WebSocketEvent represents an event sent over WebSocket
type WebSocketEvent struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
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
			log.Printf("Client %s connected to game %s", client.playerID, client.gameCode)

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
			log.Printf("Client %s disconnected from game %s", client.playerID, client.gameCode)

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

// BroadcastToGame sends a message to all clients in a game
func (h *Hub) BroadcastToGame(gameCode string, event WebSocketEvent) {
	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	h.broadcast <- &GameMessage{
		GameCode: gameCode,
		Message:  message,
	}
}

// GetConnectedPlayers returns a list of connected player IDs for a game
func (h *Hub) GetConnectedPlayers(gameCode string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	players := []string{}
	if clients, ok := h.games[gameCode]; ok {
		for client := range clients {
			players = append(players, client.playerID)
		}
	}
	return players
}

// IsPlayerConnected checks if a player is connected to a game
func (h *Hub) IsPlayerConnected(gameCode, playerID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.games[gameCode]; ok {
		for client := range clients {
			if client.playerID == playerID {
				return true
			}
		}
	}
	return false
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
		http.Error(w, "Player not in game", http.StatusForbidden)
		return
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

	// Mark player as connected in the game model
	game.SetPlayerConnected(playerID, true)

	// Notify other players
	wsh.hub.BroadcastToGame(gameCode, WebSocketEvent{
		Type: "player_connected",
		Data: map[string]interface{}{
			"player_id":         playerID,
			"connected_players": wsh.hub.GetConnectedPlayers(gameCode),
		},
		Timestamp: time.Now(),
	})

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump(wsh)
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump(wsh *WebSocketHandler) {
	defer func() {
		// Mark player as disconnected in the game model
		if game, err := wsh.gameManager.GetGame(c.gameCode); err == nil {
			game.SetPlayerConnected(c.playerID, false)
		}
		
		// Notify other players of disconnect
		wsh.hub.BroadcastToGame(c.gameCode, WebSocketEvent{
			Type: "player_disconnected",
			Data: map[string]interface{}{
				"player_id":         c.playerID,
				"connected_players": wsh.hub.GetConnectedPlayers(c.gameCode),
			},
			Timestamp: time.Now(),
		})
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

		// Handle incoming messages (e.g., ping/heartbeat)
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			if msg["type"] == "ping" {
				response, _ := json.Marshal(WebSocketEvent{
					Type:      "pong",
					Data:      map[string]interface{}{},
					Timestamp: time.Now(),
				})
				c.send <- response
			}
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
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
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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
