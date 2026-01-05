package models

import (
	"testing"
)

func TestGenerateGameCode(t *testing.T) {
	code := GenerateGameCode()
	if len(code) != 8 {
		t.Errorf("Expected code length to be 8, got %d", len(code))
	}

	// Verify all characters are digits
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("Code contains non-digit character: %c", c)
		}
	}
}

func TestCreateGame(t *testing.T) {
	gm := NewGameManager()
	game, err := gm.CreateGame("host1", "Host", 4)

	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	if game.Code == "" {
		t.Error("Game code should not be empty")
	}

	if len(game.Code) != 8 {
		t.Errorf("Expected code length to be 8, got %d", len(game.Code))
	}

	if game.MaxPlayers != 4 {
		t.Errorf("Expected max players to be 4, got %d", game.MaxPlayers)
	}

	if game.State != Waiting {
		t.Errorf("Expected game state to be Waiting, got %s", game.State)
	}
	
	// Check host is automatically added
	if len(game.Players) != 1 {
		t.Errorf("Expected 1 player (host), got %d", len(game.Players))
	}
	
	if game.HostID != "host1" {
		t.Errorf("Expected host ID to be host1, got %s", game.HostID)
	}
}

func TestJoinGame(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 4)

	// First player joins
	joinedGame, err := gm.JoinGame(game.Code, "player1", "Alice")
	if err != nil {
		t.Fatalf("Failed to join game: %v", err)
	}

	if len(joinedGame.Players) != 2 { // host + 1 player
		t.Errorf("Expected 2 players (host + joined), got %d", len(joinedGame.Players))
	}

	player := joinedGame.Players["player1"]
	if player == nil {
		t.Fatal("Player not found in game")
	}

	if player.Name != "Alice" {
		t.Errorf("Expected player name to be Alice, got %s", player.Name)
	}

	// Host is Red (first player), so Alice should be Blue (second player)
	if player.Color != Blue {
		t.Errorf("Expected second player color to be Blue, got %s", player.Color)
	}

	if len(player.Pieces) != 4 {
		t.Errorf("Expected 4 pieces, got %d", len(player.Pieces))
	}

	// Check all pieces start at home
	for _, piece := range player.Pieces {
		if !piece.IsHome {
			t.Error("Piece should start at home")
		}
		if piece.Position != HomePosition {
			t.Errorf("Expected piece position to be HomePosition (-1), got %d", piece.Position)
		}
	}
}

func TestJoinGameFull(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2) // Max 2 players, host is already 1

	// Join one more player
	gm.JoinGame(game.Code, "player2", "Bob")

	// Try to join third player
	_, err := gm.JoinGame(game.Code, "player3", "Charlie")
	if err != ErrGameFull {
		t.Errorf("Expected ErrGameFull, got %v", err)
	}
}

func TestJoinGameDuplicate(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 4)

	gm.JoinGame(game.Code, "player1", "Alice")

	// Try to join with same player ID
	_, err := gm.JoinGame(game.Code, "player1", "Alice Again")
	if err != ErrPlayerExists {
		t.Errorf("Expected ErrPlayerExists, got %v", err)
	}
}

func TestStartGame(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 4)

	gm.JoinGame(game.Code, "player2", "Bob")
	
	// Set players ready
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)

	err := game.StartGame("host1")
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	if game.State != Playing {
		t.Errorf("Expected game state to be Playing, got %s", game.State)
	}

	if game.CurrentTurn == "" {
		t.Error("Current turn should be set")
	}
}

func TestStartGameNotEnoughPlayers(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 4)
	
	// Set host ready
	game.SetPlayerReady("host1", true)

	err := game.StartGame("host1")
	if err != ErrNotEnoughPlayers {
		t.Errorf("Expected ErrNotEnoughPlayers when starting game with only 1 player, got: %v", err)
	}
}

func TestRollDice(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 4)
	gm.JoinGame(game.Code, "player2", "Bob")
	
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	roll, err := game.RollDice(game.CurrentTurn)
	if err != nil {
		t.Fatalf("Failed to roll dice: %v", err)
	}

	if roll < 1 || roll > 6 {
		t.Errorf("Dice roll should be between 1 and 6, got %d", roll)
	}

	if game.LastDiceRoll != roll {
		t.Errorf("Last dice roll should be %d, got %d", roll, game.LastDiceRoll)
	}
}

func TestMovePiece(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	// Manually set dice roll to 6 to move piece out of home
	game.HasRolled = true
	game.LastDiceRoll = 6

	currentPlayerID := game.CurrentTurn
	err := game.MovePiece(currentPlayerID, 0)
	if err != nil {
		t.Fatalf("Failed to move piece: %v", err)
	}

	player := game.Players[currentPlayerID]
	piece := player.Pieces[0]

	if piece.IsHome {
		t.Error("Piece should no longer be at home")
	}

	// Piece should be at player's start position (depends on color)
	expectedStartPos := PlayerStartPositions[player.Color]
	if piece.Position != expectedStartPos {
		t.Errorf("Expected piece position to be %d (start for %s), got %d", expectedStartPos, player.Color, piece.Position)
	}
}

func TestMovePieceNotPlayerTurn(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	game.HasRolled = true
	game.LastDiceRoll = 6

	// Try to move as player who's not current turn
	var notCurrentPlayer string
	for id := range game.Players {
		if id != game.CurrentTurn {
			notCurrentPlayer = id
			break
		}
	}

	err := game.MovePiece(notCurrentPlayer, 0)
	if err != ErrNotPlayerTurn {
		t.Errorf("Expected ErrNotPlayerTurn, got %v", err)
	}
}

func TestGetGameState(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 4)

	gm.JoinGame(game.Code, "player1", "Alice")

	state := game.GetGameState()

	if state["code"] != game.Code {
		t.Error("Game state should contain game code")
	}

	if state["state"] != Waiting {
		t.Error("Game state should be Waiting")
	}

	if state["max_players"] != 4 {
		t.Error("Game state should contain max_players")
	}
}

// Tests for new game mechanics

func TestPlayerStartPositions(t *testing.T) {
	// Verify each color has correct start position
	expectedStarts := map[PlayerColor]int{
		Red:    0,
		Blue:   13,
		Green:  26,
		Yellow: 39,
	}

	for color, expected := range expectedStarts {
		actual := PlayerStartPositions[color]
		if actual != expected {
			t.Errorf("Expected %s start position to be %d, got %d", color, expected, actual)
		}
	}
}

func TestSafeZones(t *testing.T) {
	// All start positions and star squares should be safe
	expectedSafe := []int{0, 8, 13, 21, 26, 34, 39, 47}

	for _, pos := range expectedSafe {
		if !SafeZones[pos] {
			t.Errorf("Position %d should be a safe zone", pos)
		}
	}

	// Non-safe zone positions should not be safe
	nonSafe := []int{1, 5, 10, 15, 20, 25, 30}
	for _, pos := range nonSafe {
		if SafeZones[pos] {
			t.Errorf("Position %d should not be a safe zone", pos)
		}
	}
}

func TestPieceCapture(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	// Get player references
	var redPlayer, bluePlayer *Player
	for _, p := range game.Players {
		if p.Color == Red {
			redPlayer = p
		} else if p.Color == Blue {
			bluePlayer = p
		}
	}

	// Move red piece out of home to position 0
	game.CurrentTurn = redPlayer.ID
	game.HasRolled = true
	game.LastDiceRoll = 6
	game.MovePiece(redPlayer.ID, 0)

	// Move red piece to a non-safe position (e.g., position 5)
	redPlayer.Pieces[0].Position = 5
	redPlayer.Pieces[0].IsSafe = false

	// Place blue piece on the same position to trigger capture
	bluePlayer.Pieces[0].IsHome = false
	bluePlayer.Pieces[0].Position = 5
	bluePlayer.Pieces[0].IsSafe = false

	// Now move another red piece to position 5 to capture blue
	redPlayer.Pieces[1].IsHome = false
	redPlayer.Pieces[1].Position = 3
	game.CurrentTurn = redPlayer.ID
	game.HasRolled = true
	game.LastDiceRoll = 2

	err := game.MovePiece(redPlayer.ID, 1)
	if err != nil {
		t.Fatalf("Failed to move piece: %v", err)
	}

	// Blue piece should be sent back home
	if !bluePlayer.Pieces[0].IsHome {
		t.Error("Blue piece should be captured and sent back home")
	}
	if bluePlayer.Pieces[0].Position != HomePosition {
		t.Errorf("Captured piece position should be %d, got %d", HomePosition, bluePlayer.Pieces[0].Position)
	}
}

func TestNoCaptureOnSafeZone(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	var redPlayer, bluePlayer *Player
	for _, p := range game.Players {
		if p.Color == Red {
			redPlayer = p
		} else if p.Color == Blue {
			bluePlayer = p
		}
	}

	// Position 8 is a safe zone (star square)
	// Place blue piece on position 8
	bluePlayer.Pieces[0].IsHome = false
	bluePlayer.Pieces[0].Position = 8
	bluePlayer.Pieces[0].IsSafe = true

	// Move red piece to position 8
	redPlayer.Pieces[0].IsHome = false
	redPlayer.Pieces[0].Position = 6
	game.CurrentTurn = redPlayer.ID
	game.HasRolled = true
	game.LastDiceRoll = 2

	game.MovePiece(redPlayer.ID, 0)

	// Blue piece should NOT be captured (safe zone)
	if bluePlayer.Pieces[0].IsHome {
		t.Error("Blue piece should not be captured on safe zone")
	}
}

func TestHomeStretch(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	var redPlayer *Player
	for _, p := range game.Players {
		if p.Color == Red {
			redPlayer = p
			break
		}
	}

	// Red's home stretch entry is at position 50
	// Place piece at position 50 (home stretch entry)
	redPlayer.Pieces[0].IsHome = false
	redPlayer.Pieces[0].Position = 50

	game.CurrentTurn = redPlayer.ID
	game.HasRolled = true
	game.LastDiceRoll = 3

	err := game.MovePiece(redPlayer.ID, 0)
	if err != nil {
		t.Fatalf("Failed to move piece into home stretch: %v", err)
	}

	// Piece should be in home stretch
	if redPlayer.Pieces[0].HomeStretchPosition == 0 {
		t.Error("Piece should be in home stretch")
	}
	if !redPlayer.Pieces[0].IsSafe {
		t.Error("Piece should be safe in home stretch")
	}
}

func TestExactRollToFinish(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	var redPlayer *Player
	for _, p := range game.Players {
		if p.Color == Red {
			redPlayer = p
			break
		}
	}

	// Place piece in home stretch at position 4 (need 2 to finish, since HomeStretchSize = 6)
	redPlayer.Pieces[0].IsHome = false
	redPlayer.Pieces[0].Position = -2 // In home stretch
	redPlayer.Pieces[0].HomeStretchPosition = 4

	// Try to move with a 5 (overshoots)
	game.CurrentTurn = redPlayer.ID
	game.HasRolled = true
	game.LastDiceRoll = 5

	err := game.MovePiece(redPlayer.ID, 0)
	if err != ErrInvalidMove {
		t.Error("Should not be able to overshoot the finish")
	}

	// Move with exact roll (2)
	game.HasRolled = true
	game.LastDiceRoll = 2
	err = game.MovePiece(redPlayer.ID, 0)
	if err != nil {
		t.Fatalf("Failed to move with exact roll: %v", err)
	}

	if !redPlayer.Pieces[0].IsFinished {
		t.Error("Piece should be finished with exact roll")
	}
}

func TestHasValidMoves(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	// All pieces at home, roll 3 - should have no valid moves
	game.LastDiceRoll = 3
	if game.HasValidMoves(game.CurrentTurn) {
		t.Error("Should have no valid moves when all pieces at home and roll is not 6")
	}

	// Roll 6 - should have valid moves (can move piece out)
	game.LastDiceRoll = 6
	if !game.HasValidMoves(game.CurrentTurn) {
		t.Error("Should have valid moves when roll is 6 and pieces at home")
	}
}

func TestGetValidMoves(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	// All pieces at home, roll 6 - all 4 pieces can move out
	game.LastDiceRoll = 6
	validMoves := game.GetValidMoves(game.CurrentTurn)
	if len(validMoves) != 4 {
		t.Errorf("Expected 4 valid moves with roll 6 and all pieces at home, got %d", len(validMoves))
	}

	// Roll 3 - no valid moves
	game.LastDiceRoll = 3
	validMoves = game.GetValidMoves(game.CurrentTurn)
	if len(validMoves) != 0 {
		t.Errorf("Expected 0 valid moves with roll 3 and all pieces at home, got %d", len(validMoves))
	}
}

func TestSkipTurn(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	firstPlayer := game.CurrentTurn

	// Roll something other than 6 when all pieces at home
	game.HasRolled = true
	game.LastDiceRoll = 3

	err := game.SkipTurn(firstPlayer)
	if err != nil {
		t.Fatalf("Failed to skip turn: %v", err)
	}

	if game.CurrentTurn == firstPlayer {
		t.Error("Turn should have advanced to next player")
	}
}

func TestCannotMoveFinishedPiece(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame("host1", "Host", 2)

	gm.JoinGame(game.Code, "player2", "Bob")
	game.SetPlayerReady("host1", true)
	game.SetPlayerReady("player2", true)
	game.StartGame("host1")

	player := game.Players[game.CurrentTurn]

	// Mark piece as finished
	player.Pieces[0].IsFinished = true
	player.Pieces[0].Position = FinishPosition
	player.Pieces[0].HomeStretchPosition = HomeStretchSize

	game.HasRolled = true
	game.LastDiceRoll = 6

	err := game.MovePiece(game.CurrentTurn, 0)
	if err != ErrInvalidMove {
		t.Error("Should not be able to move a finished piece")
	}
}
