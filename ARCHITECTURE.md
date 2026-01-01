# Architecture Documentation

## Overview
This Ludo game server is built with a clean, modular architecture using Go's standard library with minimal dependencies.

## Components

### 1. Models (`models/game.go`)
Core business logic and data structures:

- **GameManager**: Central manager for all game sessions
  - Thread-safe operations using sync.RWMutex
  - Creates and retrieves games
  - Manages game lifecycle

- **Game**: Represents a single game session
  - 8-digit unique code for joining
  - Supports 2-5 players
  - Tracks game state (waiting, playing, ended)
  - Manages turns and player order
  - Thread-safe with internal mutex

- **Player**: Represents a player in the game
  - Unique ID and name
  - Assigned color (red, blue, green, yellow, purple)
  - 4 game pieces
  - Turn order

- **Piece**: Individual game piece
  - Position tracking (-1 for home, 0-51 for board, 100+ for finished)
  - Home and safety status

### 2. Handlers (`handlers/game_handler.go`)
HTTP request handling and routing:

- **CreateGame**: Initialize new game session
- **JoinGame**: Add player to existing game
- **StartGame**: Begin gameplay
- **GetGameState**: Retrieve current game status
- **RollDice**: Generate random dice roll (1-6)
- **MovePiece**: Execute piece movement

All handlers:
- Validate input
- Return JSON responses
- Handle errors appropriately
- Support CORS for web clients

### 3. Main Server (`main.go`)
HTTP server setup and configuration:

- Route registration
- CORS middleware
- Port configuration via environment variable
- Health check endpoint

## Game Flow

```
1. Player creates game
   POST /api/game/create
   → Returns 8-digit code

2. Other players join
   POST /api/game/join
   → Share code to join

3. Start game (2+ players)
   POST /api/game/start
   → Game begins

4. Players take turns:
   a. Roll dice
      POST /api/game/roll
   
   b. Move piece
      POST /api/game/move
   
   c. Check state
      GET /api/game/state

5. Game ends when all pieces finish
```

## Thread Safety

The implementation uses mutexes at multiple levels:

- **GameManager.mu**: Protects the games map
- **Game.mu**: Protects individual game state
- Read-write locks used for optimal read performance

## Game Rules Implementation

### Piece Movement
- Pieces start at home (position -1)
- Must roll 6 to move piece out of home
- Pieces move clockwise around board (positions 0-51)
- Pieces reaching position > 51 enter finish area (100+)

### Turn Management
- Players take turns in join order
- Rolling 6 grants extra turn
- Turn passes to next player otherwise

### Win Condition
- First player to get all 4 pieces to finish area wins
- Game state changes to "ended"
- Winner is recorded

## Extensibility

The architecture supports easy additions:

1. **Capturing pieces**: Add collision detection in MovePiece
2. **Safe zones**: Mark certain positions as safe in board constants
3. **WebSockets**: Add real-time updates without polling
4. **Persistence**: Add database layer under GameManager
5. **Authentication**: Add JWT middleware to handlers
6. **Game history**: Store completed games for statistics

## Testing

Comprehensive unit tests cover:
- Code generation (8 digits, unique)
- Game creation and configuration
- Player joining and validation
- Game state transitions
- Dice rolling randomness
- Piece movement rules
- Turn management
- Error conditions

## Performance Considerations

- In-memory storage for fast access
- Concurrent-safe operations
- Efficient JSON serialization
- Stateless HTTP design for horizontal scaling

## Security

- No external dependencies reduces attack surface
- Input validation on all endpoints
- CORS configured for cross-origin requests
- Thread-safe concurrent access
- No SQL injection risk (no database)

## Future Improvements

1. Add game timeout and cleanup for abandoned games
2. Implement WebSocket for real-time updates
3. Add replay/spectator mode
4. Implement matchmaking system
5. Add game analytics and statistics
6. Support custom board configurations
7. Add AI players for single-player mode
