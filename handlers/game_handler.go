package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aminearbi/ludo-nadwa-server/models"
)

// Handler wraps the game manager and provides HTTP endpoints
type Handler struct {
	gameManager *models.GameManager
	hub         *Hub // WebSocket hub for broadcasting
}

// NewHandler creates a new handler
func NewHandler(gm *models.GameManager) *Handler {
	return &Handler{
		gameManager: gm,
		hub:         nil,
	}
}

// SetHub sets the WebSocket hub for broadcasting
func (h *Handler) SetHub(hub *Hub) {
	h.hub = hub
}

// broadcast sends a WebSocket event to all clients in a game
func (h *Handler) broadcast(gameCode string, eventType string, data map[string]interface{}) {
	if h.hub != nil {
		h.hub.BroadcastToGame(gameCode, WebSocketEvent{
			Type:      eventType,
			Data:      data,
			Timestamp: time.Now(),
		})
	}
}

// CreateGameRequest represents the request to create a game
type CreateGameRequest struct {
	MaxPlayers int    `json:"max_players"`
	PlayerName string `json:"player_name"`
	PlayerID   string `json:"player_id"`
}

// CreateGameResponse represents the response when creating a game
type CreateGameResponse struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	MaxPlayers int    `json:"max_players"`
}

// JoinGameRequest represents the request to join a game
type JoinGameRequest struct {
	Code       string `json:"code"`
	PlayerID   string `json:"player_id"`
	PlayerName string `json:"player_name"`
}

// JoinGameResponse represents the response when joining a game
type JoinGameResponse struct {
	Message string                 `json:"message"`
	Game    map[string]interface{} `json:"game"`
}

// StartGameRequest represents the request to start a game
type StartGameRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
}

// RollDiceRequest represents the request to roll dice
type RollDiceRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
}

// RollDiceResponse represents the response when rolling dice
type RollDiceResponse struct {
	Roll       int   `json:"roll"`
	ValidMoves []int `json:"valid_moves"` // IDs of pieces that can be moved
	HasMoves   bool  `json:"has_moves"`   // Whether any valid move exists
}

// MovePieceRequest represents the request to move a piece
type MovePieceRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
	PieceID  int    `json:"piece_id"`
}

// SkipTurnRequest represents the request to skip a turn
type SkipTurnRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
}

// SetReadyRequest represents the request to set player ready status
type SetReadyRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
	Ready    bool   `json:"ready"`
}

// KickPlayerRequest represents the request to kick a player
type KickPlayerRequest struct {
	Code         string `json:"code"`
	HostID       string `json:"host_id"`
	PlayerToKick string `json:"player_to_kick"`
}

// LeaveGameRequest represents the request to leave a game
type LeaveGameRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
}

// PauseGameRequest represents the request to pause a game
type PauseGameRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
}

// ResumeGameRequest represents the request to resume a game
type ResumeGameRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
}

// ChatMessageRequest represents the request to send a chat message
type ChatMessageRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
	Message  string `json:"message"`
}

// SpectateRequest represents the request to join as a spectator
type SpectateRequest struct {
	Code         string `json:"code"`
	SpectatorID  string `json:"spectator_id"`
	SpectatorName string `json:"spectator_name"`
}

// RematchRequest represents the request to start a rematch
type RematchRequest struct {
	Code   string `json:"code"`
	HostID string `json:"host_id"`
}

// AddBotRequest represents the request to add a bot to a game
type AddBotRequest struct {
	Code   string `json:"code"`
	HostID string `json:"host_id"`
}

// RemoveBotRequest represents the request to remove a bot from a game
type RemoveBotRequest struct {
	Code   string `json:"code"`
	HostID string `json:"host_id"`
	BotID  string `json:"bot_id"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateGame handles game creation
func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Require player info for host
	if req.PlayerID == "" || req.PlayerName == "" {
		respondWithError(w, "Player ID and name are required to create a game", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.CreateGame(req.PlayerID, req.PlayerName, req.MaxPlayers)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := CreateGameResponse{
		Code:       game.Code,
		Message:    "Game created successfully. Share this code with other players.",
		MaxPlayers: game.MaxPlayers,
	}

	respondWithJSON(w, response, http.StatusCreated)
}

// JoinGame handles joining a game
func (h *Handler) JoinGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JoinGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" || req.PlayerID == "" || req.PlayerName == "" {
		respondWithError(w, "code, player_id, and player_name are required", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.JoinGame(req.Code, req.PlayerID, req.PlayerName)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast player joined event
	h.broadcast(req.Code, "player_joined", map[string]interface{}{
		"player_id":   req.PlayerID,
		"player_name": req.PlayerName,
		"game":        game.GetGameState(),
	})

	response := JoinGameResponse{
		Message: "Successfully joined the game",
		Game:    game.GetGameState(),
	}

	respondWithJSON(w, response, http.StatusOK)
}

// StartGame handles starting a game
func (h *Handler) StartGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StartGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PlayerID == "" {
		respondWithError(w, "player_id is required", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.StartGame(req.PlayerID); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast game started event
	h.broadcast(req.Code, "game_started", map[string]interface{}{
		"game": game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Game started successfully",
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// GetGameState handles retrieving game state
func (h *Handler) GetGameState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, "code parameter is required", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	respondWithJSON(w, game.GetGameState(), http.StatusOK)
}

// RollDice handles dice rolling
func (h *Handler) RollDice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RollDiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	roll, rollErr := game.RollDice(req.PlayerID)
	
	// Handle the three-sixes case - still report the roll but turn is lost
	threeSixes := rollErr == models.ErrThreeSixes
	if rollErr != nil && !threeSixes {
		respondWithError(w, rollErr.Error(), http.StatusBadRequest)
		return
	}
	
	validMoves := game.GetValidMoves(req.PlayerID)
	game.UpdateActivity()

	// Broadcast dice roll event
	eventData := map[string]interface{}{
		"player_id":    req.PlayerID,
		"roll":         roll,
		"valid_moves":  validMoves,
		"has_moves":    len(validMoves) > 0,
		"three_sixes":  threeSixes,
	}
	
	if threeSixes {
		eventData["message"] = "Three consecutive sixes! Turn forfeited."
	}
	
	h.broadcast(req.Code, "dice_rolled", eventData)

	response := RollDiceResponse{
		Roll:       roll,
		ValidMoves: validMoves,
		HasMoves:   len(validMoves) > 0,
	}

	respondWithJSON(w, response, http.StatusOK)
}

// MovePiece handles moving a piece
func (h *Handler) MovePiece(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MovePieceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.MovePiece(req.PlayerID, req.PieceID); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	gameState := game.GetGameState()

	// Broadcast piece moved event
	h.broadcast(req.Code, "piece_moved", map[string]interface{}{
		"player_id": req.PlayerID,
		"piece_id":  req.PieceID,
		"game":      gameState,
	})

	// Check for game end
	if gameState["state"] == "ended" {
		h.broadcast(req.Code, "game_ended", map[string]interface{}{
			"winner": gameState["winner"],
			"game":   gameState,
		})
	}

	respondWithJSON(w, map[string]interface{}{
		"message": "Piece moved successfully",
		"game":    gameState,
	}, http.StatusOK)
}

// SkipTurn handles skipping a turn when no valid moves are available
func (h *Handler) SkipTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SkipTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Verify player has no valid moves before allowing skip
	if game.HasValidMoves(req.PlayerID) {
		respondWithError(w, "Cannot skip turn when valid moves are available", http.StatusBadRequest)
		return
	}

	if err := game.SkipTurn(req.PlayerID); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast turn skipped event
	h.broadcast(req.Code, "turn_skipped", map[string]interface{}{
		"player_id": req.PlayerID,
		"game":      game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Turn skipped",
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// SetReady handles setting a player's ready status
func (h *Handler) SetReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetReadyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.SetPlayerReady(req.PlayerID, req.Ready); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast player ready status change
	h.broadcast(req.Code, "player_ready", map[string]interface{}{
		"player_id":        req.PlayerID,
		"ready":            req.Ready,
		"all_players_ready": game.AreAllPlayersReady(),
		"game":             game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message":          "Ready status updated",
		"ready":            req.Ready,
		"all_players_ready": game.AreAllPlayersReady(),
	}, http.StatusOK)
}

// KickPlayer handles kicking a player from the game
func (h *Handler) KickPlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req KickPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.KickPlayer(req.HostID, req.PlayerToKick); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast player kicked event
	h.broadcast(req.Code, "player_kicked", map[string]interface{}{
		"player_id": req.PlayerToKick,
		"game":      game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Player kicked successfully",
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// LeaveGame handles a player leaving the game
func (h *Handler) LeaveGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LeaveGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.LeaveGame(req.PlayerID); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast player left event
	h.broadcast(req.Code, "player_left", map[string]interface{}{
		"player_id": req.PlayerID,
		"game":      game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Left game successfully",
	}, http.StatusOK)
}

// PauseGame handles pausing the game
func (h *Handler) PauseGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PauseGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.PauseGame(req.PlayerID); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast game paused event
	h.broadcast(req.Code, "game_paused", map[string]interface{}{
		"paused_by": req.PlayerID,
		"game":      game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Game paused",
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// ResumeGame handles resuming the game
func (h *Handler) ResumeGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ResumeGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.ResumeGame(req.PlayerID); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast game resumed event
	h.broadcast(req.Code, "game_resumed", map[string]interface{}{
		"resumed_by": req.PlayerID,
		"game":       game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Game resumed",
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// SendChat handles sending a chat message
func (h *Handler) SendChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.SendChatMessage(req.PlayerID, req.Message); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get player name
	playerName := "Unknown"
	if player, exists := game.Players[req.PlayerID]; exists {
		playerName = player.Name
	}

	// Broadcast chat message event
	h.broadcast(req.Code, "chat_message", map[string]interface{}{
		"player_id":   req.PlayerID,
		"player_name": playerName,
		"message":     req.Message,
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Chat message sent",
	}, http.StatusOK)
}

// JoinAsSpectator handles joining a game as a spectator
func (h *Handler) JoinAsSpectator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SpectateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.JoinAsSpectator(req.Code, req.SpectatorID, req.SpectatorName)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast spectator joined event
	h.broadcast(req.Code, "spectator_joined", map[string]interface{}{
		"spectator_id":   req.SpectatorID,
		"spectator_name": req.SpectatorName,
		"game":           game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Joined as spectator",
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// Rematch handles requesting a rematch
func (h *Handler) Rematch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RematchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.Rematch(req.HostID); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast rematch event
	h.broadcast(req.Code, "rematch", map[string]interface{}{
		"message": "Rematch started - waiting for all players to be ready",
		"game":    game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Rematch started - waiting for all players to be ready",
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// GetMoveHistory handles getting the move history
func (h *Handler) GetMoveHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, "code parameter is required", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	respondWithJSON(w, map[string]interface{}{
		"move_history": game.MoveHistory,
	}, http.StatusOK)
}

// GetChat handles getting the chat history
func (h *Handler) GetChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		respondWithError(w, "code parameter is required", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.GetGame(code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	respondWithJSON(w, map[string]interface{}{
		"chat_messages": game.GetRecentChat(100),
	}, http.StatusOK)
}

// AddBot handles adding an AI player to the game
func (h *Handler) AddBot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AddBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, bot, err := h.gameManager.AddBot(req.Code, req.HostID)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast bot joined event
	h.broadcast(req.Code, "player_joined", map[string]interface{}{
		"player_id":   bot.ID,
		"player_name": bot.Name,
		"is_bot":      true,
		"game":        game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Bot added successfully",
		"bot_id":  bot.ID,
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// RemoveBot handles removing an AI player from the game
func (h *Handler) RemoveBot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RemoveBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	game, err := h.gameManager.RemoveBot(req.Code, req.HostID, req.BotID)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast bot left event
	h.broadcast(req.Code, "player_left", map[string]interface{}{
		"player_id": req.BotID,
		"is_bot":    true,
		"game":      game.GetGameState(),
	})

	respondWithJSON(w, map[string]interface{}{
		"message": "Bot removed successfully",
		"game":    game.GetGameState(),
	}, http.StatusOK)
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	respondWithJSON(w, ErrorResponse{Error: message}, statusCode)
}
