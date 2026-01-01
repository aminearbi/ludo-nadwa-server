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
	game, err := gm.CreateGame(4)

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
}

func TestJoinGame(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame(4)

	// First player joins
	joinedGame, err := gm.JoinGame(game.Code, "player1", "Alice")
	if err != nil {
		t.Fatalf("Failed to join game: %v", err)
	}

	if len(joinedGame.Players) != 1 {
		t.Errorf("Expected 1 player, got %d", len(joinedGame.Players))
	}

	player := joinedGame.Players["player1"]
	if player == nil {
		t.Fatal("Player not found in game")
	}

	if player.Name != "Alice" {
		t.Errorf("Expected player name to be Alice, got %s", player.Name)
	}

	if player.Color != Red {
		t.Errorf("Expected first player color to be Red, got %s", player.Color)
	}

	if len(player.Pieces) != 4 {
		t.Errorf("Expected 4 pieces, got %d", len(player.Pieces))
	}

	// Check all pieces start at home
	for _, piece := range player.Pieces {
		if !piece.IsHome {
			t.Error("Piece should start at home")
		}
		if piece.Position != -1 {
			t.Errorf("Expected piece position to be -1, got %d", piece.Position)
		}
	}
}

func TestJoinGameFull(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame(2) // Max 2 players

	// Join two players
	gm.JoinGame(game.Code, "player1", "Alice")
	gm.JoinGame(game.Code, "player2", "Bob")

	// Try to join third player
	_, err := gm.JoinGame(game.Code, "player3", "Charlie")
	if err != ErrGameFull {
		t.Errorf("Expected ErrGameFull, got %v", err)
	}
}

func TestJoinGameDuplicate(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame(4)

	gm.JoinGame(game.Code, "player1", "Alice")

	// Try to join with same player ID
	_, err := gm.JoinGame(game.Code, "player1", "Alice Again")
	if err != ErrPlayerExists {
		t.Errorf("Expected ErrPlayerExists, got %v", err)
	}
}

func TestStartGame(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame(4)

	gm.JoinGame(game.Code, "player1", "Alice")
	gm.JoinGame(game.Code, "player2", "Bob")

	err := game.StartGame()
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
	game, _ := gm.CreateGame(4)

	gm.JoinGame(game.Code, "player1", "Alice")

	err := game.StartGame()
	if err == nil {
		t.Error("Expected error when starting game with only 1 player")
	}
}

func TestRollDice(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame(4)

	roll := game.RollDice()

	if roll < 1 || roll > 6 {
		t.Errorf("Dice roll should be between 1 and 6, got %d", roll)
	}

	if game.LastDiceRoll != roll {
		t.Errorf("Last dice roll should be %d, got %d", roll, game.LastDiceRoll)
	}
}

func TestMovePiece(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame(2)

	gm.JoinGame(game.Code, "player1", "Alice")
	gm.JoinGame(game.Code, "player2", "Bob")
	game.StartGame()

	// Manually set dice roll to 6 to move piece out of home
	game.LastDiceRoll = 6

	err := game.MovePiece(game.CurrentTurn, 0)
	if err != nil {
		t.Fatalf("Failed to move piece: %v", err)
	}

	player := game.Players[game.CurrentTurn]
	piece := player.Pieces[0]

	if piece.IsHome {
		t.Error("Piece should no longer be at home")
	}

	if piece.Position != 0 {
		t.Errorf("Expected piece position to be 0, got %d", piece.Position)
	}
}

func TestMovePieceNotPlayerTurn(t *testing.T) {
	gm := NewGameManager()
	game, _ := gm.CreateGame(2)

	gm.JoinGame(game.Code, "player1", "Alice")
	gm.JoinGame(game.Code, "player2", "Bob")
	game.StartGame()

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
	game, _ := gm.CreateGame(4)

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
