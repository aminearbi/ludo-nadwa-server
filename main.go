package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aminearbi/ludo-nadwa-server/handlers"
	"github.com/aminearbi/ludo-nadwa-server/models"
)

func main() {
	// Create game manager
	gameManager := models.NewGameManager()

	// Create handler
	handler := handlers.NewHandler(gameManager)

	// Register routes
	http.HandleFunc("/api/game/create", corsMiddleware(handler.CreateGame))
	http.HandleFunc("/api/game/join", corsMiddleware(handler.JoinGame))
	http.HandleFunc("/api/game/start", corsMiddleware(handler.StartGame))
	http.HandleFunc("/api/game/state", corsMiddleware(handler.GetGameState))
	http.HandleFunc("/api/game/roll", corsMiddleware(handler.RollDice))
	http.HandleFunc("/api/game/move", corsMiddleware(handler.MovePiece))

	// Health check endpoint
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Ludo Nadwa Server starting on port %s", port)
	log.Printf("Endpoints:")
	log.Printf("  POST   /api/game/create - Create a new game")
	log.Printf("  POST   /api/game/join   - Join an existing game")
	log.Printf("  POST   /api/game/start  - Start a game")
	log.Printf("  GET    /api/game/state  - Get game state")
	log.Printf("  POST   /api/game/roll   - Roll the dice")
	log.Printf("  POST   /api/game/move   - Move a piece")
	log.Printf("  GET    /health          - Health check")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
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
