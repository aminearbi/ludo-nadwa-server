# Ludo Nadwa Server

A classical Ludo board game server implementation written in Go. This server provides REST API endpoints for creating and managing multiplayer Ludo games that can be played on web, iOS, and Android clients.

## Features

- **Create Game**: Host a game that allows up to 5 players to join
- **Join Game**: Join a game using an 8-digit game code
- **Official Ludo Rules**: Implements standard Ludo game mechanics
- **Real-time Game State**: Track player positions, turns, and game progress
- **CORS Support**: Cross-origin requests enabled for web clients
- **RESTful API**: Simple HTTP/JSON interface for easy client integration

## Getting Started

### Prerequisites

- Go 1.21 or higher

### Installation

1. Clone the repository:
```bash
git clone https://github.com/aminearbi/ludo-nadwa-server.git
cd ludo-nadwa-server
```

2. Build the server:
```bash
go build -o ludo-server
```

3. Run the server:
```bash
./ludo-server
```

The server will start on port 8080 by default. You can change the port by setting the `PORT` environment variable:
```bash
PORT=3000 ./ludo-server
```

## API Endpoints

### Health Check
```
GET /health
```
Returns `OK` if the server is running.

### Create a Game
```
POST /api/game/create
Content-Type: application/json

{
  "max_players": 4,
  "player_id": "player1",
  "player_name": "Alice"
}
```

**Response:**
```json
{
  "code": "12345678",
  "message": "Game created successfully. Share this code with other players.",
  "max_players": 4
}
```

The creator is automatically added to the game if `player_id` and `player_name` are provided.

### Join a Game
```
POST /api/game/join
Content-Type: application/json

{
  "code": "12345678",
  "player_id": "player2",
  "player_name": "Bob"
}
```

**Response:**
```json
{
  "message": "Successfully joined the game",
  "game": {
    "code": "12345678",
    "state": "waiting",
    "players": { ... },
    "max_players": 4,
    "current_turn": "",
    "last_dice_roll": 0
  }
}
```

### Start a Game
```
POST /api/game/start
Content-Type: application/json

{
  "code": "12345678",
  "player_id": "player1"
}
```

Starts the game once at least 2 players have joined.

### Get Game State
```
GET /api/game/state?code=12345678
```

Returns the current state of the game including all player positions and whose turn it is.

### Roll Dice
```
POST /api/game/roll
Content-Type: application/json

{
  "code": "12345678",
  "player_id": "player1"
}
```

**Response:**
```json
{
  "roll": 6
}
```

### Move Piece
```
POST /api/game/move
Content-Type: application/json

{
  "code": "12345678",
  "player_id": "player1",
  "piece_id": 0
}
```

Moves the specified piece based on the last dice roll. Piece IDs range from 0 to 3.

## Game Rules

### Basic Rules
- Each player has 4 pieces that start in the home area
- Players take turns rolling a die (1-6)
- A piece can only leave home on a roll of 6
- A player gets an extra turn after rolling a 6
- The first player to get all 4 pieces to the finish area wins

### Player Colors
Players are automatically assigned colors in order:
1. Red
2. Blue
3. Green
4. Yellow
5. Purple

## Development

### Running Tests
```bash
go test ./models -v
```

### Project Structure
```
ludo-nadwa-server/
├── main.go              # Server entry point
├── models/
│   ├── game.go          # Game logic and state management
│   └── game_test.go     # Unit tests
└── handlers/
    └── game_handler.go  # HTTP request handlers
```

## Example Usage

1. **Player 1 creates a game:**
```bash
curl -X POST http://localhost:8080/api/game/create \
  -H "Content-Type: application/json" \
  -d '{"max_players": 4, "player_id": "p1", "player_name": "Alice"}'
```

2. **Player 2 joins using the 8-digit code:**
```bash
curl -X POST http://localhost:8080/api/game/join \
  -H "Content-Type: application/json" \
  -d '{"code": "12345678", "player_id": "p2", "player_name": "Bob"}'
```

3. **Start the game:**
```bash
curl -X POST http://localhost:8080/api/game/start \
  -H "Content-Type: application/json" \
  -d '{"code": "12345678", "player_id": "p1"}'
```

4. **Play the game:**
```bash
# Roll dice
curl -X POST http://localhost:8080/api/game/roll \
  -H "Content-Type: application/json" \
  -d '{"code": "12345678", "player_id": "p1"}'

# Move a piece
curl -X POST http://localhost:8080/api/game/move \
  -H "Content-Type: application/json" \
  -d '{"code": "12345678", "player_id": "p1", "piece_id": 0}'
```

## Client Integration

This server is designed to work with iOS, Android, and web clients. Clients should:

1. Call `/api/game/create` to generate a game code
2. Display the 8-digit code for other players to join
3. Poll `/api/game/state` to check for new players and game updates
4. Implement game board UI based on the game state
5. Send roll and move commands when it's the player's turn

## License

This project is open source and available under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.