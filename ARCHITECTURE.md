# Architecture Documentation

## Overview
This Ludo game server is built with a clean, modular architecture using Go's standard library with minimal dependencies. It features real-time WebSocket support, secure dice rolling, and production-ready game management.

## Components

### 1. Models (`models/game.go`)
Core business logic and data structures:

- **GameManager**: Central manager for all game sessions
  - Thread-safe operations using sync.RWMutex
  - Creates games with host player
  - Manages spectators
  - Handles game lifecycle and cleanup

- **Game**: Represents a single game session
  - 8-digit secure unique code for joining
  - Supports 2-5 players + spectators
  - Tracks game state (waiting, playing, paused, ended)
  - Host controls (start, kick, rematch)
  - Player ready system
  - Move history and chat messages
  - Thread-safe with internal mutex

- **Player**: Represents a player in the game
  - Unique ID and validated name
  - Assigned color (red, blue, green, yellow, purple)
  - 4 game pieces
  - Ready status and host flag
  - Connection tracking

- **Piece**: Individual game piece
  - Position tracking (-1 for home, 0-51 for board, home stretch, 100+ for finished)
  - Home, safety, and home stretch position status

- **Spectator**: Represents someone watching the game
  - Can view game state and send chat messages
  - Cannot interact with game play

- **MoveRecord**: Tracks game history
  - Player, piece, dice roll, positions
  - Capture tracking for replay

- **ChatMessage**: In-game chat
  - Player/spectator messages with timestamps

### 2. Handlers (`handlers/game_handler.go`)
HTTP request handling and routing:

**Core Game Operations:**
- **CreateGame**: Initialize new game session (requires host info)
- **JoinGame**: Add player to existing game
- **StartGame**: Begin gameplay (host only, all players must be ready)
- **GetGameState**: Retrieve current game status
- **RollDice**: Generate secure random dice roll (1-6)
- **MovePiece**: Execute piece movement

**Player Management:**
- **SetReady**: Set player ready status before game start
- **KickPlayer**: Remove player from lobby (host only)
- **LeaveGame**: Player voluntarily leaves
- **JoinAsSpectator**: Watch a game without playing

**Game Control:**
- **PauseGame**: Pause an active game
- **ResumeGame**: Resume a paused game
- **SkipTurn**: Skip turn when no valid moves
- **Rematch**: Start a new game with same players (host only)

**Communication:**
- **SendChat**: Send chat message to game
- **GetChat**: Retrieve chat history
- **GetMoveHistory**: Retrieve move history

All handlers:
- Validate input (names, IDs)
- Return JSON responses
- Handle errors appropriately
- Support CORS for web clients
- Broadcast events via WebSocket

### 3. Main Server (`main.go`)
HTTP server setup and configuration:

- Route registration for 18 API endpoints
- WebSocket endpoint for real-time updates
- CORS middleware
- Port configuration via environment variable
- Health check endpoint
- Background cleanup routines
- Turn timeout monitoring

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
- **Turn timeout**: 60 second default, auto-skip on timeout

### Win Condition
- First player to get all 4 pieces to finish area wins
- Game state changes to "ended"
- Winner is recorded

### Advanced Game Rules (v2)
- **Per-player board paths**: Each color starts at different position (Red=0, Blue=13, Green=26, Yellow=39)
- **Piece capturing**: Landing on opponent piece sends it back home
- **Safe zones**: Start positions and star squares (0, 8, 13, 21, 26, 34, 39, 47) protect from capture
- **Home stretch**: Each player has private 6-square path before finish
- **Exact roll to finish**: Must roll exact number to enter finish area
- **Three sixes rule**: Rolling three consecutive 6s forfeits the turn
- **Capture bonus turn**: Optionally grants extra turn on capture

## Security Features

### Secure Random Number Generation
- Uses crypto/rand for game code generation
- Uses crypto/rand-seeded math/rand for dice rolls
- Prevents predictable game codes and dice manipulation

### Input Validation
- Player names: 1-30 characters
- Player IDs: 1-64 characters, alphanumeric with _ and -
- Chat messages: Max 500 characters
- All inputs trimmed and validated before use

## Real-time Features

### WebSocket Support (`handlers/websocket_handler.go`)
- Real-time game updates via WebSocket connections
- Hub pattern for managing client connections per game
- Automatic ping/pong for connection health
- Player connection tracking
- Spectator support

### WebSocket Connection
```
WS /ws?code=<game_code>&player_id=<player_id>
```

### Events
| Event | Description |
|-------|-------------|
| player_connected | Player WebSocket connected |
| player_disconnected | Player WebSocket disconnected |
| player_joined | New player joined game |
| player_left | Player left the game |
| player_kicked | Player was kicked by host |
| player_ready | Player ready status changed |
| spectator_joined | Spectator joined game |
| game_started | Game started playing |
| game_paused | Game was paused |
| game_resumed | Game was resumed |
| dice_rolled | Player rolled dice (includes three_sixes warning) |
| piece_moved | Player moved a piece |
| turn_skipped | Player skipped turn (no valid moves) |
| turn_timeout | Player turn timed out |
| game_ended | Game finished, winner declared |
| chat_message | Chat message received |
| rematch | Rematch started |

## Game Cleanup & Lifecycle

### Automatic Cleanup
- **Cleanup interval**: Every 5 minutes
- **Ended game TTL**: 30 minutes of inactivity
- **Waiting game TTL**: 30 minutes of inactivity  
- **Maximum game TTL**: 24 hours regardless of activity
- **Empty game TTL**: 5 minutes

### Turn Timeout
- Default: 60 seconds per turn
- Warning at 10 seconds remaining
- Checker runs every 5 seconds
- Auto-skips turn and broadcasts event
- Skips disconnected players automatically

## API Endpoints

### Core Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/game/create | Create game (host + max_players) |
| POST | /api/game/join | Join existing game |
| POST | /api/game/start | Start game (host only) |
| GET | /api/game/state | Get current game state |
| POST | /api/game/roll | Roll dice |
| POST | /api/game/move | Move a piece |
| POST | /api/game/skip | Skip turn |

### Player Management
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/game/ready | Set ready status |
| POST | /api/game/kick | Kick player (host only) |
| POST | /api/game/leave | Leave game |
| POST | /api/game/spectate | Join as spectator |

### Game Control
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/game/pause | Pause game |
| POST | /api/game/resume | Resume game |
| POST | /api/game/rematch | Start rematch (host only) |

### Communication & History
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/game/chat | Send chat message |
| GET | /api/game/chat/history | Get chat history |
| GET | /api/game/history | Get move history |

### Utility
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/stats | Server statistics |
| GET | /health | Health check |
| WS | /ws | WebSocket connection |

## Extensibility

The architecture supports easy additions:

1. ~~**Capturing pieces**: Add collision detection in MovePiece~~ ✅ Implemented
2. ~~**Safe zones**: Mark certain positions as safe in board constants~~ ✅ Implemented
3. ~~**WebSockets**: Add real-time updates without polling~~ ✅ Implemented
4. ~~**Host controls**: Game owner can start, kick, rematch~~ ✅ Implemented
5. ~~**Player ready system**: All players must be ready to start~~ ✅ Implemented
6. ~~**Chat system**: In-game messaging~~ ✅ Implemented
7. ~~**Spectator mode**: Watch games without playing~~ ✅ Implemented
8. ~~**Game pause/resume**: Temporarily halt gameplay~~ ✅ Implemented
9. ~~**Move history**: Track all moves for replay~~ ✅ Implemented
10. **Persistence**: Add database layer under GameManager
11. **Authentication**: Add JWT middleware to handlers

## Testing

Comprehensive unit tests cover:
- Code generation (8 digits, secure)
- Game creation with host
- Player joining and validation
- Ready system and game start
- Game state transitions
- Secure dice rolling
- Piece movement rules
- Capture mechanics
- Safe zone protection
- Home stretch navigation
- Turn management
- Three sixes rule
- Error conditions

## Performance Considerations

- In-memory storage for fast access
- Concurrent-safe operations
- Efficient JSON serialization
- Stateless HTTP design for horizontal scaling
- Skip disconnected players in turn rotation

## Security

- Cryptographically secure random numbers
- Input validation on all endpoints
- CORS configured for cross-origin requests
- Thread-safe concurrent access
- No SQL injection risk (no database)
- Rate limiting ready (see TODO)

## Future Improvements

See `TODO_LOW_PRIORITY.md` for planned features:
- Database persistence
- Rate limiting
- AI opponents
- Game variants
- Infrastructure (Docker, K8s)
