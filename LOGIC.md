# Ludo Server Logic & Algorithms

## Game Flow Overview

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   WAITING   │────▶│   PLAYING   │────▶│    ENDED    │────▶│   REMATCH   │
│  (Lobby)    │     │  (In Game)  │     │  (Winner)   │     │  (Restart)  │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
     │                    │
     │ All players ready  │ One player finishes
     │ + Host starts      │ all 4 pieces
     ▼                    ▼
```

## Turn Flow Algorithm

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           TURN START                                      │
│                    CurrentTurn = PlayerID                                 │
│                    HasRolled = false                                      │
│                    TurnStartTime = now()                                  │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                         ROLL DICE                                         │
│  Preconditions:                                                          │
│    - State == Playing                                                    │
│    - CurrentTurn == PlayerID                                             │
│    - HasRolled == false                                                  │
│                                                                          │
│  Actions:                                                                │
│    1. Generate secure random 1-6                                         │
│    2. Set LastDiceRoll = roll                                            │
│    3. Set HasRolled = true                                               │
│    4. If roll == 6: ConsecutiveSixes++                                   │
│       - If ConsecutiveSixes >= 3: FORFEIT TURN (go to NEXT TURN)         │
│    5. Else: ConsecutiveSixes = 0                                         │
│    6. Calculate valid moves                                              │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      VALID MOVES CHECK                                    │
│                                                                          │
│  For each piece (0-3):                                                   │
│    - If piece.IsFinished: SKIP                                           │
│    - If piece.IsHome AND roll == 6: VALID (can exit home)                │
│    - If piece.IsHome AND roll != 6: INVALID                              │
│    - If piece in HomeStretch:                                            │
│        newPos = HomeStretchPosition + roll                               │
│        If newPos <= 6: VALID                                             │
│        Else: INVALID (must land exactly)                                 │
│    - If piece on MainBoard:                                              │
│        Calculate if entering home stretch                                │
│        If entering and overshoot: INVALID                                │
│        Else: VALID                                                       │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │                               │
                    ▼                               ▼
        ┌─────────────────────┐         ┌─────────────────────┐
        │  NO VALID MOVES     │         │  HAS VALID MOVES    │
        │  Auto-skip turn     │         │  Player must move   │
        └─────────────────────┘         └─────────────────────┘
                    │                               │
                    │                               ▼
                    │               ┌──────────────────────────────────────┐
                    │               │            MOVE PIECE                 │
                    │               │  Preconditions:                       │
                    │               │    - HasRolled == true                │
                    │               │    - PieceID is valid move            │
                    │               │                                       │
                    │               │  Actions:                             │
                    │               │    1. Calculate new position          │
                    │               │    2. Check for captures              │
                    │               │    3. Check if piece finished         │
                    │               │    4. Check if player won             │
                    │               │    5. Set HasRolled = false           │
                    │               │    6. Determine next turn             │
                    │               └──────────────────────────────────────┘
                    │                               │
                    │                               ▼
                    │               ┌──────────────────────────────────────┐
                    │               │        EXTRA TURN CHECK               │
                    │               │                                       │
                    │               │  extraTurn = false                    │
                    │               │  IF roll == 6: extraTurn = true       │
                    │               │  IF captured AND CaptureGrantsTurn:   │
                    │               │     extraTurn = true                  │
                    │               └──────────────────────────────────────┘
                    │                               │
                    │               ┌───────────────┴───────────────┐
                    │               │                               │
                    │               ▼                               ▼
                    │   ┌─────────────────────┐         ┌─────────────────────┐
                    │   │   EXTRA TURN        │         │   NEXT TURN         │
                    │   │   Same player       │         │   nextTurn()        │
                    │   │   HasRolled = false │         │                     │
                    │   │   (can roll again)  │         │                     │
                    │   └─────────────────────┘         └─────────────────────┘
                    │               │                               │
                    └───────────────┴───────────────────────────────┘
                                    │
                                    ▼
                            TURN START (loop)
```

## nextTurn() Algorithm

```go
func nextTurn() {
    currentPlayer = Players[CurrentTurn]
    nextOrder = (currentPlayer.Order + 1) % len(Players)
    
    // Simple round-robin - find player with next order
    for each player in Players {
        if player.Order == nextOrder {
            CurrentTurn = player.ID
            TurnStartTime = now()
            HasRolled = false
            return
        }
    }
}
```

**Note**: Turn order is simple round-robin. No connection tracking needed.

## Board Position System

### Square Board (2-4 Players)
```
Board Size: 52 positions (0-51)
Home Stretch: 6 positions per player

Start Positions:
  Red:    0
  Blue:   13
  Green:  26  
  Yellow: 39

Home Stretch Entry (position before entering home stretch):
  Red:    50
  Blue:   11
  Green:  24
  Yellow: 37

Safe Zones: 0, 8, 13, 21, 26, 34, 39, 47
```

### Position Calculation

```go
func calculateNewPosition(color, currentPos, diceRoll) (newPos, enteredHomeStretch, homeStretchPos) {
    homeStretchEntry = GetHomeStretchEntry(color)
    boardSize = 52  // or 72 for hex
    
    // Calculate steps to home stretch entry
    if currentPos <= homeStretchEntry {
        stepsToEntry = homeStretchEntry - currentPos
    } else {
        stepsToEntry = (boardSize - currentPos) + homeStretchEntry
    }
    
    // Check if piece has completed a lap (eligible for home stretch)
    hasCompletedLap = checkLapComplete(color, currentPos)
    
    if hasCompletedLap AND diceRoll > stepsToEntry {
        // Enter home stretch
        homeStretchPos = diceRoll - stepsToEntry
        return -2, true, homeStretchPos
    }
    
    // Normal movement
    newPos = (currentPos + diceRoll) % boardSize
    return newPos, false, 0
}
```

## Piece States

```
┌─────────────────────────────────────────────────────────────────┐
│                        PIECE LIFECYCLE                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────┐    Roll 6    ┌──────────────┐                    │
│   │   HOME   │ ───────────▶ │  MAIN BOARD  │                    │
│   │ IsHome=T │              │  Position=N  │                    │
│   │ Pos=-1   │              │  IsHome=F    │                    │
│   └──────────┘              └──────────────┘                    │
│        ▲                           │                            │
│        │ Captured                  │ Pass home stretch entry    │
│        │                           ▼                            │
│        │                    ┌──────────────┐                    │
│        └────────────────────│ HOME STRETCH │                    │
│                             │ Position=-2  │                    │
│                             │ HomeStretch  │                    │
│                             │   Pos=1-6    │                    │
│                             └──────────────┘                    │
│                                    │                            │
│                                    │ HomeStretchPos == 6        │
│                                    ▼                            │
│                             ┌──────────────┐                    │
│                             │   FINISHED   │                    │
│                             │ IsFinished=T │                    │
│                             │ Pos=100+ID   │                    │
│                             └──────────────┘                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Capture Logic

```go
func checkAndCapture(currentPlayerID, position) captured {
    // Cannot capture on safe zones
    if IsSafeZone(position) {
        return false
    }
    
    captured = false
    for each player in Players {
        if player.ID == currentPlayerID {
            continue  // Don't capture own pieces
        }
        
        for each piece in player.Pieces {
            // Only capture pieces on main board at same position
            if piece.Position == position && 
               !piece.IsHome && 
               !piece.IsFinished && 
               piece.HomeStretchPosition == 0 {
                // Send piece back home
                piece.IsHome = true
                piece.Position = -1
                piece.IsSafe = false
                captured = true
            }
        }
    }
    return captured
}
```

## WebSocket Events (Hybrid Architecture)

WebSocket now only sends refresh hints. Clients fetch state via HTTP.

```
Server → Client:
  { type: "refresh", hint: "player_joined" }
  { type: "refresh", hint: "game_started" }
  { type: "refresh", hint: "dice_rolled" }
  { type: "refresh", hint: "piece_moved" }
  { type: "refresh", hint: "turn_skipped" }
  { type: "refresh", hint: "player_left" }
  { type: "refresh", hint: "chat_message" }
  etc.

Client receives refresh → GET /api/game/state?code=XXX
```

## State Flags Summary

| Flag | Set When | Reset When | Purpose |
|------|----------|------------|---------|
| `HasRolled` | After RollDice() | After MovePiece() or nextTurn() | Prevent multiple rolls |
| `IsHome` | Game start, piece captured | Piece exits with 6 | Track if piece in home base |
| `IsFinished` | Piece reaches end of home stretch | Never (game ends) | Track completed pieces |
| `IsSafe` | Piece on safe zone or home stretch | Piece moves to unsafe position | Prevent capture |

## Known Issues / Bug-Prone Areas

1. **Race conditions on roll**: Frontend must set `hasRolled = true` immediately after API response, not after animation
2. **Home stretch entry calculation**: Complex wrap-around logic for determining when piece can enter home stretch
3. **Extra turn on 6**: After rolling 6 and moving, `HasRolled` must be reset but `CurrentTurn` stays same
4. **Polling fallback**: If WebSocket disconnects, client polls every 2 seconds as fallback
