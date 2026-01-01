package models

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// PlayerColor represents the color of a player's pieces
type PlayerColor string

const (
	Red    PlayerColor = "red"
	Blue   PlayerColor = "blue"
	Green  PlayerColor = "green"
	Yellow PlayerColor = "yellow"
	Purple PlayerColor = "purple"
)

// Piece represents a single game piece
type Piece struct {
	ID       int  `json:"id"`
	Position int  `json:"position"` // -1 for home, 0-51 for board, 100+ for finished
	IsHome   bool `json:"is_home"`
	IsSafe   bool `json:"is_safe"`
}

// Player represents a player in the game
type Player struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Color  PlayerColor `json:"color"`
	Pieces []Piece     `json:"pieces"`
	Order  int         `json:"order"` // Turn order
}

// GameState represents the current state of the game
type GameState string

const (
	Waiting GameState = "waiting" // Waiting for players to join
	Playing GameState = "playing" // Game in progress
	Ended   GameState = "ended"   // Game has ended
)

// Game represents a Ludo game session
type Game struct {
	Code          string               `json:"code"`           // 8-digit game code
	Players       map[string]*Player   `json:"players"`        // Map of player ID to Player
	State         GameState            `json:"state"`
	CurrentTurn   string               `json:"current_turn"`   // Player ID whose turn it is
	MaxPlayers    int                  `json:"max_players"`
	CreatedAt     time.Time            `json:"created_at"`
	LastDiceRoll  int                  `json:"last_dice_roll"`
	TurnStartTime time.Time            `json:"turn_start_time"`
	Winner        string               `json:"winner,omitempty"`
	mu            sync.RWMutex         `json:"-"`
}

// GameManager manages all active games
type GameManager struct {
	games map[string]*Game
	mu    sync.RWMutex
}

var (
	ErrGameNotFound     = errors.New("game not found")
	ErrGameFull         = errors.New("game is full")
	ErrGameStarted      = errors.New("game already started")
	ErrInvalidCode      = errors.New("invalid game code")
	ErrPlayerExists     = errors.New("player already in game")
	ErrNotPlayerTurn    = errors.New("not player's turn")
	ErrInvalidMove      = errors.New("invalid move")
)

// NewGameManager creates a new game manager
func NewGameManager() *GameManager {
	return &GameManager{
		games: make(map[string]*Game),
	}
}

// GenerateGameCode generates an 8-digit game code
func GenerateGameCode() string {
	code := rand.Intn(90000000) + 10000000 // Ensures 8 digits (10000000-99999999)
	return fmt.Sprintf("%08d", code)
}

// CreateGame creates a new game
func (gm *GameManager) CreateGame(maxPlayers int) (*Game, error) {
	if maxPlayers < 2 || maxPlayers > 5 {
		maxPlayers = 4 // Default to 4 players
	}

	gm.mu.Lock()
	defer gm.mu.Unlock()

	code := GenerateGameCode()
	// Ensure unique code
	for gm.games[code] != nil {
		code = GenerateGameCode()
	}

	game := &Game{
		Code:       code,
		Players:    make(map[string]*Player),
		State:      Waiting,
		MaxPlayers: maxPlayers,
		CreatedAt:  time.Now(),
	}

	gm.games[code] = game
	return game, nil
}

// GetGame retrieves a game by code
func (gm *GameManager) GetGame(code string) (*Game, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	game, exists := gm.games[code]
	if !exists {
		return nil, ErrGameNotFound
	}
	return game, nil
}

// JoinGame adds a player to a game
func (gm *GameManager) JoinGame(code, playerID, playerName string) (*Game, error) {
	game, err := gm.GetGame(code)
	if err != nil {
		return nil, err
	}

	game.mu.Lock()
	defer game.mu.Unlock()

	if game.State != Waiting {
		return nil, ErrGameStarted
	}

	if len(game.Players) >= game.MaxPlayers {
		return nil, ErrGameFull
	}

	if _, exists := game.Players[playerID]; exists {
		return nil, ErrPlayerExists
	}

	// Assign color based on join order
	colors := []PlayerColor{Red, Blue, Green, Yellow, Purple}
	color := colors[len(game.Players)%5]

	// Create pieces for the player
	pieces := make([]Piece, 4)
	for i := 0; i < 4; i++ {
		pieces[i] = Piece{
			ID:       i,
			Position: -1,
			IsHome:   true,
			IsSafe:   false,
		}
	}

	player := &Player{
		ID:     playerID,
		Name:   playerName,
		Color:  color,
		Pieces: pieces,
		Order:  len(game.Players),
	}

	game.Players[playerID] = player

	return game, nil
}

// StartGame starts a game
func (g *Game) StartGame() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != Waiting {
		return ErrGameStarted
	}

	if len(g.Players) < 2 {
		return errors.New("need at least 2 players to start")
	}

	g.State = Playing
	// Set first player as current turn
	for _, player := range g.Players {
		if player.Order == 0 {
			g.CurrentTurn = player.ID
			break
		}
	}
	g.TurnStartTime = time.Now()

	return nil
}

// RollDice simulates a dice roll
func (g *Game) RollDice() int {
	roll := rand.Intn(6) + 1
	g.mu.Lock()
	g.LastDiceRoll = roll
	g.mu.Unlock()
	return roll
}

// MovePiece moves a piece for a player
func (g *Game) MovePiece(playerID string, pieceID int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != Playing {
		return errors.New("game not in playing state")
	}

	if g.CurrentTurn != playerID {
		return ErrNotPlayerTurn
	}

	player, exists := g.Players[playerID]
	if !exists {
		return errors.New("player not found")
	}

	if pieceID < 0 || pieceID >= len(player.Pieces) {
		return errors.New("invalid piece ID")
	}

	piece := &player.Pieces[pieceID]

	// If piece is at home, can only move out with a 6
	if piece.IsHome && g.LastDiceRoll != 6 {
		return ErrInvalidMove
	}

	if piece.IsHome && g.LastDiceRoll == 6 {
		piece.IsHome = false
		piece.Position = 0
	} else {
		// Move piece forward
		piece.Position += g.LastDiceRoll
		// Simplified: if position > 51, piece reaches home (finished)
		if piece.Position > 51 {
			piece.Position = 100 + pieceID
			piece.IsSafe = true
		}
	}

	// Check if player won (all pieces finished)
	allFinished := true
	for _, p := range player.Pieces {
		if p.Position < 100 {
			allFinished = false
			break
		}
	}

	if allFinished {
		g.State = Ended
		g.Winner = playerID
		return nil
	}

	// Move to next player if didn't roll a 6
	if g.LastDiceRoll != 6 {
		g.nextTurn()
	}

	return nil
}

// nextTurn moves to the next player's turn
func (g *Game) nextTurn() {
	currentPlayer := g.Players[g.CurrentTurn]
	nextOrder := (currentPlayer.Order + 1) % len(g.Players)

	for _, player := range g.Players {
		if player.Order == nextOrder {
			g.CurrentTurn = player.ID
			g.TurnStartTime = time.Now()
			break
		}
	}
}

// GetGameState returns the current game state
func (g *Game) GetGameState() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return map[string]interface{}{
		"code":           g.Code,
		"players":        g.Players,
		"state":          g.State,
		"current_turn":   g.CurrentTurn,
		"max_players":    g.MaxPlayers,
		"last_dice_roll": g.LastDiceRoll,
		"winner":         g.Winner,
	}
}
