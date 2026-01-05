package models

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Initialize secure random seed on package load
func init() {
	var seed int64
	if err := binary.Read(crypto_rand.Reader, binary.BigEndian, &seed); err != nil {
		seed = time.Now().UnixNano()
	}
	rand.Seed(seed)
}

// PlayerColor represents the color of a player's pieces
type PlayerColor string

const (
	Red    PlayerColor = "red"
	Blue   PlayerColor = "blue"
	Green  PlayerColor = "green"
	Yellow PlayerColor = "yellow"
	Purple PlayerColor = "purple"
	Orange PlayerColor = "orange"
	Olive  PlayerColor = "olive"
	Indigo PlayerColor = "indigo"
)

// Game board constants
const (
	// Standard (square) board - 2-4 players
	BoardSize        = 52  // Total squares on the main board (0-51)
	BoardMaxPosition = 51  // Maximum position on the main board
	
	// Hexagonal board - 5-6 players
	HexBoardSize        = 72  // Total squares on hexagonal board (6 arms × 12)
	HexBoardMaxPosition = 71  // Maximum position on hex board
	
	HomeStretchSize  = 6   // Each player has 6 home stretch squares
	FinishPosition   = 100 // Position indicating piece has finished
	PiecesPerPlayer  = 4   // Number of pieces each player has
	HomePosition     = -1  // Position indicating piece is at home
)

// Timeout and cleanup constants
const (
	DefaultTurnTimeout   = 60 * time.Second  // Time allowed per turn
	DefaultGameTTL       = 24 * time.Hour    // Time before abandoned game is cleaned up
	DefaultInactivityTTL = 30 * time.Minute  // Time before inactive game is cleaned up
	CleanupInterval      = 5 * time.Minute   // How often to run cleanup
	TurnTimeoutWarning   = 10 * time.Second  // Warning before timeout
)

// Validation constants
const (
	MinPlayerNameLength = 1
	MaxPlayerNameLength = 30
	MinPlayerIDLength   = 1
	MaxPlayerIDLength   = 64
	MaxConsecutiveSixes = 3   // Rolling 3 sixes in a row forfeits turn
	MaxChatMessageLen   = 500 // Max chat message length
)

// Validation regex for player IDs
var playerIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Player start positions on the main board (where they enter after rolling 6)
// Square board (2-4 players)
var PlayerStartPositions = map[PlayerColor]int{
	Red:    0,
	Blue:   13,
	Green:  26,
	Yellow: 39,
}

// Hexagonal board start positions (5-6 players)
// Colors clockwise from bottom: Blue, Red, Green, Purple, Olive, Indigo
// Each arm is 12 positions, players start at beginning of each arm
var HexPlayerStartPositions = map[PlayerColor]int{
	Blue:   0,   // Arm 0 (bottom) - Player 2
	Red:    12,  // Arm 1 (bottom-right) - Player 1
	Green:  24,  // Arm 2 (right)
	Purple: 36,  // Arm 3 (top-right) - Player 5
	Olive:  48,  // Arm 4 (top-left) - Player 4
	Indigo: 60,  // Arm 5 (left) - Player 3
}

// Position where each player enters their home stretch (last position on main board)
// Square board (2-4 players)
var PlayerHomeStretchEntry = map[PlayerColor]int{
	Red:    50,
	Blue:   11,
	Green:  24,
	Yellow: 37,
}

// Hex board home stretch entry positions
// Home stretch entry is 2 positions before own start (going backwards on track)
var HexPlayerHomeStretchEntry = map[PlayerColor]int{
	Blue:   70,  // 72 - 2 = 70 (before position 0)
	Red:    10,  // 12 - 2 = 10
	Green:  22,  // 24 - 2 = 22
	Purple: 34,  // 36 - 2 = 34
	Olive:  46,  // 48 - 2 = 46
	Indigo: 58,  // 60 - 2 = 58
}

// Safe zones - positions where pieces cannot be captured
// Square board safe zones
var SafeZones = map[int]bool{
	0: true, 8: true, 13: true, 21: true, 26: true, 34: true, 39: true, 47: true,
}

// Hexagonal board safe zones (start positions + one more per arm)
var HexSafeZones = map[int]bool{
	0: true, 3: true, 12: true, 15: true, 24: true, 27: true,
	36: true, 39: true, 48: true, 51: true, 60: true, 63: true,
}

// GetBoardSize returns the board size based on max players
func GetBoardSize(maxPlayers int) int {
	if maxPlayers >= 5 {
		return HexBoardSize
	}
	return BoardSize
}

// GetBoardMaxPosition returns the max board position based on max players
func GetBoardMaxPosition(maxPlayers int) int {
	if maxPlayers >= 5 {
		return HexBoardMaxPosition
	}
	return BoardMaxPosition
}

// GetStartPosition returns the start position for a color based on board type
func GetStartPosition(color PlayerColor, maxPlayers int) int {
	if maxPlayers >= 5 {
		return HexPlayerStartPositions[color]
	}
	return PlayerStartPositions[color]
}

// GetHomeStretchEntry returns the home stretch entry position for a color based on board type
func GetHomeStretchEntry(color PlayerColor, maxPlayers int) int {
	if maxPlayers >= 5 {
		return HexPlayerHomeStretchEntry[color]
	}
	return PlayerHomeStretchEntry[color]
}

// IsSafeZone checks if a position is a safe zone based on board type
func IsSafeZone(position int, maxPlayers int) bool {
	if maxPlayers >= 5 {
		return HexSafeZones[position]
	}
	return SafeZones[position]
}

// Piece represents a single game piece
type Piece struct {
	ID                  int  `json:"id"`
	Position            int  `json:"position"`              // -1 for home, 0-51 for main board, 100+ for finished
	HomeStretchPosition int  `json:"home_stretch_position"` // 0 = not in home stretch, 1-6 = position in home stretch
	IsHome              bool `json:"is_home"`
	IsSafe              bool `json:"is_safe"`
	IsFinished          bool `json:"is_finished"`
}

// Player represents a player in the game
type Player struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Color        PlayerColor `json:"color"`
	Pieces       []Piece     `json:"pieces"`
	Order        int         `json:"order"`         // Turn order (randomized at start)
	LastActivity time.Time   `json:"last_activity"` // Last activity timestamp
	IsReady      bool        `json:"is_ready"`      // Ready to start
	IsHost       bool        `json:"is_host"`       // Is game host
	IsBot        bool        `json:"is_bot"`        // Is AI player
}

// Spectator represents someone watching the game
type Spectator struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	LastActivity time.Time `json:"last_activity"`
}

// MoveRecord represents a move in game history
type MoveRecord struct {
	PlayerID    string    `json:"player_id"`
	PlayerName  string    `json:"player_name"`
	PieceID     int       `json:"piece_id"`
	DiceRoll    int       `json:"dice_roll"`
	FromPos     int       `json:"from_pos"`
	ToPos       int       `json:"to_pos"`
	WasCapture  bool      `json:"was_capture"`
	WasFromHome bool      `json:"was_from_home"`
	CapturedPID string    `json:"captured_player_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	PlayerID    string    `json:"player_id"`
	PlayerName  string    `json:"player_name"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	IsSpectator bool      `json:"is_spectator"`
}

// GameState represents the current state of the game
type GameState string

const (
	Waiting GameState = "waiting" // Waiting for players to join
	Playing GameState = "playing" // Game in progress
	Paused  GameState = "paused"  // Game is paused
	Ended   GameState = "ended"   // Game has ended
)

// Game represents a Ludo game session
type Game struct {
	Code              string                `json:"code"`
	Players           map[string]*Player    `json:"players"`
	Spectators        map[string]*Spectator `json:"spectators"`
	State             GameState             `json:"state"`
	CurrentTurn       string                `json:"current_turn"`
	MaxPlayers        int                   `json:"max_players"`
	CreatedAt         time.Time             `json:"created_at"`
	LastDiceRoll      int                   `json:"last_dice_roll"`
	HasRolled         bool                  `json:"has_rolled"`
	TurnStartTime     time.Time             `json:"turn_start_time"`
	LastActivity      time.Time             `json:"last_activity"`
	TurnTimeout       time.Duration         `json:"-"`
	Winner            string                `json:"winner,omitempty"`
	ConsecutiveSixes  int                   `json:"consecutive_sixes"`
	HostID            string                `json:"host_id"`
	MoveHistory       []MoveRecord          `json:"move_history,omitempty"`
	ChatMessages      []ChatMessage         `json:"chat_messages,omitempty"`
	PausedBy          string                `json:"paused_by,omitempty"`
	PausedAt          time.Time             `json:"paused_at,omitempty"`
	CaptureGrantsTurn bool                  `json:"capture_grants_turn"`
	mu                sync.RWMutex          `json:"-"`
}

// GameManager manages all active games
type GameManager struct {
	games map[string]*Game
	mu    sync.RWMutex
}

var (
	ErrGameNotFound       = errors.New("game not found")
	ErrGameFull           = errors.New("game is full")
	ErrGameStarted        = errors.New("game already started")
	ErrGamePaused         = errors.New("game is paused")
	ErrGameNotPaused      = errors.New("game is not paused")
	ErrInvalidCode        = errors.New("invalid game code")
	ErrPlayerExists       = errors.New("player already in game")
	ErrNotPlayerTurn      = errors.New("not player's turn")
	ErrInvalidMove        = errors.New("invalid move")
	ErrTurnTimeout        = errors.New("turn timeout")
	ErrNotHost            = errors.New("only host can perform this action")
	ErrPlayersNotReady    = errors.New("not all players are ready")
	ErrInvalidPlayerName  = errors.New("invalid player name")
	ErrInvalidPlayerID    = errors.New("invalid player ID")
	ErrMustRollFirst      = errors.New("must roll dice before moving")
	ErrAlreadyRolled      = errors.New("already rolled this turn")
	ErrThreeSixes         = errors.New("three consecutive sixes - loss of turn")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrCannotKickSelf     = errors.New("cannot kick yourself")
	ErrChatTooLong        = errors.New("chat message too long")
	ErrNotEnoughPlayers   = errors.New("need at least 2 players to start")
)

// ValidatePlayerName validates a player name
func ValidatePlayerName(name string) error {
	name = strings.TrimSpace(name)
	length := utf8.RuneCountInString(name)
	if length < MinPlayerNameLength || length > MaxPlayerNameLength {
		return ErrInvalidPlayerName
	}
	return nil
}

// ValidatePlayerID validates a player ID
func ValidatePlayerID(id string) error {
	if len(id) < MinPlayerIDLength || len(id) > MaxPlayerIDLength {
		return ErrInvalidPlayerID
	}
	if !playerIDRegex.MatchString(id) {
		return ErrInvalidPlayerID
	}
	return nil
}

// SecureRollDice generates a cryptographically secure dice roll
func SecureRollDice() int {
	var b [1]byte
	for {
		crypto_rand.Read(b[:])
		if b[0] < 252 { // Rejection sampling to avoid bias (252 is divisible by 6)
			return int(b[0]%6) + 1
		}
	}
}

// NewGameManager creates a new game manager
func NewGameManager() *GameManager {
	return &GameManager{
		games: make(map[string]*Game),
	}
}

// GenerateGameCode generates an 8-digit game code using secure random
func GenerateGameCode() string {
	var b [4]byte
	crypto_rand.Read(b[:])
	code := binary.BigEndian.Uint32(b[:])%90000000 + 10000000
	return fmt.Sprintf("%08d", code)
}

// CreateGame creates a new game with host
func (gm *GameManager) CreateGame(hostID, hostName string, maxPlayers int) (*Game, error) {
	// Validate inputs
	if err := ValidatePlayerID(hostID); err != nil {
		return nil, err
	}
	if err := ValidatePlayerName(hostName); err != nil {
		return nil, err
	}

	if maxPlayers < 2 || maxPlayers > 6 {
		maxPlayers = 4 // Default to 4 players
	}

	gm.mu.Lock()
	defer gm.mu.Unlock()

	code := GenerateGameCode()
	// Ensure unique code
	for gm.games[code] != nil {
		code = GenerateGameCode()
	}

	// Create pieces for host
	pieces := make([]Piece, PiecesPerPlayer)
	for i := 0; i < PiecesPerPlayer; i++ {
		pieces[i] = Piece{
			ID:       i,
			Position: HomePosition,
			IsHome:   true,
		}
	}

	host := &Player{
		ID:           hostID,
		Name:         strings.TrimSpace(hostName),
		Color:        Red,
		Pieces:       pieces,
		Order:        0,
		LastActivity: time.Now(),
		IsReady:      false,
		IsHost:       true,
	}

	game := &Game{
		Code:              code,
		Players:           map[string]*Player{hostID: host},
		Spectators:        make(map[string]*Spectator),
		State:             Waiting,
		MaxPlayers:        maxPlayers,
		CreatedAt:         time.Now(),
		LastActivity:      time.Now(),
		TurnTimeout:       DefaultTurnTimeout,
		HostID:            hostID,
		MoveHistory:       []MoveRecord{},
		ChatMessages:      []ChatMessage{},
		CaptureGrantsTurn: true,
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
	// Validate inputs
	if err := ValidatePlayerID(playerID); err != nil {
		return nil, err
	}
	if err := ValidatePlayerName(playerName); err != nil {
		return nil, err
	}

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

	// Assign color based on join order and game type
	var color PlayerColor
	if game.MaxPlayers >= 5 {
		// Hex board colors (clockwise from bottom)
		hexColors := []PlayerColor{Blue, Red, Green, Purple, Olive, Indigo}
		color = hexColors[len(game.Players)%6]
	} else {
		// Square board colors
		squareColors := []PlayerColor{Red, Blue, Green, Yellow}
		color = squareColors[len(game.Players)%4]
	}

	// Create pieces for the player
	pieces := make([]Piece, PiecesPerPlayer)
	for i := 0; i < PiecesPerPlayer; i++ {
		pieces[i] = Piece{
			ID:                  i,
			Position:            HomePosition,
			HomeStretchPosition: 0,
			IsHome:              true,
			IsSafe:              false,
			IsFinished:          false,
		}
	}

	player := &Player{
		ID:           playerID,
		Name:         strings.TrimSpace(playerName),
		Color:        color,
		Pieces:       pieces,
		Order:        len(game.Players),
		LastActivity: time.Now(),
		IsReady:      false,
		IsHost:       false,
	}

	game.Players[playerID] = player
	game.LastActivity = time.Now()

	return game, nil
}

// Bot names for AI players
var botNames = []string{
	"Bot Alice", "Bot Bob", "Bot Charlie", "Bot Diana",
	"Bot Eve", "Bot Frank", "Bot Grace", "Bot Henry",
}

// AddBot adds an AI player to the game
func (gm *GameManager) AddBot(code, hostID string) (*Game, *Player, error) {
	game, err := gm.GetGame(code)
	if err != nil {
		return nil, nil, err
	}

	game.mu.Lock()
	defer game.mu.Unlock()

	// Only host can add bots
	if game.HostID != hostID {
		return nil, nil, ErrNotHost
	}

	if game.State != Waiting {
		return nil, nil, ErrGameStarted
	}

	if len(game.Players) >= game.MaxPlayers {
		return nil, nil, ErrGameFull
	}

	// Generate unique bot ID
	botID := fmt.Sprintf("bot_%d_%d", time.Now().UnixNano(), len(game.Players))
	
	// Pick a bot name
	botName := botNames[len(game.Players)%len(botNames)]

	// Assign color based on join order and game type
	var color PlayerColor
	if game.MaxPlayers >= 5 {
		hexColors := []PlayerColor{Blue, Red, Green, Purple, Olive, Indigo}
		color = hexColors[len(game.Players)%6]
	} else {
		squareColors := []PlayerColor{Red, Blue, Green, Yellow}
		color = squareColors[len(game.Players)%4]
	}

	// Create pieces for the bot
	pieces := make([]Piece, PiecesPerPlayer)
	for i := 0; i < PiecesPerPlayer; i++ {
		pieces[i] = Piece{
			ID:                  i,
			Position:            HomePosition,
			HomeStretchPosition: 0,
			IsHome:              true,
			IsSafe:              false,
			IsFinished:          false,
		}
	}

	bot := &Player{
		ID:           botID,
		Name:         botName,
		Color:        color,
		Pieces:       pieces,
		Order:        len(game.Players),
		LastActivity: time.Now(),
		IsReady:      true, // Bots are always ready
		IsHost:       false,
		IsBot:        true,
	}

	game.Players[botID] = bot
	game.LastActivity = time.Now()

	return game, bot, nil
}

// RemoveBot removes an AI player from the game
func (gm *GameManager) RemoveBot(code, hostID, botID string) (*Game, error) {
	game, err := gm.GetGame(code)
	if err != nil {
		return nil, err
	}

	game.mu.Lock()
	defer game.mu.Unlock()

	// Only host can remove bots
	if game.HostID != hostID {
		return nil, ErrNotHost
	}

	if game.State != Waiting {
		return nil, ErrGameStarted
	}

	player, exists := game.Players[botID]
	if !exists {
		return nil, ErrPlayerNotFound
	}

	if !player.IsBot {
		return nil, errors.New("player is not a bot")
	}

	delete(game.Players, botID)
	game.LastActivity = time.Now()

	return game, nil
}

// IsCurrentPlayerBot checks if the current turn player is a bot
func (g *Game) IsCurrentPlayerBot() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.State != Playing {
		return false
	}

	player, exists := g.Players[g.CurrentTurn]
	if !exists {
		return false
	}

	return player.IsBot
}

// GetBotMove returns a random valid move for the bot
func (g *Game) GetBotMove() (pieceID int, hasMove bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.State != Playing || !g.HasRolled {
		return -1, false
	}

	player, exists := g.Players[g.CurrentTurn]
	if !exists || !player.IsBot {
		return -1, false
	}

	// Get valid moves
	validMoves := g.getValidMovesInternal(g.CurrentTurn)
	if len(validMoves) == 0 {
		return -1, false
	}

	// Pick a random valid move
	return validMoves[rand.Intn(len(validMoves))], true
}

// JoinAsSpectator adds a spectator to the game
func (gm *GameManager) JoinAsSpectator(code, spectatorID, spectatorName string) (*Game, error) {
	if err := ValidatePlayerID(spectatorID); err != nil {
		return nil, err
	}
	if err := ValidatePlayerName(spectatorName); err != nil {
		return nil, err
	}

	game, err := gm.GetGame(code)
	if err != nil {
		return nil, err
	}

	game.mu.Lock()
	defer game.mu.Unlock()

	// Check if already a player
	if _, exists := game.Players[spectatorID]; exists {
		return nil, ErrPlayerExists
	}

	game.Spectators[spectatorID] = &Spectator{
		ID:           spectatorID,
		Name:         strings.TrimSpace(spectatorName),
		LastActivity: time.Now(),
	}

	return game, nil
}

// SetPlayerReady sets a player's ready status
func (g *Game) SetPlayerReady(playerID string, ready bool) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	player, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	player.IsReady = ready
	g.LastActivity = time.Now()
	return nil
}

// AreAllPlayersReady checks if all players are ready
func (g *Game) AreAllPlayersReady() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, player := range g.Players {
		if !player.IsReady {
			return false
		}
	}
	return true
}

// KickPlayer removes a player from the game (host only)
func (g *Game) KickPlayer(hostID, playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.HostID != hostID {
		return ErrNotHost
	}

	if hostID == playerID {
		return ErrCannotKickSelf
	}

	if g.State != Waiting {
		return ErrGameStarted
	}

	if _, exists := g.Players[playerID]; !exists {
		return ErrPlayerNotFound
	}

	delete(g.Players, playerID)
	g.LastActivity = time.Now()

	// Reassign colors and orders
	order := 0
	var colors []PlayerColor
	if g.MaxPlayers >= 5 {
		colors = []PlayerColor{Blue, Red, Green, Purple, Olive, Indigo}
	} else {
		colors = []PlayerColor{Red, Blue, Green, Yellow}
	}
	for _, player := range g.Players {
		player.Order = order
		player.Color = colors[order%len(colors)]
		order++
	}

	return nil
}

// LeaveGame allows a player to leave
func (g *Game) LeaveGame(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	player, exists := g.Players[playerID]
	if !exists {
		// Check spectators
		if _, specExists := g.Spectators[playerID]; specExists {
			delete(g.Spectators, playerID)
			return nil
		}
		return ErrPlayerNotFound
	}

	if g.State == Waiting {
		wasHost := player.IsHost
		delete(g.Players, playerID)

		// Transfer host if needed
		if wasHost && len(g.Players) > 0 {
			for _, p := range g.Players {
				p.IsHost = true
				g.HostID = p.ID
				break
			}
		}

		// Reassign orders
		order := 0
		var colors []PlayerColor
		if g.MaxPlayers >= 5 {
			colors = []PlayerColor{Blue, Red, Green, Purple, Olive, Indigo}
		} else {
			colors = []PlayerColor{Red, Blue, Green, Yellow}
		}
		for _, p := range g.Players {
			p.Order = order
			p.Color = colors[order%len(colors)]
			order++
		}
	} else if g.State == Playing {
		// If leaving player's turn, move to next
		if g.CurrentTurn == playerID {
			g.nextTurn()
		}
	}

	g.LastActivity = time.Now()
	return nil
}

// StartGame starts a game (host only, all players must be ready)
func (g *Game) StartGame(hostID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.HostID != hostID {
		return ErrNotHost
	}

	if g.State != Waiting {
		return ErrGameStarted
	}

	if len(g.Players) < 2 {
		return ErrNotEnoughPlayers
	}

	// Check all players ready
	for _, player := range g.Players {
		if !player.IsReady {
			return ErrPlayersNotReady
		}
	}

	// Randomize turn order
	g.randomizeTurnOrder()

	g.State = Playing
	// Set first player (order 0) as current turn
	for _, player := range g.Players {
		if player.Order == 0 {
			g.CurrentTurn = player.ID
			break
		}
	}
	g.TurnStartTime = time.Now()
	g.HasRolled = false
	g.ConsecutiveSixes = 0
	g.LastActivity = time.Now()

	return nil
}

// randomizeTurnOrder shuffles player turn order
func (g *Game) randomizeTurnOrder() {
	playerIDs := make([]string, 0, len(g.Players))
	for id := range g.Players {
		playerIDs = append(playerIDs, id)
	}

	// Fisher-Yates shuffle
	for i := len(playerIDs) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		playerIDs[i], playerIDs[j] = playerIDs[j], playerIDs[i]
	}

	for order, id := range playerIDs {
		g.Players[id].Order = order
	}
}

// PauseGame pauses the game
func (g *Game) PauseGame(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != Playing {
		return errors.New("can only pause a playing game")
	}

	g.State = Paused
	g.PausedBy = playerID
	g.PausedAt = time.Now()
	g.LastActivity = time.Now()

	return nil
}

// ResumeGame resumes a paused game
func (g *Game) ResumeGame(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != Paused {
		return ErrGameNotPaused
	}

	// Extend turn time by pause duration
	pauseDuration := time.Since(g.PausedAt)
	g.TurnStartTime = g.TurnStartTime.Add(pauseDuration)

	g.State = Playing
	g.PausedBy = ""
	g.LastActivity = time.Now()

	return nil
}

// RollDice simulates a secure dice roll
func (g *Game) RollDice(playerID string) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State == Paused {
		return 0, ErrGamePaused
	}

	if g.State != Playing {
		return 0, errors.New("game not in playing state")
	}

	if g.CurrentTurn != playerID {
		return 0, ErrNotPlayerTurn
	}

	if g.HasRolled {
		return 0, ErrAlreadyRolled
	}

	roll := SecureRollDice()
	g.LastDiceRoll = roll
	g.HasRolled = true
	g.LastActivity = time.Now()

	// Track consecutive sixes
	if roll == 6 {
		g.ConsecutiveSixes++
		if g.ConsecutiveSixes >= MaxConsecutiveSixes {
			// Three sixes - loss of turn
			g.ConsecutiveSixes = 0
			g.HasRolled = false
			g.nextTurn()
			return roll, ErrThreeSixes
		}
	} else {
		g.ConsecutiveSixes = 0
	}

	return roll, nil
}

// MovePiece moves a piece for a player
func (g *Game) MovePiece(playerID string, pieceID int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State == Paused {
		return ErrGamePaused
	}

	if g.State != Playing {
		return errors.New("game not in playing state")
	}

	if g.CurrentTurn != playerID {
		return ErrNotPlayerTurn
	}

	if !g.HasRolled {
		return ErrMustRollFirst
	}

	player, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	if pieceID < 0 || pieceID >= len(player.Pieces) {
		return errors.New("invalid piece ID")
	}

	piece := &player.Pieces[pieceID]
	oldPosition := piece.Position
	wasHome := piece.IsHome
	wasHomeStretch := piece.HomeStretchPosition

	// Cannot move a finished piece
	if piece.IsFinished {
		return ErrInvalidMove
	}

	// If piece is at home, can only move out with a 6
	if piece.IsHome && g.LastDiceRoll != 6 {
		return ErrInvalidMove
	}

	captured := false

	if piece.IsHome && g.LastDiceRoll == 6 {
		// Move piece out of home to player's start position
		piece.IsHome = false
		piece.Position = GetStartPosition(player.Color, g.MaxPlayers)
		piece.IsSafe = true // Start position is always safe
	} else if piece.HomeStretchPosition > 0 {
		// Piece is in home stretch - move within home stretch
		newHomeStretchPos := piece.HomeStretchPosition + g.LastDiceRoll
		if newHomeStretchPos > HomeStretchSize {
			// Exact roll required to finish - bounce back
			return ErrInvalidMove
		} else if newHomeStretchPos == HomeStretchSize {
			// Piece finished!
			piece.HomeStretchPosition = HomeStretchSize
			piece.Position = FinishPosition + pieceID
			piece.IsFinished = true
			piece.IsSafe = true
		} else {
			piece.HomeStretchPosition = newHomeStretchPos
			piece.IsSafe = true // Always safe in home stretch
		}
	} else {
		// Piece is on main board - calculate new position
		newPosition, enteredHomeStretch, homeStretchPos := g.calculateNewPosition(player.Color, piece.Position, g.LastDiceRoll)

		if enteredHomeStretch {
			if homeStretchPos > HomeStretchSize {
				// Overshot - cannot make this move (exact roll required)
				return ErrInvalidMove
			} else if homeStretchPos == HomeStretchSize {
				// Piece finished!
				piece.Position = FinishPosition + pieceID
				piece.HomeStretchPosition = HomeStretchSize
				piece.IsFinished = true
				piece.IsSafe = true
			} else {
				// Entered home stretch
				piece.Position = -2 // Special value indicating in home stretch
				piece.HomeStretchPosition = homeStretchPos
				piece.IsSafe = true
			}
		} else {
			piece.Position = newPosition
			piece.IsSafe = IsSafeZone(newPosition, g.MaxPlayers)

			// Check for captures - only if not on safe zone
			if !piece.IsSafe {
				captured = g.checkAndCapture(playerID, newPosition)
			}
		}
	}

	// Record move in history
	moveRecord := MoveRecord{
		PlayerID:    playerID,
		PieceID:     pieceID,
		FromPos:     oldPosition,
		ToPos:       piece.Position,
		DiceRoll:    g.LastDiceRoll,
		WasCapture:  captured,
		Timestamp:   time.Now(),
		WasFromHome: wasHome,
	}
	if wasHomeStretch > 0 {
		moveRecord.FromPos = -wasHomeStretch // Encode home stretch as negative
	}
	g.MoveHistory = append(g.MoveHistory, moveRecord)

	// Check if player won (all pieces finished)
	allFinished := true
	for _, p := range player.Pieces {
		if !p.IsFinished {
			allFinished = false
			break
		}
	}

	if allFinished {
		g.State = Ended
		g.Winner = playerID
		g.HasRolled = false
		return nil
	}

	g.LastActivity = time.Now()
	g.HasRolled = false // Reset for next roll/turn

	// Determine next turn
	// Extra turn if: rolled 6 (and not 3 sixes), or captured a piece (if enabled)
	extraTurn := g.LastDiceRoll == 6
	if captured && g.CaptureGrantsTurn {
		extraTurn = true
	}

	if !extraTurn {
		g.ConsecutiveSixes = 0
		g.nextTurn()
	}

	return nil
}

// calculateNewPosition calculates the new position for a piece moving on the main board
// Returns: (newPosition, enteredHomeStretch, homeStretchPosition)
func (g *Game) calculateNewPosition(color PlayerColor, currentPos, diceRoll int) (int, bool, int) {
	homeStretchEntry := GetHomeStretchEntry(color, g.MaxPlayers)
	boardSize := GetBoardSize(g.MaxPlayers)

	// Calculate steps needed to reach home stretch entry from current position
	var stepsToEntry int
	if currentPos <= homeStretchEntry {
		stepsToEntry = homeStretchEntry - currentPos
	} else {
		// Need to wrap around the board
		stepsToEntry = (boardSize - currentPos) + homeStretchEntry
	}

	// Special case: if current position is at or past home stretch entry relative to start
	// we need to check if we've completed a full lap
	hasCompletedLap := g.hasCompletedLap(color, currentPos)

	if hasCompletedLap && diceRoll > stepsToEntry {
		// Enter home stretch
		homeStretchPos := diceRoll - stepsToEntry
		return -2, true, homeStretchPos
	} else if hasCompletedLap && diceRoll == stepsToEntry {
		// Land exactly on home stretch entry, then enter home stretch at position 0
		// Actually, landing on entry means entering position 1 of home stretch? 
		// No - landing exactly means you're at the entry, next roll enters home stretch
		// Let's treat entry as the last square before home stretch
		return homeStretchEntry, false, 0
	}

	// Normal movement on main board
	newPos := (currentPos + diceRoll) % boardSize
	return newPos, false, 0
}

// hasCompletedLap checks if a piece has traveled far enough to enter home stretch
// A piece must pass its start position to be eligible for home stretch
func (g *Game) hasCompletedLap(color PlayerColor, currentPos int) bool {
	startPos := GetStartPosition(color, g.MaxPlayers)
	homeStretchEntry := GetHomeStretchEntry(color, g.MaxPlayers)

	// For simplicity, we consider a piece eligible for home stretch if:
	// - It's past the start position (has traveled some distance)
	// - For Red (start=0, entry=50): positions 1-50 are eligible
	// - For Blue (start=13, entry=11): positions 14-51 and 0-11 are eligible
	// The logic is: current position is "between" start and entry going forward

	if startPos <= homeStretchEntry {
		// Simple case: start is before entry on the number line
		// e.g., Red: start=0, entry=50
		return currentPos > startPos || currentPos <= homeStretchEntry
	} else {
		// Wrap case: entry comes before start on the number line
		// e.g., Blue: start=13, entry=11 - valid range is 14-51 and 0-11
		return currentPos > startPos || currentPos <= homeStretchEntry
	}
}

// checkAndCapture checks if landing on a position captures any opponent pieces
// Returns true if at least one capture occurred
func (g *Game) checkAndCapture(currentPlayerID string, position int) bool {
	captured := false
	for playerID, player := range g.Players {
		if playerID == currentPlayerID {
			continue // Don't capture own pieces
		}
		for i := range player.Pieces {
			piece := &player.Pieces[i]
			// Capture if piece is on same position, not in home stretch, not finished, and not at home
			if piece.Position == position && !piece.IsHome && !piece.IsFinished && piece.HomeStretchPosition == 0 {
				// Send piece back home
				piece.Position = HomePosition
				piece.IsHome = true
				piece.IsSafe = false
				piece.HomeStretchPosition = 0
				captured = true
			}
		}
	}
	return captured
}

// nextTurn moves to the next player's turn
func (g *Game) nextTurn() {
	currentPlayer := g.Players[g.CurrentTurn]
	nextOrder := (currentPlayer.Order + 1) % len(g.Players)

	// Simple round-robin - find player with next order
	for _, player := range g.Players {
		if player.Order == nextOrder {
			g.CurrentTurn = player.ID
			g.TurnStartTime = time.Now()
			g.HasRolled = false
			return
		}
	}
}

// SendChatMessage adds a chat message to the game
func (g *Game) SendChatMessage(playerID, message string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	player, exists := g.Players[playerID]
	if !exists {
		// Check if spectator
		if spec, specExists := g.Spectators[playerID]; specExists {
			if len(message) > MaxChatMessageLen {
				return ErrChatTooLong
			}
			g.ChatMessages = append(g.ChatMessages, ChatMessage{
				PlayerID:    playerID,
				PlayerName:  spec.Name,
				Message:     strings.TrimSpace(message),
				Timestamp:   time.Now(),
				IsSpectator: true,
			})
			return nil
		}
		return ErrPlayerNotFound
	}

	if len(message) > MaxChatMessageLen {
		return ErrChatTooLong
	}

	g.ChatMessages = append(g.ChatMessages, ChatMessage{
		PlayerID:   playerID,
		PlayerName: player.Name,
		Message:    strings.TrimSpace(message),
		Timestamp:  time.Now(),
		IsSpectator: false,
	})
	g.LastActivity = time.Now()
	return nil
}

// GetRecentChat returns the most recent chat messages
func (g *Game) GetRecentChat(limit int) []ChatMessage {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if limit <= 0 || limit > len(g.ChatMessages) {
		return g.ChatMessages
	}
	return g.ChatMessages[len(g.ChatMessages)-limit:]
}

// HasValidMoves checks if the current player has any valid moves with the current dice roll
func (g *Game) HasValidMoves(playerID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	player, exists := g.Players[playerID]
	if !exists {
		return false
	}

	for _, piece := range player.Pieces {
		if piece.IsFinished {
			continue
		}

		// Check if piece at home can move (requires 6)
		if piece.IsHome {
			if g.LastDiceRoll == 6 {
				return true
			}
			continue
		}

		// Check if piece in home stretch can move
		if piece.HomeStretchPosition > 0 {
			newPos := piece.HomeStretchPosition + g.LastDiceRoll
			if newPos <= HomeStretchSize {
				return true
			}
			continue
		}

		// Check if piece on main board can move
		_, enteredHomeStretch, homeStretchPos := g.calculateNewPosition(player.Color, piece.Position, g.LastDiceRoll)
		if enteredHomeStretch {
			if homeStretchPos <= HomeStretchSize {
				return true
			}
		} else {
			return true // Can always move on main board if not entering home stretch
		}
	}

	return false
}

// SkipTurn skips the current player's turn (used when no valid moves available)
func (g *Game) SkipTurn(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State == Paused {
		return ErrGamePaused
	}

	if g.State != Playing {
		return errors.New("game not in playing state")
	}

	if g.CurrentTurn != playerID {
		return ErrNotPlayerTurn
	}

	if !g.HasRolled {
		return ErrMustRollFirst
	}

	g.HasRolled = false
	g.ConsecutiveSixes = 0
	g.nextTurn()
	return nil
}

// GetValidMoves returns a list of piece IDs that can be moved with the current dice roll
func (g *Game) GetValidMoves(playerID string) []int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.getValidMovesInternal(playerID)
}

// getValidMovesInternal returns valid moves without locking (caller must hold lock)
func (g *Game) getValidMovesInternal(playerID string) []int {
	player, exists := g.Players[playerID]
	if !exists {
		return nil
	}

	validPieces := []int{}

	for _, piece := range player.Pieces {
		if piece.IsFinished {
			continue
		}

		// Check if piece at home can move (requires 6)
		if piece.IsHome {
			if g.LastDiceRoll == 6 {
				validPieces = append(validPieces, piece.ID)
			}
			continue
		}

		// Check if piece in home stretch can move
		if piece.HomeStretchPosition > 0 {
			newPos := piece.HomeStretchPosition + g.LastDiceRoll
			if newPos <= HomeStretchSize {
				validPieces = append(validPieces, piece.ID)
			}
			continue
		}

		// Check if piece on main board can move
		_, enteredHomeStretch, homeStretchPos := g.calculateNewPosition(player.Color, piece.Position, g.LastDiceRoll)
		if enteredHomeStretch {
			if homeStretchPos <= HomeStretchSize {
				validPieces = append(validPieces, piece.ID)
			}
		} else {
			validPieces = append(validPieces, piece.ID)
		}
	}

	return validPieces
}

// GetGameState returns the current game state
func (g *Game) GetGameState() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return map[string]interface{}{
		"code":               g.Code,
		"players":            g.Players,
		"spectators":         g.Spectators,
		"state":              g.State,
		"current_turn":       g.CurrentTurn,
		"max_players":        g.MaxPlayers,
		"last_dice_roll":     g.LastDiceRoll,
		"has_rolled":         g.HasRolled,
		"winner":             g.Winner,
		"turn_start_time":    g.TurnStartTime,
		"last_activity":      g.LastActivity,
		"consecutive_sixes":  g.ConsecutiveSixes,
		"host_id":            g.HostID,
		"paused_by":          g.PausedBy,
		"capture_grants_turn": g.CaptureGrantsTurn,
	}
}

// UpdateActivity updates the last activity timestamp for the game
func (g *Game) UpdateActivity() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.LastActivity = time.Now()
}

// IsTurnTimedOut checks if the current turn has exceeded the timeout
func (g *Game) IsTurnTimedOut() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.State != Playing || g.TurnStartTime.IsZero() {
		return false
	}
	return time.Since(g.TurnStartTime) > g.TurnTimeout
}

// GetTurnTimeRemaining returns the time remaining for the current turn
func (g *Game) GetTurnTimeRemaining() time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.State != Playing || g.TurnStartTime.IsZero() {
		return g.TurnTimeout
	}
	remaining := g.TurnTimeout - time.Since(g.TurnStartTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ForceSkipTurn forces the current player's turn to be skipped (used for timeout)
// Returns empty string if turn was not skipped (game not playing or turn not actually timed out)
func (g *Game) ForceSkipTurn() (skippedPlayerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != Playing {
		return ""
	}

	// Double-check that the turn is actually timed out (prevents race conditions)
	if g.TurnStartTime.IsZero() || time.Since(g.TurnStartTime) <= g.TurnTimeout {
		return "" // Turn is not actually timed out, don't skip
	}

	skippedPlayerID = g.CurrentTurn
	g.HasRolled = false
	g.nextTurn()
	g.ConsecutiveSixes = 0 // Reset consecutive sixes on forced skip
	return skippedPlayerID
}

// Rematch resets the game for a rematch with the same players
func (g *Game) Rematch(hostID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.HostID != hostID {
		return ErrNotHost
	}

	if g.State != Ended {
		return errors.New("can only rematch an ended game")
	}

	// Reset all pieces to home
	for _, player := range g.Players {
		player.IsReady = false
		for i := range player.Pieces {
			player.Pieces[i] = Piece{
				ID:                  i,
				Position:            HomePosition,
				IsHome:              true,
				IsFinished:          false,
				IsSafe:              false,
				HomeStretchPosition: 0,
			}
		}
	}

	// Reset game state
	g.State = Waiting
	g.CurrentTurn = ""
	g.LastDiceRoll = 0
	g.HasRolled = false
	g.ConsecutiveSixes = 0
	g.Winner = ""
	g.MoveHistory = []MoveRecord{}
	g.ChatMessages = []ChatMessage{}
	g.TurnStartTime = time.Time{}
	g.LastActivity = time.Now()

	return nil
}

// RemoveGame removes a game from the manager
func (gm *GameManager) RemoveGame(code string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	delete(gm.games, code)
}

// GetAllGames returns all games (for cleanup purposes)
func (gm *GameManager) GetAllGames() []*Game {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	games := make([]*Game, 0, len(gm.games))
	for _, game := range gm.games {
		games = append(games, game)
	}
	return games
}

// CleanupAbandonedGames removes games that have been inactive for too long
func (gm *GameManager) CleanupAbandonedGames() (removed []string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	now := time.Now()
	removed = []string{}

	for code, game := range gm.games {
		game.mu.RLock()
		shouldRemove := false

		// Remove ended games after inactivity period
		if game.State == Ended && now.Sub(game.LastActivity) > DefaultInactivityTTL {
			shouldRemove = true
		}

		// Remove waiting games that have been inactive
		if game.State == Waiting && now.Sub(game.LastActivity) > DefaultInactivityTTL {
			shouldRemove = true
		}

		// Remove any game that exceeds the maximum TTL
		if now.Sub(game.CreatedAt) > DefaultGameTTL {
			shouldRemove = true
		}

		// Remove games with no players that have been inactive
		if len(game.Players) == 0 && now.Sub(game.CreatedAt) > 5*time.Minute {
			shouldRemove = true
		}

		game.mu.RUnlock()

		if shouldRemove {
			delete(gm.games, code)
			removed = append(removed, code)
		}
	}

	return removed
}

// GetGameStats returns statistics about the game manager
func (gm *GameManager) GetGameStats() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	waiting := 0
	playing := 0
	ended := 0
	totalPlayers := 0

	for _, game := range gm.games {
		game.mu.RLock()
		switch game.State {
		case Waiting:
			waiting++
		case Playing:
			playing++
		case Ended:
			ended++
		}
		totalPlayers += len(game.Players)
		game.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_games":   len(gm.games),
		"waiting":       waiting,
		"playing":       playing,
		"ended":         ended,
		"total_players": totalPlayers,
	}
}
