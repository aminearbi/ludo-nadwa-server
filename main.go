package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aminearbi/ludo-nadwa-server/handlers"
	"github.com/aminearbi/ludo-nadwa-server/models"
)

func main() {
	// Parse command line flags
	portFlag := flag.String("port", "", "Port to run the server on (default: 8080)")
	flag.Parse()

	// Create game manager
	gameManager := models.NewGameManager()

	// Create WebSocket hub and start it
	hub := handlers.NewHub()
	go hub.Run()

	// Create handlers
	handler := handlers.NewHandler(gameManager)
	handler.SetHub(hub)

	wsHandler := handlers.NewWebSocketHandler(hub, gameManager)

	// Start cleanup goroutine
	go startCleanupRoutine(gameManager, hub)

	// Start turn timeout checker
	go startTurnTimeoutChecker(gameManager, hub)

	// Start bot turn handler
	go startBotTurnHandler(gameManager, hub)

	// Register REST API routes
	http.HandleFunc("/api/game/create", corsMiddleware(handler.CreateGame))
	http.HandleFunc("/api/game/join", corsMiddleware(handler.JoinGame))
	http.HandleFunc("/api/game/start", corsMiddleware(handler.StartGame))
	http.HandleFunc("/api/game/state", corsMiddleware(handler.GetGameState))
	http.HandleFunc("/api/game/roll", corsMiddleware(handler.RollDice))
	http.HandleFunc("/api/game/move", corsMiddleware(handler.MovePiece))
	http.HandleFunc("/api/game/skip", corsMiddleware(handler.SkipTurn))
	
	// New endpoints
	http.HandleFunc("/api/game/ready", corsMiddleware(handler.SetReady))
	http.HandleFunc("/api/game/kick", corsMiddleware(handler.KickPlayer))
	http.HandleFunc("/api/game/leave", corsMiddleware(handler.LeaveGame))
	http.HandleFunc("/api/game/pause", corsMiddleware(handler.PauseGame))
	http.HandleFunc("/api/game/resume", corsMiddleware(handler.ResumeGame))
	http.HandleFunc("/api/game/chat", corsMiddleware(handler.SendChat))
	http.HandleFunc("/api/game/spectate", corsMiddleware(handler.JoinAsSpectator))
	http.HandleFunc("/api/game/rematch", corsMiddleware(handler.Rematch))
	http.HandleFunc("/api/game/history", corsMiddleware(handler.GetMoveHistory))
	http.HandleFunc("/api/game/chat/history", corsMiddleware(handler.GetChat))
	
	// Bot endpoints
	http.HandleFunc("/api/game/bot/add", corsMiddleware(handler.AddBot))
	http.HandleFunc("/api/game/bot/remove", corsMiddleware(handler.RemoveBot))

	// WebSocket endpoint
	http.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// Stats endpoint
	http.HandleFunc("/api/stats", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gameManager.GetGameStats())
	}))

	// Health check endpoint
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Serve static web files
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	// Get port from flag, environment, or use default
	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	log.Printf("Ludo Nadwa Server starting on port %s", port)
	log.Printf("Endpoints:")
	log.Printf("  POST   /api/game/create       - Create a new game (host)")
	log.Printf("  POST   /api/game/join         - Join an existing game")
	log.Printf("  POST   /api/game/start        - Start a game (host only)")
	log.Printf("  GET    /api/game/state        - Get game state")
	log.Printf("  POST   /api/game/roll         - Roll the dice")
	log.Printf("  POST   /api/game/move         - Move a piece")
	log.Printf("  POST   /api/game/skip         - Skip turn (when no valid moves)")
	log.Printf("  POST   /api/game/ready        - Set player ready status")
	log.Printf("  POST   /api/game/kick         - Kick a player (host only)")
	log.Printf("  POST   /api/game/leave        - Leave a game")
	log.Printf("  POST   /api/game/pause        - Pause a game")
	log.Printf("  POST   /api/game/resume       - Resume a paused game")
	log.Printf("  POST   /api/game/chat         - Send a chat message")
	log.Printf("  GET    /api/game/chat/history - Get chat history")
	log.Printf("  POST   /api/game/spectate     - Join as spectator")
	log.Printf("  POST   /api/game/rematch      - Request a rematch (host only)")
	log.Printf("  GET    /api/game/history      - Get move history")
	log.Printf("  WS     /ws                    - WebSocket connection")
	log.Printf("  GET    /api/stats             - Server statistics")
	log.Printf("  GET    /health                - Health check")
	log.Printf("  GET    /                      - Web interface")
	log.Printf("")
	log.Printf("🎲 Open http://localhost:%s in your browser to play!", port)

	if err := http.ListenAndServe("0.0.0.0:"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// startCleanupRoutine periodically cleans up abandoned games
func startCleanupRoutine(gm *models.GameManager, hub *handlers.Hub) {
	ticker := time.NewTicker(models.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		removed := gm.CleanupAbandonedGames()
		if len(removed) > 0 {
			log.Printf("Cleaned up %d abandoned games: %v", len(removed), removed)
		}
	}
}

// startTurnTimeoutChecker checks for turn timeouts and auto-skips
func startTurnTimeoutChecker(gm *models.GameManager, hub *handlers.Hub) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		games := gm.GetAllGames()
		for _, game := range games {
			if game.IsTurnTimedOut() {
				skippedPlayer := game.ForceSkipTurn()
				if skippedPlayer != "" {
					log.Printf("Turn timeout for player %s in game %s", skippedPlayer, game.Code)
					hub.BroadcastToGame(game.Code, handlers.WebSocketEvent{
						Type: "turn_timeout",
						Data: map[string]interface{}{
							"skipped_player": skippedPlayer,
							"game":           game.GetGameState(),
						},
						Timestamp: time.Now(),
					})
				}
			}
		}
	}
}

// startBotTurnHandler checks if it's a bot's turn and plays automatically
func startBotTurnHandler(gm *models.GameManager, hub *handlers.Hub) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		games := gm.GetAllGames()
		for _, game := range games {
			if game.IsCurrentPlayerBot() {
				handleBotTurn(game, hub)
			}
		}
	}
}

// handleBotTurn plays a turn for the bot
func handleBotTurn(game *models.Game, hub *handlers.Hub) {
	gameState := game.GetGameState()
	currentTurn := gameState["current_turn"].(string)
	hasRolled := gameState["has_rolled"].(bool)
	
	// If bot hasn't rolled yet, roll the dice
	if !hasRolled {
		roll, err := game.RollDice(currentTurn)
		if err != nil {
			if err == models.ErrThreeSixes {
				// Three sixes - turn is forfeited, broadcast and return
				hub.BroadcastToGame(game.Code, handlers.WebSocketEvent{
					Type: "dice_rolled",
					Data: map[string]interface{}{
						"player_id":   currentTurn,
						"roll":        roll,
						"valid_moves": []int{},
						"has_moves":   false,
						"three_sixes": true,
						"is_bot":      true,
					},
					Timestamp: time.Now(),
				})
			}
			return
		}
		
		validMoves := game.GetValidMoves(currentTurn)
		hub.BroadcastToGame(game.Code, handlers.WebSocketEvent{
			Type: "dice_rolled",
			Data: map[string]interface{}{
				"player_id":   currentTurn,
				"roll":        roll,
				"valid_moves": validMoves,
				"has_moves":   len(validMoves) > 0,
				"is_bot":      true,
			},
			Timestamp: time.Now(),
		})
		
		// Small delay before moving to make it feel more natural
		time.Sleep(500 * time.Millisecond)
	}
	
	// Check for valid move and make it
	pieceID, hasMove := game.GetBotMove()
	if hasMove {
		if err := game.MovePiece(currentTurn, pieceID); err != nil {
			// No valid moves, skip turn
			game.SkipTurn(currentTurn)
			hub.BroadcastToGame(game.Code, handlers.WebSocketEvent{
				Type: "turn_skipped",
				Data: map[string]interface{}{
					"player_id": currentTurn,
					"is_bot":    true,
					"game":      game.GetGameState(),
				},
				Timestamp: time.Now(),
			})
			return
		}
		
		newGameState := game.GetGameState()
		hub.BroadcastToGame(game.Code, handlers.WebSocketEvent{
			Type: "piece_moved",
			Data: map[string]interface{}{
				"player_id": currentTurn,
				"piece_id":  pieceID,
				"is_bot":    true,
				"game":      newGameState,
			},
			Timestamp: time.Now(),
		})
		
		// Check for game end
		if newGameState["state"] == "ended" {
			hub.BroadcastToGame(game.Code, handlers.WebSocketEvent{
				Type: "game_ended",
				Data: map[string]interface{}{
					"winner":  newGameState["winner"],
					"is_bot":  true,
					"game":    newGameState,
				},
				Timestamp: time.Now(),
			})
		}
	} else {
		// No valid moves, skip turn
		game.SkipTurn(currentTurn)
		hub.BroadcastToGame(game.Code, handlers.WebSocketEvent{
			Type: "turn_skipped",
			Data: map[string]interface{}{
				"player_id": currentTurn,
				"is_bot":    true,
				"game":      game.GetGameState(),
			},
			Timestamp: time.Now(),
		})
	}
}

// corsMiddleware adds CORS headers to allow cross-origin requests
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
