package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/aminearbi/ludo-nadwa-server/models"
)

// Handler wraps the game manager and provides HTTP endpoints
type Handler struct {
	gameManager *models.GameManager
}

// NewHandler creates a new handler
func NewHandler(gm *models.GameManager) *Handler {
	return &Handler{
		gameManager: gm,
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
	Roll int `json:"roll"`
}

// MovePieceRequest represents the request to move a piece
type MovePieceRequest struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
	PieceID  int    `json:"piece_id"`
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

	game, err := h.gameManager.CreateGame(req.MaxPlayers)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-join creator if playerID and name provided
	if req.PlayerID != "" && req.PlayerName != "" {
		_, err = h.gameManager.JoinGame(game.Code, req.PlayerID, req.PlayerName)
		if err != nil {
			respondWithError(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

	game, err := h.gameManager.GetGame(req.Code)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := game.StartGame(); err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

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

	// Check if it's the player's turn
	if game.CurrentTurn != req.PlayerID {
		respondWithError(w, "Not your turn", http.StatusBadRequest)
		return
	}

	roll := game.RollDice()

	response := RollDiceResponse{
		Roll: roll,
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

	respondWithJSON(w, map[string]interface{}{
		"message": "Piece moved successfully",
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
