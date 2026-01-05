// ==================== Configuration ====================
// Use the same host and port that served this page
const API_BASE = `${window.location.protocol}//${window.location.host}`;
const WS_BASE = API_BASE.replace('http', 'ws');

// ==================== Game State ====================
let gameState = {
    code: null,
    playerId: null,
    playerName: null,
    isHost: false,
    players: {},
    currentTurn: null,
    lastDiceRoll: null,
    hasRolled: false,
    validMoves: [],
    myColor: null,
    state: 'waiting',
    ws: null
};

// Timer and timeout tracking
const TURN_TIME_LIMIT = 15000; // 15 seconds per action
const MAX_TIMEOUTS = 3;
let turnTimer = null;
let turnStartTime = null;
let timerInterval = null;
let playerTimeouts = {}; // playerId -> timeout count

// Animation state
let animatingPieces = new Map(); // pieceKey -> {waypoints, currentWaypoint, progress}
let lastPiecePositions = new Map(); // pieceKey -> {x, y}
let lastPieceState = new Map(); // pieceKey -> {position, home_stretch_position, is_home, is_finished}
const HOP_DURATION = 150; // ms per cell hop

// ==================== DOM Elements ====================
const screens = {
    lobby: document.getElementById('lobby-screen'),
    waiting: document.getElementById('waiting-screen'),
    game: document.getElementById('game-screen')
};

const elements = {
    // Lobby
    createName: document.getElementById('create-name'),
    joinName: document.getElementById('join-name'),
    gameCode: document.getElementById('game-code'),
    createBtn: document.getElementById('create-btn'),
    joinBtn: document.getElementById('join-btn'),
    
    // Waiting
    displayCode: document.getElementById('display-code'),
    copyCode: document.getElementById('copy-code'),
    playersList: document.getElementById('players-list'),
    readyCheckbox: document.getElementById('ready-checkbox'),
    startBtn: document.getElementById('start-btn'),
    leaveBtn: document.getElementById('leave-btn'),
    
    // Game
    canvas: document.getElementById('game-board'),
    dice: document.getElementById('dice'),
    diceFace: document.getElementById('dice-face'),
    rollBtn: document.getElementById('roll-btn'),
    turnIndicator: document.getElementById('turn-indicator'),
    validMovesDiv: document.getElementById('valid-moves'),
    pieceButtons: document.getElementById('piece-buttons'),
    skipBtn: document.getElementById('skip-btn'),
    gamePlayersList: document.getElementById('game-players-list'),
    chatMessages: document.getElementById('chat-messages'),
    chatInput: document.getElementById('chat-input'),
    sendChat: document.getElementById('send-chat'),
    
    // Modal
    winnerModal: document.getElementById('winner-modal'),
    winnerName: document.getElementById('winner-name'),
    rematchBtn: document.getElementById('rematch-btn'),
    backLobbyBtn: document.getElementById('back-lobby-btn'),
    
    // Effects
    particles: document.getElementById('particles'),
    toastContainer: document.getElementById('toast-container')
};

const ctx = elements.canvas.getContext('2d');

// ==================== Colors ====================
const COLORS = {
    red: { main: '#e74c3c', light: '#ff6b6b', dark: '#c0392b' },
    blue: { main: '#2196F3', light: '#64B5F6', dark: '#1565C0' },
    green: { main: '#4CAF50', light: '#81C784', dark: '#2E7D32' },
    yellow: { main: '#f1c40f', light: '#ffeaa7', dark: '#f39c12' },
    purple: { main: '#9C27B0', light: '#BA68C8', dark: '#6A1B9A' },
    orange: { main: '#e67e22', light: '#fab1a0', dark: '#d35400' },
    olive: { main: '#808000', light: '#9ACD32', dark: '#556B2F' },
    indigo: { main: '#3F51B5', light: '#7986CB', dark: '#283593' }
};

const PLAYER_COLORS = ['red', 'blue', 'green', 'yellow', 'purple', 'orange'];

// ==================== Board Configuration ====================
const BOARD_SIZE = 600;
let CELL_SIZE = BOARD_SIZE / 15;
let PIECE_RADIUS = CELL_SIZE * 0.35;

// Track the current board type (will be set based on player count)
let currentBoardType = 'square'; // 'square' for 2-4 players, 'hex' for 5-6 players

// ==================== Square Board (2-4 Players) ====================
// Board positions for each cell (0-51)
const SQUARE_BOARD_POSITIONS = [];
function initSquareBoardPositions() {
    const positions = [
        // Bottom row (left to right) - positions 0-5
        {x: 6, y: 13}, {x: 6, y: 12}, {x: 6, y: 11}, {x: 6, y: 10}, {x: 6, y: 9}, {x: 5, y: 8},
        // Left column (bottom to top) - positions 6-12
        {x: 4, y: 8}, {x: 3, y: 8}, {x: 2, y: 8}, {x: 1, y: 8}, {x: 0, y: 8}, {x: 0, y: 7}, {x: 0, y: 6},
        // Top row (left to right) - positions 13-18
        {x: 1, y: 6}, {x: 2, y: 6}, {x: 3, y: 6}, {x: 4, y: 6}, {x: 5, y: 6}, {x: 6, y: 5},
        // Top-left to top-right - positions 19-25
        {x: 6, y: 4}, {x: 6, y: 3}, {x: 6, y: 2}, {x: 6, y: 1}, {x: 6, y: 0}, {x: 7, y: 0}, {x: 8, y: 0},
        // Right column (top to bottom) - positions 26-31
        {x: 8, y: 1}, {x: 8, y: 2}, {x: 8, y: 3}, {x: 8, y: 4}, {x: 8, y: 5}, {x: 9, y: 6},
        // Continue right side - positions 32-38
        {x: 10, y: 6}, {x: 11, y: 6}, {x: 12, y: 6}, {x: 13, y: 6}, {x: 14, y: 6}, {x: 14, y: 7}, {x: 14, y: 8},
        // Bottom-right going left - positions 39-44
        {x: 13, y: 8}, {x: 12, y: 8}, {x: 11, y: 8}, {x: 10, y: 8}, {x: 9, y: 8}, {x: 8, y: 9},
        // Final stretch - positions 45-51
        {x: 8, y: 10}, {x: 8, y: 11}, {x: 8, y: 12}, {x: 8, y: 13}, {x: 8, y: 14}, {x: 7, y: 14}, {x: 6, y: 14}
    ];
    positions.forEach(p => SQUARE_BOARD_POSITIONS.push(p));
}

// Square board home positions (2-4 players)
const SQUARE_HOME_POSITIONS = {
    blue: [{x: 1.2, y: 1.2}, {x: 3.8, y: 1.2}, {x: 1.2, y: 3.8}, {x: 3.8, y: 3.8}],
    green: [{x: 10.2, y: 1.2}, {x: 12.8, y: 1.2}, {x: 10.2, y: 3.8}, {x: 12.8, y: 3.8}],
    red: [{x: 1.2, y: 10.2}, {x: 3.8, y: 10.2}, {x: 1.2, y: 12.8}, {x: 3.8, y: 12.8}],
    yellow: [{x: 10.2, y: 10.2}, {x: 12.8, y: 10.2}, {x: 10.2, y: 12.8}, {x: 12.8, y: 12.8}]
};

// Square board home stretch positions (2-4 players)
const SQUARE_HOME_STRETCH = {
    blue: [{x: 1, y: 7}, {x: 2, y: 7}, {x: 3, y: 7}, {x: 4, y: 7}, {x: 5, y: 7}, {x: 6, y: 7}],
    green: [{x: 7, y: 1}, {x: 7, y: 2}, {x: 7, y: 3}, {x: 7, y: 4}, {x: 7, y: 5}, {x: 7, y: 6}],
    red: [{x: 7, y: 13}, {x: 7, y: 12}, {x: 7, y: 11}, {x: 7, y: 10}, {x: 7, y: 9}, {x: 7, y: 8}],
    yellow: [{x: 13, y: 7}, {x: 12, y: 7}, {x: 11, y: 7}, {x: 10, y: 7}, {x: 9, y: 7}, {x: 8, y: 7}]
};

const SQUARE_START_POSITIONS = {
    red: 0, blue: 13, yellow: 26, green: 39
};

const SQUARE_SAFE_ZONES = [0, 8, 13, 21, 26, 34, 39, 47];

// ==================== Hexagonal Board (5-6 Players) ====================
// Based on the reference 6-player Ludo board image
// Colors clockwise from bottom: Blue, Red, Green, Purple, Olive, Indigo
// 6-player Ludo has 72 squares on the main track (6 arms × 12 squares per arm)
const HEX_BOARD_SIZE = 72;
const HEX_BOARD_POSITIONS = [];

// Center of the hexagonal board in canvas coordinates
const HEX_CENTER_X = BOARD_SIZE / 2;
const HEX_CENTER_Y = BOARD_SIZE / 2;

// Color order matching reference image (clockwise from bottom/6 o'clock position)
// Player 2=Blue(bottom), Player 1=Red(bottom-right), Green(right), Player 5=Purple(top-right), Player 4=Olive(top-left), Player 3=Indigo(left)
const HEX_COLOR_ORDER = ['blue', 'red', 'green', 'purple', 'olive', 'indigo'];

// Cell dimensions
const HEX_CELL_W = BOARD_SIZE / 18;  // Cell width
const HEX_CELL_H = BOARD_SIZE / 15;  // Cell height

// Generate hexagonal board positions - EXACTLY matching reference image
// The track forms a continuous path: each arm has cells on both sides of the home stretch
function initHexBoardPositions() {
    HEX_BOARD_POSITIONS.length = 0;
    
    // For each arm, we need to place 12 cells:
    // - 3 cells on the LEFT side of home stretch (going outward from corner)
    // - 3 cells at the OUTER junction (curved around home area)
    // - 3 cells on the RIGHT side of home stretch (going inward toward corner)
    // - 3 cells at the INNER corner (connecting to next arm)
    
    for (let arm = 0; arm < 6; arm++) {
        const armAngle = (arm * 60 - 90) * Math.PI / 180; // Start from top, go clockwise
        const nextArmAngle = ((arm + 1) * 60 - 90) * Math.PI / 180;
        
        // Perpendicular to arm direction
        const perpAngle = armAngle + Math.PI / 2;
        
        // Distances
        const homeStretchOffset = HEX_CELL_W * 0.7; // Offset from center line to track
        const outerDist = BOARD_SIZE * 0.40;  // Outer edge of track
        const midDist = BOARD_SIZE * 0.28;    // Corner area
        const innerDist = BOARD_SIZE * 0.18;  // Near center
        
        // Position 0: Start position (colored) - at outer left of home stretch
        // Position 1-2: Going inward on left side
        for (let i = 0; i < 3; i++) {
            const dist = outerDist - i * HEX_CELL_H;
            HEX_BOARD_POSITIONS.push({
                x: HEX_CENTER_X + Math.cos(armAngle) * dist - Math.cos(perpAngle) * homeStretchOffset,
                y: HEX_CENTER_Y + Math.sin(armAngle) * dist - Math.sin(perpAngle) * homeStretchOffset,
                arm: arm,
                section: 'left',
                isStart: i === 0
            });
        }
        
        // Positions 3-5: Corner (curving from this arm to next)
        for (let i = 0; i < 3; i++) {
            const t = (i + 0.5) / 3;
            const angle = armAngle + t * (Math.PI / 3);
            const dist = midDist;
            HEX_BOARD_POSITIONS.push({
                x: HEX_CENTER_X + Math.cos(angle) * dist,
                y: HEX_CENTER_Y + Math.sin(angle) * dist,
                arm: arm,
                section: 'corner'
            });
        }
        
        // Positions 6-8: Right side of NEXT arm (going outward)
        for (let i = 0; i < 3; i++) {
            const dist = midDist + i * HEX_CELL_H * 0.9;
            const nextPerpAngle = nextArmAngle + Math.PI / 2;
            HEX_BOARD_POSITIONS.push({
                x: HEX_CENTER_X + Math.cos(nextArmAngle) * dist + Math.cos(nextPerpAngle) * homeStretchOffset,
                y: HEX_CENTER_Y + Math.sin(nextArmAngle) * dist + Math.sin(nextPerpAngle) * homeStretchOffset,
                arm: arm,
                section: 'right'
            });
        }
        
        // Positions 9-11: Outer junction (around the home area)
        for (let i = 0; i < 3; i++) {
            const t = (i + 0.5) / 4;
            const startAngle = nextArmAngle - Math.PI / 10;
            const endAngle = nextArmAngle + Math.PI / 10;
            const angle = startAngle + t * (endAngle - startAngle);
            const dist = outerDist + HEX_CELL_H * 0.3;
            HEX_BOARD_POSITIONS.push({
                x: HEX_CENTER_X + Math.cos(angle) * dist,
                y: HEX_CENTER_Y + Math.sin(angle) * dist,
                arm: arm,
                section: 'junction'
            });
        }
    }
}

// Hex board home positions (6 areas with 4 circles each)
const HEX_HOME_POSITIONS = {};
function initHexHomePositions() {
    HEX_COLOR_ORDER.forEach((color, i) => {
        const angle = (i * 60 - 90) * Math.PI / 180;
        const homeRadius = BOARD_SIZE * 0.38;
        const cx = HEX_CENTER_X + Math.cos(angle) * homeRadius;
        const cy = HEX_CENTER_Y + Math.sin(angle) * homeRadius;
        
        // 4 piece positions in a 2x2 grid
        const spacing = BOARD_SIZE * 0.028;
        HEX_HOME_POSITIONS[color] = [
            { x: cx - spacing, y: cy - spacing },
            { x: cx + spacing, y: cy - spacing },
            { x: cx - spacing, y: cy + spacing },
            { x: cx + spacing, y: cy + spacing }
        ];
    });
}

// Hex board home stretch (5 colored cells leading to center)
const HEX_HOME_STRETCH = {};
function initHexHomeStretch() {
    HEX_COLOR_ORDER.forEach((color, i) => {
        const angle = (i * 60 - 90) * Math.PI / 180;
        const positions = [];
        
        // 5 colored cells + 1 finish position
        const startDist = BOARD_SIZE * 0.28;
        const endDist = BOARD_SIZE * 0.08;
        
        for (let j = 0; j < 6; j++) {
            const t = j / 5;
            const dist = startDist - t * (startDist - endDist);
            positions.push({
                x: HEX_CENTER_X + Math.cos(angle) * dist,
                y: HEX_CENTER_Y + Math.sin(angle) * dist
            });
        }
        
        HEX_HOME_STRETCH[color] = positions;
    });
}

// Hex start positions (which main board position each color starts at)
const HEX_START_POSITIONS = {
    blue: 0,     // Arm 0 (bottom) - Player 2
    red: 12,     // Arm 1 (bottom-right) - Player 1
    green: 24,   // Arm 2 (right)
    purple: 36,  // Arm 3 (top-right) - Player 5
    olive: 48,   // Arm 4 (top-left) - Player 4
    indigo: 60   // Arm 5 (left) - Player 3
};

// Hex safe zones (star positions - start position + one more per arm)
const HEX_SAFE_ZONES = [0, 3, 12, 15, 24, 27, 36, 39, 48, 51, 60, 63];

// ==================== Active Board Config (switches based on player count) ====================
let BOARD_POSITIONS = [];
let HOME_POSITIONS = {};
let HOME_STRETCH = {};
let START_POSITIONS = {};
let SAFE_ZONES = [];

// Initialize default (square) board
function initBoardPositions() {
    initSquareBoardPositions();
    initHexBoardPositions();
    initHexHomePositions();
    initHexHomeStretch();
    
    // Default to square board
    setBoardType('square');
}

// Switch board configuration based on player count
function setBoardType(type) {
    currentBoardType = type;
    
    if (type === 'hex') {
        BOARD_POSITIONS = HEX_BOARD_POSITIONS;
        HOME_POSITIONS = HEX_HOME_POSITIONS;
        HOME_STRETCH = HEX_HOME_STRETCH;
        START_POSITIONS = HEX_START_POSITIONS;
        SAFE_ZONES = HEX_SAFE_ZONES;
        CELL_SIZE = BOARD_SIZE / 20; // Smaller cells for hex board
        PIECE_RADIUS = CELL_SIZE * 0.4;
    } else {
        BOARD_POSITIONS = SQUARE_BOARD_POSITIONS;
        HOME_POSITIONS = SQUARE_HOME_POSITIONS;
        HOME_STRETCH = SQUARE_HOME_STRETCH;
        START_POSITIONS = SQUARE_START_POSITIONS;
        SAFE_ZONES = SQUARE_SAFE_ZONES;
        CELL_SIZE = BOARD_SIZE / 15;
        PIECE_RADIUS = CELL_SIZE * 0.35;
    }
}

// Determine board type from player count
function getBoardTypeForPlayerCount(count) {
    return count >= 5 ? 'hex' : 'square';
}

initBoardPositions();

// ==================== Utility Functions ====================
function generatePlayerId() {
    return 'player_' + Math.random().toString(36).substr(2, 9);
}

function showScreen(screenName) {
    Object.values(screens).forEach(s => s.classList.remove('active'));
    screens[screenName].classList.add('active');
}

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.innerHTML = `<span>${message}</span>`;
    elements.toastContainer.appendChild(toast);
    
    setTimeout(() => {
        toast.classList.add('removing');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

function playSound(soundId) {
    try {
        const sound = document.getElementById(soundId);
        if (sound) {
            sound.currentTime = 0;
            sound.volume = 0.3;
            sound.play().catch(() => {});
        }
    } catch (e) {}
}

// ==================== API Functions ====================
async function apiCall(endpoint, method = 'GET', body = null) {
    const options = {
        method,
        headers: { 'Content-Type': 'application/json' }
    };
    if (body) options.body = JSON.stringify(body);
    
    const response = await fetch(`${API_BASE}${endpoint}`, options);
    const data = await response.json();
    
    if (!response.ok) {
        throw new Error(data.error || 'API request failed');
    }
    
    return data;
}

async function createGame() {
    const name = elements.createName.value.trim();
    if (!name) {
        showToast('Please enter your name', 'error');
        return;
    }
    
    const maxPlayers = parseInt(document.querySelector('.player-btn.active').dataset.players);
    gameState.playerId = generatePlayerId();
    gameState.playerName = name;
    
    try {
        const response = await apiCall('/api/game/create', 'POST', {
            player_id: gameState.playerId,
            player_name: name,
            max_players: maxPlayers
        });
        
        gameState.code = response.code;
        gameState.isHost = true;
        
        connectWebSocket();
        showWaitingRoom();
        showToast('Game created!', 'success');
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function joinGame() {
    const name = elements.joinName.value.trim();
    const code = elements.gameCode.value.trim();
    
    if (!name) {
        showToast('Please enter your name', 'error');
        return;
    }
    if (!code || code.length !== 8) {
        showToast('Please enter a valid 8-digit code', 'error');
        return;
    }
    
    gameState.playerId = generatePlayerId();
    gameState.playerName = name;
    gameState.code = code;
    
    try {
        const response = await apiCall('/api/game/join', 'POST', {
            code: code,
            player_id: gameState.playerId,
            player_name: name
        });
        
        connectWebSocket();
        showWaitingRoom();
        showToast('Joined game!', 'success');
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function setReady(ready) {
    try {
        await apiCall('/api/game/ready', 'POST', {
            code: gameState.code,
            player_id: gameState.playerId,
            ready: ready
        });
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function startGame() {
    try {
        await apiCall('/api/game/start', 'POST', {
            code: gameState.code,
            player_id: gameState.playerId
        });
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function rollDice() {
    try {
        elements.rollBtn.disabled = true;
        elements.dice.classList.add('rolling');
        playSound('diceSound');
        
        // Simulate roll animation
        const animDuration = 800;
        const startTime = Date.now();
        const animate = () => {
            if (Date.now() - startTime < animDuration) {
                elements.diceFace.textContent = Math.floor(Math.random() * 6) + 1;
                requestAnimationFrame(animate);
            }
        };
        animate();
        
        const response = await apiCall('/api/game/roll', 'POST', {
            code: gameState.code,
            player_id: gameState.playerId
        });
        
        setTimeout(() => {
            elements.dice.classList.remove('rolling');
            elements.diceFace.textContent = response.roll;
            gameState.lastDiceRoll = response.roll;
            gameState.hasRolled = true;
            gameState.validMoves = response.valid_moves;
            
            if (response.roll === 6) {
                showSixEffect();
            }
            
            updateValidMoves();
        }, animDuration);
        
    } catch (error) {
        elements.dice.classList.remove('rolling');
        // Only show error if it's not a "not your turn" error (expected during race conditions)
        if (!error.message.includes('not your turn') && !error.message.includes('already rolled')) {
            showToast(error.message, 'error');
        }
        // Reset UI state - updateUI will set proper button disabled state
        updateUI();
    }
}

async function movePiece(pieceId) {
    try {
        playSound('moveSound');
        const response = await apiCall('/api/game/move', 'POST', {
            code: gameState.code,
            player_id: gameState.playerId,
            piece_id: pieceId
        });
        
        gameState.hasRolled = false;
        gameState.validMoves = [];
        elements.validMovesDiv.style.display = 'none';
        
    } catch (error) {
        // Only show error if it's not a "not your turn" error (which is expected during race conditions)
        if (!error.message.includes('not your turn') && !error.message.includes('not in playing')) {
            showToast(error.message, 'error');
        }
        // Reset UI state regardless
        updateUI();
    }
}

async function skipTurn() {
    try {
        await apiCall('/api/game/skip', 'POST', {
            code: gameState.code,
            player_id: gameState.playerId
        });
        
        gameState.hasRolled = false;
        gameState.validMoves = [];
        elements.validMovesDiv.style.display = 'none';
        
    } catch (error) {
        // Only show error if it's not a "not your turn" error (which is expected during race conditions)
        if (!error.message.includes('not your turn') && !error.message.includes('not in playing')) {
            showToast(error.message, 'error');
        }
        // Reset UI state regardless
        updateUI();
    }
}

async function sendChat() {
    const message = elements.chatInput.value.trim();
    if (!message) return;
    
    try {
        await apiCall('/api/game/chat', 'POST', {
            code: gameState.code,
            player_id: gameState.playerId,
            message: message
        });
        elements.chatInput.value = '';
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function requestRematch() {
    try {
        await apiCall('/api/game/rematch', 'POST', {
            code: gameState.code,
            host_id: gameState.playerId
        });
        elements.winnerModal.classList.remove('active');
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function leaveGame() {
    try {
        await apiCall('/api/game/leave', 'POST', {
            code: gameState.code,
            player_id: gameState.playerId
        });
    } catch (error) {}
    
    if (gameState.ws) {
        gameState.ws.close();
    }
    resetGameState();
    showScreen('lobby');
}

// ==================== WebSocket ====================
function connectWebSocket() {
    const wsUrl = `${WS_BASE}/ws?code=${gameState.code}&player_id=${gameState.playerId}`;
    gameState.ws = new WebSocket(wsUrl);
    
    gameState.ws.onopen = () => {
        console.log('WebSocket connected');
    };
    
    gameState.ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
    };
    
    gameState.ws.onclose = () => {
        console.log('WebSocket disconnected');
    };
    
    gameState.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
}

function handleWebSocketMessage(message) {
    console.log('WS Message:', message.type, message.data);
    
    switch (message.type) {
        case 'player_joined':
        case 'player_left':
        case 'player_kicked':
        case 'player_ready':
            updateFromGameState(message.data.game);
            if (message.type === 'player_joined') {
                showToast(`${getPlayerName(message.data.player_id)} joined!`, 'success');
            } else if (message.type === 'player_left') {
                showToast(`A player left`, 'warning');
            }
            break;
            
        case 'game_started':
            updateFromGameState(message.data.game);
            showGameScreen();
            showToast('Game started! 🎲', 'success');
            break;
            
        case 'dice_rolled':
            if (message.data.player_id !== gameState.playerId) {
                elements.diceFace.textContent = message.data.roll;
                gameState.lastDiceRoll = message.data.roll;
                if (message.data.roll === 6) {
                    showSixEffect();
                }
                if (message.data.three_sixes) {
                    showToast(`${getPlayerName(message.data.player_id)} rolled 3 sixes! Turn lost!`, 'warning');
                }
            }
            break;
            
        case 'piece_moved':
            // Store old positions and state before update
            const oldPositions = new Map();
            const oldPieceStates = new Map();
            Object.values(gameState.players).forEach(player => {
                if (!player.pieces) return;
                player.pieces.forEach((piece, idx) => {
                    const key = `${player.color}-${idx}`;
                    if (lastPiecePositions.has(key)) {
                        oldPositions.set(key, lastPiecePositions.get(key));
                    }
                    // Store the actual piece state
                    oldPieceStates.set(key, {
                        position: piece.position,
                        home_stretch_position: piece.home_stretch_position,
                        is_home: piece.is_home,
                        is_finished: piece.is_finished,
                        color: player.color
                    });
                });
            });
            
            updateFromGameState(message.data.game);
            
            // Detect which piece moved and animate it
            if (message.data.player_id && message.data.piece_id !== undefined) {
                const movedPlayer = gameState.players[message.data.player_id];
                if (movedPlayer) {
                    const pieceKey = `${movedPlayer.color}-${message.data.piece_id}`;
                    const oldPos = oldPositions.get(pieceKey);
                    const oldState = oldPieceStates.get(pieceKey);
                    if (oldPos && oldState) {
                        animatePieceMovement(pieceKey, oldPos, oldState);
                    }
                }
            }
            
            if (message.data.captured) {
                showKillEffect();
                playSound('captureSound');
            }
            drawBoard();
            updateUI();
            break;
            
        case 'turn_skipped':
        case 'turn_timeout':
            updateFromGameState(message.data.game);
            if (message.type === 'turn_timeout') {
                showToast(`${getPlayerName(message.data.skipped_player)} timed out!`, 'warning');
            }
            drawBoard();
            updateUI();
            break;
            
        case 'game_ended':
            updateFromGameState(message.data.game);
            showWinner(message.data.winner);
            break;
            
        case 'chat_message':
            addChatMessage(message.data.player_name, message.data.message, message.data.player_id);
            break;
            
        case 'game_paused':
            showToast('Game paused', 'warning');
            break;
            
        case 'game_resumed':
            showToast('Game resumed', 'success');
            break;
            
        case 'rematch':
            updateFromGameState(message.data.game);
            showWaitingRoom();
            elements.winnerModal.classList.remove('active');
            showToast('Rematch! Get ready!', 'success');
            break;
    }
}

function getPlayerName(playerId) {
    const player = gameState.players[playerId];
    return player ? player.name : 'Unknown';
}

// ==================== UI Updates ====================
function updateFromGameState(game) {
    if (!game) return;
    
    // Track if turn changed to reset validMoves
    const turnChanged = gameState.currentTurn !== game.current_turn;
    
    gameState.players = game.players || {};
    gameState.currentTurn = game.current_turn;
    gameState.state = game.state;
    gameState.lastDiceRoll = game.last_dice_roll;
    gameState.hasRolled = game.has_rolled;
    
    // Reset validMoves when turn changes (new player hasn't rolled yet)
    if (turnChanged) {
        gameState.validMoves = [];
    }
    
    // Set board type based on max_players (5-6 uses hex board)
    const playerCount = game.max_players || Object.keys(game.players).length;
    const newBoardType = getBoardTypeForPlayerCount(playerCount);
    if (newBoardType !== currentBoardType) {
        setBoardType(newBoardType);
    }
    
    // Find my color
    if (gameState.players[gameState.playerId]) {
        gameState.myColor = gameState.players[gameState.playerId].color;
        gameState.isHost = gameState.players[gameState.playerId].is_host;
    }
    
    updatePlayersUI();
    
    if (gameState.state === 'playing') {
        drawBoard();
        updateUI();
    }
}

function updatePlayersUI() {
    // Update waiting room
    if (screens.waiting.classList.contains('active')) {
        updateWaitingPlayersUI();
    }
    
    // Update game screen
    if (screens.game.classList.contains('active')) {
        updateGamePlayersUI();
    }
}

function updateWaitingPlayersUI() {
    elements.playersList.innerHTML = '';
    
    Object.values(gameState.players).forEach(player => {
        const card = document.createElement('div');
        card.className = `player-card ${player.is_ready ? 'ready' : ''} ${player.is_host ? 'host' : ''}`;
        
        card.innerHTML = `
            <div class="player-avatar ${player.color}">${player.name.charAt(0).toUpperCase()}</div>
            <div class="player-info">
                <div class="name">${player.name} ${player.is_host ? '👑' : ''}</div>
                <div class="status ${player.is_ready ? 'ready' : ''}">${player.is_ready ? '✓ Ready' : 'Not ready'}</div>
            </div>
        `;
        
        elements.playersList.appendChild(card);
    });
    
    // Show start button for host if all ready
    const allReady = Object.values(gameState.players).every(p => p.is_ready);
    const enoughPlayers = Object.keys(gameState.players).length >= 2;
    
    if (gameState.isHost) {
        elements.startBtn.style.display = 'inline-flex';
        elements.startBtn.disabled = !(allReady && enoughPlayers);
    }
}

function updateGamePlayersUI() {
    elements.gamePlayersList.innerHTML = '';
    
    Object.values(gameState.players).forEach(player => {
        const isCurrentTurn = player.id === gameState.currentTurn;
        const finishedPieces = player.pieces ? player.pieces.filter(p => p.is_finished).length : 0;
        const timeouts = playerTimeouts[player.id] || 0;
        
        const card = document.createElement('div');
        card.className = `game-player-card ${player.color} ${isCurrentTurn ? 'current-turn' : ''}`;
        card.id = `player-card-${player.id}`;
        
        // Create timeout dots (3 dots, red for consumed)
        let timeoutDots = '';
        for (let i = 0; i < MAX_TIMEOUTS; i++) {
            const consumed = i < timeouts;
            timeoutDots += `<span class="timeout-dot ${consumed ? 'consumed' : ''}"></span>`;
        }
        
        card.innerHTML = `
            <div class="mini-avatar ${player.color}">${player.name.charAt(0).toUpperCase()}</div>
            <div class="game-player-info">
                <div class="name">${player.name}</div>
                <div class="pieces-status">🏠 ${finishedPieces}/4 finished</div>
                <div class="timeout-dots">${timeoutDots}</div>
            </div>
            <div class="player-timer-container">
                ${isCurrentTurn ? '<div class="player-timer" id="timer-' + player.id + '"><svg class="timer-svg" viewBox="0 0 36 36"><circle class="timer-bg" cx="18" cy="18" r="16"/><circle class="timer-progress" cx="18" cy="18" r="16" id="timer-circle-' + player.id + '"/></svg><span class="timer-text" id="timer-text-' + player.id + '">15</span></div>' : ''}
                ${isCurrentTurn ? '<span class="turn-badge">Playing</span>' : ''}
            </div>
        `;
        
        elements.gamePlayersList.appendChild(card);
    });
    
    // Start timer if it's someone's turn
    if (gameState.currentTurn && gameState.state === 'playing') {
        startTurnTimer();
    }
}

function updateUI() {
    const isMyTurn = gameState.currentTurn === gameState.playerId;
    
    // Update turn indicator
    if (isMyTurn) {
        elements.turnIndicator.textContent = 'Your turn!';
        elements.turnIndicator.classList.add('your-turn');
    } else {
        const currentPlayer = gameState.players[gameState.currentTurn];
        elements.turnIndicator.textContent = currentPlayer ? `${currentPlayer.name}'s turn` : 'Waiting...';
        elements.turnIndicator.classList.remove('your-turn');
    }
    
    // Update roll button
    elements.rollBtn.disabled = !isMyTurn || gameState.hasRolled;
    
    // Update valid moves
    updateValidMoves();
}

function updateValidMoves() {
    if (!gameState.hasRolled || gameState.validMoves.length === 0) {
        elements.validMovesDiv.style.display = 'none';
        
        // Auto-skip turn if rolled but no moves available
        if (gameState.hasRolled && gameState.validMoves.length === 0 && gameState.currentTurn === gameState.playerId) {
            // Show brief "No valid moves" message then auto-skip
            elements.validMovesDiv.style.display = 'flex';
            elements.pieceButtons.innerHTML = '<span style="color: var(--text-muted)">No valid moves! Skipping...</span>';
            elements.skipBtn.style.display = 'none';
            
            // Auto-skip after a short delay to let user see the message
            setTimeout(() => {
                if (gameState.hasRolled && gameState.validMoves.length === 0 && gameState.currentTurn === gameState.playerId) {
                    skipTurn();
                }
            }, 1000);
        }
        return;
    }
    
    // Auto-move if only one valid move
    if (gameState.validMoves.length === 1 && gameState.currentTurn === gameState.playerId) {
        elements.validMovesDiv.style.display = 'flex';
        elements.pieceButtons.innerHTML = '<span style="color: var(--text-muted)">Auto-moving piece ' + (gameState.validMoves[0] + 1) + '...</span>';
        elements.skipBtn.style.display = 'none';
        
        setTimeout(() => {
            if (gameState.validMoves.length === 1 && gameState.currentTurn === gameState.playerId) {
                movePiece(gameState.validMoves[0]);
            }
        }, 500);
        return;
    }
    
    elements.validMovesDiv.style.display = 'flex';
    elements.skipBtn.style.display = 'none';
    elements.pieceButtons.innerHTML = '';
    
    gameState.validMoves.forEach(pieceId => {
        const btn = document.createElement('button');
        btn.className = `piece-btn ${gameState.myColor}`;
        btn.textContent = pieceId + 1;
        btn.onclick = () => movePiece(pieceId);
        elements.pieceButtons.appendChild(btn);
    });
}

function showWaitingRoom() {
    elements.displayCode.textContent = gameState.code;
    showScreen('waiting');
    updateWaitingPlayersUI();
}

function showGameScreen() {
    showScreen('game');
    drawBoard();
    updateUI();
}

// ==================== Effects ====================
function showSixEffect() {
    playSound('diceSound');
    elements.dice.classList.add('six');
    setTimeout(() => elements.dice.classList.remove('six'), 500);
    
    // Show big 6
    const sixText = document.createElement('div');
    sixText.className = 'six-effect';
    sixText.textContent = '6!';
    document.body.appendChild(sixText);
    setTimeout(() => sixText.remove(), 1000);
    
    // Particles
    createParticles(window.innerWidth / 2, window.innerHeight / 2, '#f1c40f', 30);
}

function showKillEffect() {
    // Show CAPTURED text
    const killText = document.createElement('div');
    killText.className = 'kill-text';
    killText.textContent = '💀 CAPTURED!';
    document.body.appendChild(killText);
    setTimeout(() => killText.remove(), 1200);
    
    // Red particles
    createParticles(window.innerWidth / 2, window.innerHeight / 2, '#e74c3c', 40);
}

function createParticles(x, y, color, count) {
    for (let i = 0; i < count; i++) {
        const particle = document.createElement('div');
        particle.className = 'explosion-particle';
        particle.style.left = x + 'px';
        particle.style.top = y + 'px';
        particle.style.background = color;
        
        const angle = (Math.PI * 2 * i) / count;
        const distance = 50 + Math.random() * 100;
        const tx = Math.cos(angle) * distance;
        const ty = Math.sin(angle) * distance;
        
        particle.style.setProperty('--tx', tx + 'px');
        particle.style.setProperty('--ty', ty + 'px');
        
        elements.particles.appendChild(particle);
        setTimeout(() => particle.remove(), 800);
    }
}

function showWinner(winnerId) {
    // Stop the timer
    stopTurnTimer();
    
    const winner = gameState.players[winnerId];
    elements.winnerName.textContent = winner ? `${winner.name} wins!` : 'Game Over!';
    elements.winnerName.style.color = winner ? COLORS[winner.color].main : 'white';
    
    // Show rematch only for host
    elements.rematchBtn.style.display = gameState.isHost ? 'inline-flex' : 'none';
    
    elements.winnerModal.classList.add('active');
    playSound('winSound');
    
    // Confetti
    createConfetti();
}

function createConfetti() {
    const confettiContainer = document.getElementById('confetti');
    confettiContainer.innerHTML = '';
    
    const colors = ['#e74c3c', '#3498db', '#2ecc71', '#f1c40f', '#9b59b6'];
    
    for (let i = 0; i < 100; i++) {
        const confetti = document.createElement('div');
        confetti.className = 'confetti-piece';
        confetti.style.left = Math.random() * 100 + '%';
        confetti.style.background = colors[Math.floor(Math.random() * colors.length)];
        confetti.style.animationDelay = Math.random() * 2 + 's';
        confettiContainer.appendChild(confetti);
    }
}

function addChatMessage(sender, text, playerId) {
    const player = gameState.players[playerId];
    const colorClass = player ? player.color : '';
    
    const messageDiv = document.createElement('div');
    messageDiv.className = 'chat-message';
    messageDiv.innerHTML = `
        <span class="sender ${colorClass}">${sender}:</span>
        <span class="text">${escapeHtml(text)}</span>
    `;
    
    elements.chatMessages.appendChild(messageDiv);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

function addSystemMessage(text) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'chat-message system';
    messageDiv.innerHTML = `<span class="text">${text}</span>`;
    
    elements.chatMessages.appendChild(messageDiv);
    elements.chatMessages.scrollTop = elements.chatMessages.scrollHeight;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ==================== Timer Functions ====================
function startTurnTimer() {
    // Clear any existing timer
    stopTurnTimer();
    
    turnStartTime = Date.now();
    
    // Update timer display every 100ms
    timerInterval = setInterval(() => {
        const elapsed = Date.now() - turnStartTime;
        const remaining = Math.max(0, TURN_TIME_LIMIT - elapsed);
        const seconds = Math.ceil(remaining / 1000);
        const progress = remaining / TURN_TIME_LIMIT;
        
        // Update timer display for current player
        const timerText = document.getElementById(`timer-text-${gameState.currentTurn}`);
        const timerCircle = document.getElementById(`timer-circle-${gameState.currentTurn}`);
        
        if (timerText) {
            timerText.textContent = seconds;
            timerText.style.color = seconds <= 5 ? '#e74c3c' : '#fff';
        }
        if (timerCircle) {
            // Circle progress (circumference = 2 * PI * 16 ≈ 100.53)
            const circumference = 100.53;
            const offset = circumference * (1 - progress);
            timerCircle.style.strokeDashoffset = offset;
            timerCircle.style.stroke = seconds <= 5 ? '#e74c3c' : '#2ecc71';
        }
        
        // Handle timeout
        if (remaining <= 0 && gameState.currentTurn === gameState.playerId) {
            handleMyTimeout();
        }
    }, 100);
}

function stopTurnTimer() {
    if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
    }
    turnStartTime = null;
}

function handleMyTimeout() {
    stopTurnTimer();
    
    // Double-check it's still my turn before taking timeout action
    // (Server may have already processed the timeout and changed turn)
    if (gameState.currentTurn !== gameState.playerId) {
        return;
    }
    
    // Increment timeout count
    const myTimeouts = (playerTimeouts[gameState.playerId] || 0) + 1;
    playerTimeouts[gameState.playerId] = myTimeouts;
    
    if (myTimeouts >= MAX_TIMEOUTS) {
        showToast('Too many timeouts! You have been kicked.', 'error');
        leaveGame();
        return;
    }
    
    showToast(`Timeout! (${myTimeouts}/${MAX_TIMEOUTS})`, 'warning');
    
    // Auto roll if haven't rolled
    if (!gameState.hasRolled) {
        autoRollAndMove();
    } else if (gameState.validMoves.length > 0) {
        // Auto move first valid piece
        movePiece(gameState.validMoves[0]);
    } else {
        // No valid moves, skip
        skipTurn();
    }
    
    // Don't call updateGamePlayersUI here - it restarts the timer while async ops are pending
    // The timer will be restarted when websocket events arrive with updated game state
}

async function autoRollAndMove() {
    try {
        elements.rollBtn.disabled = true;
        
        const response = await apiCall('/api/game/roll', 'POST', {
            code: gameState.code,
            player_id: gameState.playerId
        });
        
        elements.diceFace.textContent = response.roll;
        gameState.lastDiceRoll = response.roll;
        gameState.hasRolled = true;
        gameState.validMoves = response.valid_moves;
        
        // Auto move or skip after short delay
        setTimeout(() => {
            // Verify it's still our turn before acting
            if (gameState.currentTurn !== gameState.playerId) {
                return; // Turn changed, don't act
            }
            if (gameState.validMoves.length > 0) {
                movePiece(gameState.validMoves[0]);
            } else {
                skipTurn();
            }
        }, 300);
        
    } catch (error) {
        // Reset button state on error - updateUI will handle proper state
        updateUI();
    }
}

// ==================== Board Drawing ====================
function drawBoard() {
    ctx.clearRect(0, 0, BOARD_SIZE, BOARD_SIZE);
    
    // Draw background (cream/beige)
    ctx.fillStyle = '#f5f5dc';
    ctx.fillRect(0, 0, BOARD_SIZE, BOARD_SIZE);
    
    if (currentBoardType === 'hex') {
        drawHexBoard();
    } else {
        drawSquareBoard();
    }
    
    // Draw pieces (works for both board types)
    drawAllPieces();
}

// ==================== Square Board Drawing (2-4 Players) ====================
function drawSquareBoard() {
    // Draw colored home areas based on server color assignments:
    // Blue: top-left, Green: top-right, Red: bottom-left, Yellow: bottom-right
    drawHomeArea(0, 0, 'blue');
    drawHomeArea(9 * CELL_SIZE, 0, 'green');
    drawHomeArea(0, 9 * CELL_SIZE, 'red');
    drawHomeArea(9 * CELL_SIZE, 9 * CELL_SIZE, 'yellow');
    
    // Draw path cells first (white background)
    drawSquarePath();
    
    // Draw home stretch paths (colored paths to center)
    drawSquareHomeStretches();
    
    // Draw center triangles
    drawSquareCenter();
    
    // Draw safe zones (stars)
    drawSquareSafeZones();
    
    // Draw start arrows
    drawSquareStartArrows();
}

function drawHomeArea(x, y, color) {
    const size = 6 * CELL_SIZE;
    const col = COLORS[color];
    
    // Background
    ctx.fillStyle = col.main;
    ctx.fillRect(x, y, size, size);
    
    // Border
    ctx.strokeStyle = col.dark;
    ctx.lineWidth = 3;
    ctx.strokeRect(x, y, size, size);
    
    // Inner white square with rounded corners for pawn slots
    const innerMargin = CELL_SIZE * 0.8;
    const innerSize = size - innerMargin * 2;
    
    ctx.fillStyle = 'white';
    ctx.strokeStyle = col.dark;
    ctx.lineWidth = 2;
    
    // Draw inner white area
    ctx.beginPath();
    ctx.roundRect(x + innerMargin, y + innerMargin, innerSize, innerSize, 8);
    ctx.fill();
    ctx.stroke();
    
    // Draw 4 colored circles for pawn positions
    const circleRadius = CELL_SIZE * 0.45;
    const positions = [
        {cx: x + size * 0.3, cy: y + size * 0.3},
        {cx: x + size * 0.7, cy: y + size * 0.3},
        {cx: x + size * 0.3, cy: y + size * 0.7},
        {cx: x + size * 0.7, cy: y + size * 0.7}
    ];
    
    positions.forEach(pos => {
        ctx.fillStyle = col.main;
        ctx.beginPath();
        ctx.arc(pos.cx, pos.cy, circleRadius, 0, Math.PI * 2);
        ctx.fill();
        ctx.strokeStyle = col.dark;
        ctx.lineWidth = 2;
        ctx.stroke();
    });
}

// ==================== Hexagonal Board Drawing (5-6 Players) ====================
// Matches the reference 6-player Ludo board image exactly
function drawHexBoard() {
    // White background
    ctx.fillStyle = '#FFFFFF';
    ctx.fillRect(0, 0, BOARD_SIZE, BOARD_SIZE);
    
    // Draw 6 colored home areas (rounded rectangles with 4 white circles)
    HEX_COLOR_ORDER.forEach((color, i) => {
        drawHexHomeArea(color, i);
    });
    
    // Draw the main track (white rectangular cells with borders)
    drawHexMainTrack();
    
    // Draw 6 colored home stretches (5 cells each leading to center)
    HEX_COLOR_ORDER.forEach((color, i) => {
        drawHexHomeStretchCells(color, i);
    });
    
    // Draw center hexagon with 6 colored triangles and dice
    drawHexCenter();
    
    // Draw safe zone stars
    drawHexSafeZones();
}

// Draw rounded rectangle home area with 4 white circles (matching reference)
function drawHexHomeArea(color, armIndex) {
    const col = COLORS[color];
    const angle = (armIndex * 60 - 90) * Math.PI / 180;
    
    // Home area position
    const homeRadius = BOARD_SIZE * 0.38;
    const cx = HEX_CENTER_X + Math.cos(angle) * homeRadius;
    const cy = HEX_CENTER_Y + Math.sin(angle) * homeRadius;
    
    // Draw colored rounded rectangle
    const rectW = BOARD_SIZE * 0.16;
    const rectH = BOARD_SIZE * 0.12;
    const cornerRadius = BOARD_SIZE * 0.02;
    
    ctx.save();
    ctx.translate(cx, cy);
    ctx.rotate(angle + Math.PI/2);
    
    // Main colored shape
    ctx.fillStyle = col.main;
    ctx.beginPath();
    roundedRect(ctx, -rectW/2, -rectH/2, rectW, rectH, cornerRadius);
    ctx.fill();
    ctx.strokeStyle = col.dark;
    ctx.lineWidth = 2;
    ctx.stroke();
    
    // Inner darker circle
    const innerRadius = Math.min(rectW, rectH) * 0.38;
    ctx.fillStyle = col.dark;
    ctx.beginPath();
    ctx.arc(0, 0, innerRadius, 0, Math.PI * 2);
    ctx.fill();
    
    // 4 white circles for pieces in 2x2 pattern
    const pieceRadius = BOARD_SIZE * 0.018;
    const spacing = pieceRadius * 2.0;
    const piecePositions = [
        { x: -spacing, y: -spacing },
        { x: spacing, y: -spacing },
        { x: -spacing, y: spacing },
        { x: spacing, y: spacing }
    ];
    
    piecePositions.forEach((pos, idx) => {
        ctx.fillStyle = 'white';
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, pieceRadius, 0, Math.PI * 2);
        ctx.fill();
        ctx.strokeStyle = '#aaa';
        ctx.lineWidth = 1;
        ctx.stroke();
    });
    
    ctx.restore();
    
    // Update HEX_HOME_POSITIONS with screen coordinates
    const cos = Math.cos(angle + Math.PI/2);
    const sin = Math.sin(angle + Math.PI/2);
    HEX_HOME_POSITIONS[color] = piecePositions.map(pos => ({
        x: cx + pos.x * cos - pos.y * sin,
        y: cy + pos.x * sin + pos.y * cos
    }));
}

// Helper function for rounded rectangle
function roundedRect(ctx, x, y, width, height, radius) {
    ctx.moveTo(x + radius, y);
    ctx.lineTo(x + width - radius, y);
    ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
    ctx.lineTo(x + width, y + height - radius);
    ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
    ctx.lineTo(x + radius, y + height);
    ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
    ctx.lineTo(x, y + radius);
    ctx.quadraticCurveTo(x, y, x + radius, y);
}

// Draw colored home stretch cells (5 rectangular cells per arm)
function drawHexHomeStretchCells(color, armIndex) {
    const col = COLORS[color];
    const angle = (armIndex * 60 - 90) * Math.PI / 180;
    
    const stretchPositions = HEX_HOME_STRETCH[color];
    if (!stretchPositions) return;
    
    const cellW = HEX_CELL_W;
    const cellH = HEX_CELL_H * 0.9;
    
    // Draw 5 colored cells
    for (let i = 0; i < 5; i++) {
        const pos = stretchPositions[i];
        
        ctx.save();
        ctx.translate(pos.x, pos.y);
        ctx.rotate(angle);
        
        // Colored cell with white border
        ctx.fillStyle = col.main;
        ctx.fillRect(-cellW/2, -cellH/2, cellW, cellH);
        ctx.strokeStyle = 'white';
        ctx.lineWidth = 2;
        ctx.strokeRect(-cellW/2, -cellH/2, cellW, cellH);
        
        ctx.restore();
    }
}

// Draw main track cells
function drawHexMainTrack() {
    const cellW = HEX_CELL_W;
    const cellH = HEX_CELL_H * 0.9;
    
    // Draw each track cell
    HEX_BOARD_POSITIONS.forEach((pos, i) => {
        const armIndex = Math.floor(i / 12);
        const posInArm = i % 12;
        const color = HEX_COLOR_ORDER[armIndex];
        const section = pos.section;
        
        // Start position (position 0) is colored
        const isStartPos = pos.isStart;
        
        // Calculate rotation based on section
        const armAngle = (armIndex * 60 - 90) * Math.PI / 180;
        const nextArmAngle = ((armIndex + 1) * 60 - 90) * Math.PI / 180;
        
        let cellAngle;
        if (section === 'left') {
            cellAngle = armAngle;
        } else if (section === 'corner') {
            const t = (posInArm - 3 + 0.5) / 3;
            cellAngle = armAngle + t * (Math.PI / 3);
        } else if (section === 'right') {
            cellAngle = nextArmAngle;
        } else { // junction
            cellAngle = nextArmAngle + Math.PI / 2;
        }
        
        ctx.save();
        ctx.translate(pos.x, pos.y);
        ctx.rotate(cellAngle);
        
        // Draw cell
        ctx.fillStyle = isStartPos ? COLORS[color].main : 'white';
        ctx.fillRect(-cellW/2, -cellH/2, cellW, cellH);
        ctx.strokeStyle = isStartPos ? COLORS[color].dark : '#888';
        ctx.lineWidth = 1;
        ctx.strokeRect(-cellW/2, -cellH/2, cellW, cellH);
        
        // Draw arrow on start position
        if (isStartPos) {
            ctx.fillStyle = 'white';
            const arrowSize = cellW * 0.3;
            ctx.beginPath();
            ctx.moveTo(0, -arrowSize);
            ctx.lineTo(-arrowSize * 0.6, arrowSize * 0.5);
            ctx.lineTo(arrowSize * 0.6, arrowSize * 0.5);
            ctx.closePath();
            ctx.fill();
        }
        
        ctx.restore();
    });
}

function drawHexCenter() {
    // Draw center hexagon with 6 colored triangles
    const centerRadius = BOARD_SIZE * 0.08;
    
    // Draw hexagonal background
    ctx.fillStyle = '#f0f0f0';
    ctx.strokeStyle = '#333';
    ctx.lineWidth = 2;
    ctx.beginPath();
    for (let i = 0; i < 6; i++) {
        const angle = (i * 60 - 90) * Math.PI / 180;
        const x = HEX_CENTER_X + Math.cos(angle) * centerRadius;
        const y = HEX_CENTER_Y + Math.sin(angle) * centerRadius;
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
    }
    ctx.closePath();
    ctx.fill();
    ctx.stroke();
    
    // Draw each colored triangle
    HEX_COLOR_ORDER.forEach((color, i) => {
        const angle1 = (i * 60 - 90) * Math.PI / 180;
        const angle2 = ((i + 1) * 60 - 90) * Math.PI / 180;
        
        ctx.fillStyle = COLORS[color].main;
        ctx.beginPath();
        ctx.moveTo(HEX_CENTER_X, HEX_CENTER_Y);
        ctx.lineTo(
            HEX_CENTER_X + Math.cos(angle1) * centerRadius,
            HEX_CENTER_Y + Math.sin(angle1) * centerRadius
        );
        ctx.lineTo(
            HEX_CENTER_X + Math.cos(angle2) * centerRadius,
            HEX_CENTER_Y + Math.sin(angle2) * centerRadius
        );
        ctx.closePath();
        ctx.fill();
        ctx.strokeStyle = COLORS[color].dark;
        ctx.lineWidth = 1;
        ctx.stroke();
    });
    
    // Draw center dice area (white rounded square with dots)
    const diceSize = centerRadius * 0.65;
    ctx.fillStyle = 'white';
    ctx.beginPath();
    roundedRect(ctx, HEX_CENTER_X - diceSize/2, HEX_CENTER_Y - diceSize/2, diceSize, diceSize, diceSize * 0.15);
    ctx.fill();
    ctx.strokeStyle = '#333';
    ctx.lineWidth = 1;
    ctx.stroke();
    
    // Draw dice dots (showing 6)
    const dotRadius = diceSize * 0.08;
    const dotOffset = diceSize * 0.22;
    const dotPositions = [
        {x: -dotOffset, y: -dotOffset}, {x: dotOffset, y: -dotOffset},
        {x: -dotOffset, y: 0}, {x: dotOffset, y: 0},
        {x: -dotOffset, y: dotOffset}, {x: dotOffset, y: dotOffset}
    ];
    ctx.fillStyle = '#333';
    dotPositions.forEach(dot => {
        ctx.beginPath();
        ctx.arc(HEX_CENTER_X + dot.x, HEX_CENTER_Y + dot.y, dotRadius, 0, Math.PI * 2);
        ctx.fill();
    });
}

function drawHexSafeZones() {
    // Draw stars on safe zone positions (not start positions)
    HEX_SAFE_ZONES.forEach(posIdx => {
        if (posIdx >= HEX_BOARD_POSITIONS.length) return;
        
        const pos = HEX_BOARD_POSITIONS[posIdx];
        if (!pos || pos.isStart) return; // Skip start positions
        
        const starSize = BOARD_SIZE * 0.012;
        
        ctx.fillStyle = '#ddd';
        ctx.strokeStyle = '#888';
        ctx.lineWidth = 1;
        drawStar(pos.x, pos.y, 5, starSize, starSize * 0.4);
        ctx.fill();
        ctx.stroke();
    });
}

function drawSquareCenter() {
    const x = 6 * CELL_SIZE;
    const y = 6 * CELL_SIZE;
    const size = 3 * CELL_SIZE;
    
    // Draw white background first
    ctx.fillStyle = 'white';
    ctx.fillRect(x, y, size, size);
    
    // Draw triangles for each color pointing to center
    // Each triangle corresponds to the home stretch that enters from that direction
    ctx.save();
    ctx.translate(x + size / 2, y + size / 2);
    
    // Order: top (green - enters from top), right (yellow - enters from right), 
    //        bottom (red - enters from bottom), left (blue - enters from left)
    const triangleColors = ['green', 'yellow', 'red', 'blue'];
    triangleColors.forEach((color, i) => {
        ctx.fillStyle = COLORS[color].main;
        ctx.beginPath();
        ctx.moveTo(0, 0);
        ctx.rotate(Math.PI / 2);
        ctx.lineTo(-size / 2, -size / 2);
        ctx.lineTo(size / 2, -size / 2);
        ctx.closePath();
        ctx.fill();
        ctx.strokeStyle = COLORS[color].dark;
        ctx.lineWidth = 2;
        ctx.stroke();
    });
    
    ctx.restore();
}

function drawSquarePath() {
    // Draw grid cells around the board
    ctx.strokeStyle = '#ccc';
    ctx.lineWidth = 1;
    
    // Draw all path cells
    for (let i = 0; i < SQUARE_BOARD_POSITIONS.length; i++) {
        const pos = SQUARE_BOARD_POSITIONS[i];
        const x = pos.x * CELL_SIZE;
        const y = pos.y * CELL_SIZE;
        
        ctx.fillStyle = 'white';
        ctx.fillRect(x, y, CELL_SIZE, CELL_SIZE);
        ctx.strokeRect(x, y, CELL_SIZE, CELL_SIZE);
    }
    
    // Draw the 4 corner cells adjacent to the center as white
    const centerCornerCells = [
        {x: 6, y: 6}, {x: 8, y: 6}, {x: 6, y: 8}, {x: 8, y: 8}
    ];
    centerCornerCells.forEach(pos => {
        ctx.fillStyle = 'white';
        ctx.fillRect(pos.x * CELL_SIZE, pos.y * CELL_SIZE, CELL_SIZE, CELL_SIZE);
        ctx.strokeStyle = '#ccc';
        ctx.lineWidth = 1;
        ctx.strokeRect(pos.x * CELL_SIZE, pos.y * CELL_SIZE, CELL_SIZE, CELL_SIZE);
    });
    
    // Color start positions (colored squares where pieces enter the board)
    const startColors = { 0: 'red', 13: 'blue', 26: 'yellow', 39: 'green' };
    Object.entries(startColors).forEach(([pos, color]) => {
        const p = SQUARE_BOARD_POSITIONS[parseInt(pos)];
        ctx.fillStyle = COLORS[color].main;
        ctx.fillRect(p.x * CELL_SIZE, p.y * CELL_SIZE, CELL_SIZE, CELL_SIZE);
        ctx.strokeStyle = COLORS[color].dark;
        ctx.lineWidth = 1;
        ctx.strokeRect(p.x * CELL_SIZE, p.y * CELL_SIZE, CELL_SIZE, CELL_SIZE);
    });
}

function drawSquareStartArrows() {
    // Draw arrows indicating entry direction for each color (square board only)
    const arrows = [
        { x: 6, y: 1, color: 'yellow', direction: 'down' },
        { x: 13, y: 6, color: 'green', direction: 'left' },
        { x: 8, y: 13, color: 'red', direction: 'up' },
        { x: 1, y: 8, color: 'blue', direction: 'right' }
    ];
    
    arrows.forEach(arrow => {
        const cx = arrow.x * CELL_SIZE + CELL_SIZE / 2;
        const cy = arrow.y * CELL_SIZE + CELL_SIZE / 2;
        const col = COLORS[arrow.color];
        
        ctx.fillStyle = col.dark;
        ctx.beginPath();
        
        const size = CELL_SIZE * 0.25;
        switch(arrow.direction) {
            case 'down':
                ctx.moveTo(cx, cy + size);
                ctx.lineTo(cx - size, cy - size);
                ctx.lineTo(cx + size, cy - size);
                break;
            case 'up':
                ctx.moveTo(cx, cy - size);
                ctx.lineTo(cx - size, cy + size);
                ctx.lineTo(cx + size, cy + size);
                break;
            case 'left':
                ctx.moveTo(cx - size, cy);
                ctx.lineTo(cx + size, cy - size);
                ctx.lineTo(cx + size, cy + size);
                break;
            case 'right':
                ctx.moveTo(cx + size, cy);
                ctx.lineTo(cx - size, cy - size);
                ctx.lineTo(cx - size, cy + size);
                break;
        }
        ctx.closePath();
        ctx.fill();
    });
}

function drawSquareSafeZones() {
    SQUARE_SAFE_ZONES.forEach(i => {
        if (i >= SQUARE_BOARD_POSITIONS.length) return;
        const pos = SQUARE_BOARD_POSITIONS[i];
        const x = pos.x * CELL_SIZE + CELL_SIZE / 2;
        const y = pos.y * CELL_SIZE + CELL_SIZE / 2;
        
        // Draw star
        ctx.fillStyle = '#ffd700';
        drawStar(x, y, 5, CELL_SIZE * 0.35, CELL_SIZE * 0.15);
    });
}

function drawStar(cx, cy, spikes, outerRadius, innerRadius) {
    let rot = Math.PI / 2 * 3;
    let x = cx;
    let y = cy;
    let step = Math.PI / spikes;

    ctx.beginPath();
    ctx.moveTo(cx, cy - outerRadius);
    
    for (let i = 0; i < spikes; i++) {
        x = cx + Math.cos(rot) * outerRadius;
        y = cy + Math.sin(rot) * outerRadius;
        ctx.lineTo(x, y);
        rot += step;

        x = cx + Math.cos(rot) * innerRadius;
        y = cy + Math.sin(rot) * innerRadius;
        ctx.lineTo(x, y);
        rot += step;
    }
    
    ctx.lineTo(cx, cy - outerRadius);
    ctx.closePath();
    ctx.fill();
}

function drawSquareHomeStretches() {
    Object.entries(SQUARE_HOME_STRETCH).forEach(([color, positions]) => {
        positions.forEach((pos, i) => {
            const isLastCell = (i === positions.length - 1);
            ctx.fillStyle = isLastCell ? COLORS[color].main : COLORS[color].light;
            
            const cellX = pos.x * CELL_SIZE;
            const cellY = pos.y * CELL_SIZE;
            
            ctx.fillRect(cellX, cellY, CELL_SIZE, CELL_SIZE);
            ctx.strokeStyle = COLORS[color].dark;
            ctx.lineWidth = 2;
            ctx.strokeRect(cellX, cellY, CELL_SIZE, CELL_SIZE);
        });
    });
}

function drawAllPieces() {
    // Group pieces by position for stacking
    const piecesByPosition = new Map();
    
    Object.values(gameState.players).forEach(player => {
        if (!player.pieces) return;
        
        player.pieces.forEach((piece, idx) => {
            const posKey = getPiecePositionKey(player.color, piece, idx);
            const pieceData = { color: player.color, piece, idx, posKey };
            
            if (!piecesByPosition.has(posKey)) {
                piecesByPosition.set(posKey, []);
            }
            piecesByPosition.get(posKey).push(pieceData);
        });
    });
    
    // Draw pieces, handling stacking
    piecesByPosition.forEach((piecesAtPos, posKey) => {
        const count = piecesAtPos.length;
        piecesAtPos.forEach((pieceData, stackIdx) => {
            drawPiece(pieceData.color, pieceData.piece, pieceData.idx, count, stackIdx);
        });
    });
}

function getPiecePositionKey(color, piece, idx) {
    if (piece.is_finished) {
        return `finish-${color}-${idx}`;
    } else if (piece.is_home) {
        return `home-${color}-${idx}`;
    } else if (piece.home_stretch_position > 0) {
        return `stretch-${color}-${piece.home_stretch_position}`;
    } else {
        return `board-${piece.position}`;
    }
}

function drawPiece(color, piece, idx, stackCount = 1, stackIdx = 0) {
    let targetX, targetY;
    
    if (piece.is_finished) {
        // In center/finish
        if (currentBoardType === 'hex') {
            const colorIdx = HEX_COLOR_ORDER.indexOf(color);
            const angle = ((colorIdx * 60) + 270 + idx * 12) * Math.PI / 180;
            targetX = HEX_CENTER_X + Math.cos(angle) * CELL_SIZE * 0.6;
            targetY = HEX_CENTER_Y + Math.sin(angle) * CELL_SIZE * 0.6;
        } else {
            const angle = (idx * Math.PI / 2) + (Object.keys(COLORS).indexOf(color) * Math.PI / 8);
            targetX = 7.5 * CELL_SIZE + Math.cos(angle) * CELL_SIZE * 0.8;
            targetY = 7.5 * CELL_SIZE + Math.sin(angle) * CELL_SIZE * 0.8;
        }
    } else if (piece.is_home) {
        // At home base - hex home positions are in pixels, square are in grid units
        const homePos = HOME_POSITIONS[color];
        if (homePos && homePos[idx]) {
            if (currentBoardType === 'hex') {
                // Hex positions are already in pixels
                targetX = homePos[idx].x;
                targetY = homePos[idx].y;
            } else {
                // Square positions are in grid units
                targetX = homePos[idx].x * CELL_SIZE + CELL_SIZE / 2;
                targetY = homePos[idx].y * CELL_SIZE + CELL_SIZE / 2;
            }
        } else {
            return;
        }
    } else if (piece.home_stretch_position > 0) {
        // In home stretch
        const stretchPos = HOME_STRETCH[color];
        if (stretchPos && stretchPos[piece.home_stretch_position - 1]) {
            const pos = stretchPos[piece.home_stretch_position - 1];
            if (currentBoardType === 'hex') {
                // Hex positions are in pixels
                targetX = pos.x;
                targetY = pos.y;
            } else {
                // Square positions are in grid units
                targetX = pos.x * CELL_SIZE + CELL_SIZE / 2;
                targetY = pos.y * CELL_SIZE + CELL_SIZE / 2;
            }
        } else {
            return;
        }
    } else {
        // On main board
        const pos = BOARD_POSITIONS[piece.position];
        if (pos) {
            if (currentBoardType === 'hex') {
                // Hex positions are in pixels
                targetX = pos.x;
                targetY = pos.y;
            } else {
                // Square positions are in grid units
                targetX = pos.x * CELL_SIZE + CELL_SIZE / 2;
                targetY = pos.y * CELL_SIZE + CELL_SIZE / 2;
            }
        } else {
            return;
        }
    }
    
    // Handle hopping animation
    const pieceKey = `${color}-${idx}`;
    let x = targetX, y = targetY;
    let hopHeight = 0;
    
    if (animatingPieces.has(pieceKey)) {
        const anim = animatingPieces.get(pieceKey);
        const waypoints = anim.waypoints;
        const currentIdx = anim.currentWaypoint;
        const progress = anim.progress;
        
        if (currentIdx < waypoints.length - 1) {
            const from = waypoints[currentIdx];
            const to = waypoints[currentIdx + 1];
            
            // Interpolate between waypoints
            x = from.x + (to.x - from.x) * progress;
            y = from.y + (to.y - from.y) * progress;
            
            // Add hop arc - piece jumps up in the middle of each hop
            hopHeight = Math.sin(progress * Math.PI) * CELL_SIZE * 0.6;
        } else {
            x = waypoints[waypoints.length - 1].x;
            y = waypoints[waypoints.length - 1].y;
        }
    }
    
    // Adjust position and size for stacked pieces
    let scale = 1;
    if (stackCount > 1) {
        // Reduce size based on stack count
        scale = stackCount === 2 ? 0.75 : stackCount === 3 ? 0.65 : 0.55;
        
        // Offset position based on stack index
        const offsetAngle = (stackIdx * 2 * Math.PI) / stackCount;
        const offsetDistance = PIECE_RADIUS * scale * 0.6;
        x += Math.cos(offsetAngle) * offsetDistance;
        y += Math.sin(offsetAngle) * offsetDistance;
    }
    
    // Store position for animation tracking
    lastPiecePositions.set(pieceKey, { x: targetX, y: targetY });
    
    // Draw chess pawn-like piece
    drawChessPawn(x, y - hopHeight, color, idx, scale);
}

function drawChessPawn(x, y, color, idx, scale = 1) {
    const col = COLORS[color];
    const baseWidth = CELL_SIZE * 0.7 * scale;
    const totalHeight = CELL_SIZE * 0.9 * scale;
    
    // Outer glow (for current player's pieces)
    if (color === gameState.myColor && gameState.currentTurn === gameState.playerId) {
        ctx.save();
        ctx.shadowColor = col.main;
        ctx.shadowBlur = 15;
        ctx.beginPath();
        ctx.arc(x, y - totalHeight * 0.3, baseWidth * 0.3, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(255,255,255,0.1)';
        ctx.fill();
        ctx.restore();
    }
    
    // Shadow on ground
    ctx.fillStyle = 'rgba(0,0,0,0.25)';
    ctx.beginPath();
    ctx.ellipse(x + 2, y + 3, baseWidth * 0.45, baseWidth * 0.2, 0, 0, Math.PI * 2);
    ctx.fill();
    
    // === PAWN BASE (wide bottom) ===
    const baseY = y;
    const baseHeight = totalHeight * 0.15;
    
    // Base gradient
    const baseGradient = ctx.createLinearGradient(x - baseWidth/2, baseY, x + baseWidth/2, baseY);
    baseGradient.addColorStop(0, col.dark);
    baseGradient.addColorStop(0.3, col.main);
    baseGradient.addColorStop(0.7, col.main);
    baseGradient.addColorStop(1, col.dark);
    
    ctx.fillStyle = baseGradient;
    ctx.beginPath();
    ctx.ellipse(x, baseY, baseWidth * 0.5, baseHeight, 0, 0, Math.PI * 2);
    ctx.fill();
    
    // Base rim highlight
    ctx.strokeStyle = col.light;
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.ellipse(x, baseY - 1, baseWidth * 0.48, baseHeight * 0.8, 0, Math.PI, Math.PI * 2);
    ctx.stroke();
    
    // === PAWN BODY (tapered column) ===
    const bodyBottomY = baseY - baseHeight * 0.5;
    const bodyTopY = baseY - totalHeight * 0.55;
    const bodyBottomWidth = baseWidth * 0.38;
    const bodyTopWidth = baseWidth * 0.25;
    
    // Body gradient
    const bodyGradient = ctx.createLinearGradient(x - bodyBottomWidth, bodyBottomY, x + bodyBottomWidth, bodyBottomY);
    bodyGradient.addColorStop(0, col.dark);
    bodyGradient.addColorStop(0.2, col.main);
    bodyGradient.addColorStop(0.5, col.light);
    bodyGradient.addColorStop(0.8, col.main);
    bodyGradient.addColorStop(1, col.dark);
    
    ctx.fillStyle = bodyGradient;
    ctx.beginPath();
    ctx.moveTo(x - bodyBottomWidth, bodyBottomY);
    ctx.quadraticCurveTo(x - bodyTopWidth * 1.1, (bodyBottomY + bodyTopY) / 2, x - bodyTopWidth, bodyTopY);
    ctx.lineTo(x + bodyTopWidth, bodyTopY);
    ctx.quadraticCurveTo(x + bodyTopWidth * 1.1, (bodyBottomY + bodyTopY) / 2, x + bodyBottomWidth, bodyBottomY);
    ctx.closePath();
    ctx.fill();
    
    // === COLLAR (small rim between body and head) ===
    const collarY = bodyTopY;
    const collarWidth = baseWidth * 0.3;
    
    ctx.fillStyle = col.main;
    ctx.beginPath();
    ctx.ellipse(x, collarY, collarWidth, totalHeight * 0.05, 0, 0, Math.PI * 2);
    ctx.fill();
    
    // === PAWN HEAD (sphere on top) ===
    const headY = baseY - totalHeight * 0.72;
    const headRadius = baseWidth * 0.28;
    
    // Head gradient for 3D sphere effect
    const headGradient = ctx.createRadialGradient(
        x - headRadius * 0.3, headY - headRadius * 0.3, 0,
        x, headY, headRadius
    );
    headGradient.addColorStop(0, col.light);
    headGradient.addColorStop(0.4, col.main);
    headGradient.addColorStop(1, col.dark);
    
    ctx.fillStyle = headGradient;
    ctx.beginPath();
    ctx.arc(x, headY, headRadius, 0, Math.PI * 2);
    ctx.fill();
    
    // Head highlight (glossy effect)
    ctx.fillStyle = 'rgba(255,255,255,0.4)';
    ctx.beginPath();
    ctx.ellipse(x - headRadius * 0.25, headY - headRadius * 0.3, headRadius * 0.3, headRadius * 0.2, -Math.PI / 4, 0, Math.PI * 2);
    ctx.fill();
    
    // === OUTLINE ===
    ctx.strokeStyle = col.dark;
    ctx.lineWidth = 1.5 * scale;
    
    // Draw full outline
    ctx.beginPath();
    // Base outline
    ctx.ellipse(x, baseY, baseWidth * 0.5, baseHeight, 0, 0, Math.PI);
    // Left side up
    ctx.lineTo(x - bodyBottomWidth, bodyBottomY);
    ctx.quadraticCurveTo(x - bodyTopWidth * 1.1, (bodyBottomY + bodyTopY) / 2, x - bodyTopWidth, bodyTopY);
    // Around head
    ctx.arc(x, headY, headRadius, Math.PI * 0.85, Math.PI * 0.15, true);
    // Right side down
    ctx.quadraticCurveTo(x + bodyTopWidth * 1.1, (bodyBottomY + bodyTopY) / 2, x + bodyBottomWidth, bodyBottomY);
    ctx.stroke();
    
    // === NUMBER ===
    ctx.fillStyle = 'white';
    ctx.font = `bold ${Math.floor(headRadius * 1.2)}px Nunito`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    
    ctx.save();
    ctx.shadowColor = 'rgba(0,0,0,0.7)';
    ctx.shadowBlur = 2;
    ctx.shadowOffsetX = 1;
    ctx.shadowOffsetY = 1;
    ctx.fillText(idx + 1, x, headY);
    ctx.restore();
}

// ==================== Reset ====================
function resetGameState() {
    // Stop any running timer
    stopTurnTimer();
    
    // Reset timeout tracking
    playerTimeouts = {};
    
    gameState = {
        code: null,
        playerId: null,
        playerName: null,
        isHost: false,
        players: {},
        currentTurn: null,
        lastDiceRoll: null,
        hasRolled: false,
        validMoves: [],
        myColor: null,
        state: 'waiting',
        ws: null
    };
    
    elements.readyCheckbox.checked = false;
    elements.chatMessages.innerHTML = '';
}

// ==================== Event Listeners ====================
// Tabs
document.querySelectorAll('.tab').forEach(tab => {
    tab.addEventListener('click', () => {
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
        tab.classList.add('active');
        document.getElementById(tab.dataset.tab + '-tab').classList.add('active');
    });
});

// Player count buttons
document.querySelectorAll('.player-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.player-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
    });
});

// Game actions
elements.createBtn.addEventListener('click', createGame);
elements.joinBtn.addEventListener('click', joinGame);
elements.readyCheckbox.addEventListener('change', (e) => setReady(e.target.checked));
elements.startBtn.addEventListener('click', startGame);
elements.leaveBtn.addEventListener('click', leaveGame);
elements.rollBtn.addEventListener('click', rollDice);
elements.skipBtn.addEventListener('click', skipTurn);
elements.sendChat.addEventListener('click', sendChat);
elements.chatInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') sendChat();
});
elements.rematchBtn.addEventListener('click', requestRematch);
elements.backLobbyBtn.addEventListener('click', () => {
    elements.winnerModal.classList.remove('active');
    leaveGame();
});

elements.copyCode.addEventListener('click', () => {
    navigator.clipboard.writeText(gameState.code);
    showToast('Code copied!', 'success');
});

// Enter key for inputs
elements.createName.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') createGame();
});
elements.joinName.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') elements.gameCode.focus();
});
elements.gameCode.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') joinGame();
});

// ==================== Animation Functions ====================
function animatePieceMovement(pieceKey, oldPos, oldState) {
    // Calculate new position and build waypoints
    const parts = pieceKey.split('-');
    const color = parts[0];
    const idx = parseInt(parts[1]);
    const player = Object.values(gameState.players).find(p => p.color === color);
    
    if (!player || !player.pieces || !player.pieces[idx]) return;
    
    const piece = player.pieces[idx];
    
    // Build waypoints following the actual board path
    const waypoints = buildBoardPathWaypoints(oldState, piece, color);
    
    if (waypoints.length < 2) return; // No movement
    
    // Start animation
    animatingPieces.set(pieceKey, {
        waypoints: waypoints,
        currentWaypoint: 0,
        progress: 0,
        startTime: performance.now()
    });
    
    // Start animation loop if not already running
    if (animatingPieces.size === 1) {
        requestAnimationFrame(updateAnimations);
    }
}

function getCellCenter(gridX, gridY) {
    return {
        x: gridX * CELL_SIZE + CELL_SIZE / 2,
        y: gridY * CELL_SIZE + CELL_SIZE / 2
    };
}

function buildBoardPathWaypoints(oldState, newPiece, color) {
    const waypoints = [];
    
    // Case 1: Piece coming out of home onto the board
    if (oldState.is_home && !newPiece.is_home) {
        // Add home position
        const homePos = HOME_POSITIONS[color];
        const idx = parseInt(Object.keys(HOME_POSITIONS[color]).find(i => 
            HOME_POSITIONS[color][i].x === homePos[0].x || true)); // Get any home pos
        const startHomeIdx = oldState.position || 0;
        if (homePos[startHomeIdx]) {
            waypoints.push(getCellCenter(homePos[startHomeIdx].x, homePos[startHomeIdx].y));
        }
        // Add start position on board
        const startPos = BOARD_POSITIONS[START_POSITIONS[color]];
        if (startPos) {
            waypoints.push(getCellCenter(startPos.x, startPos.y));
        }
        return waypoints;
    }
    
    // Case 2: Piece moving to finish
    if (!oldState.is_finished && newPiece.is_finished) {
        // Add last home stretch position
        if (oldState.home_stretch_position > 0) {
            const stretchPos = HOME_STRETCH[color][oldState.home_stretch_position - 1];
            if (stretchPos) {
                waypoints.push(getCellCenter(stretchPos.x, stretchPos.y));
            }
        }
        // Add center finish position
        const angle = (0 * Math.PI / 2) + (Object.keys(COLORS).indexOf(color) * Math.PI / 8);
        waypoints.push({
            x: 7.5 * CELL_SIZE + Math.cos(angle) * CELL_SIZE * 0.8,
            y: 7.5 * CELL_SIZE + Math.sin(angle) * CELL_SIZE * 0.8
        });
        return waypoints;
    }
    
    // Case 3: Moving within home stretch
    if (oldState.home_stretch_position > 0 && newPiece.home_stretch_position > 0) {
        for (let i = oldState.home_stretch_position; i <= newPiece.home_stretch_position; i++) {
            const pos = HOME_STRETCH[color][i - 1];
            if (pos) {
                waypoints.push(getCellCenter(pos.x, pos.y));
            }
        }
        return waypoints;
    }
    
    // Case 4: Entering home stretch from board
    if (oldState.home_stretch_position === 0 && newPiece.home_stretch_position > 0) {
        // Add current board position
        const boardPos = BOARD_POSITIONS[oldState.position];
        if (boardPos) {
            waypoints.push(getCellCenter(boardPos.x, boardPos.y));
        }
        // Add home stretch positions
        for (let i = 1; i <= newPiece.home_stretch_position; i++) {
            const pos = HOME_STRETCH[color][i - 1];
            if (pos) {
                waypoints.push(getCellCenter(pos.x, pos.y));
            }
        }
        return waypoints;
    }
    
    // Case 5: Normal board movement
    if (!oldState.is_home && !newPiece.is_home && oldState.home_stretch_position === 0 && newPiece.home_stretch_position === 0) {
        const startIdx = oldState.position;
        const endIdx = newPiece.position;
        
        // Follow the board path
        let currentIdx = startIdx;
        waypoints.push(getCellCenter(BOARD_POSITIONS[currentIdx].x, BOARD_POSITIONS[currentIdx].y));
        
        // Move forward along the path (wrapping around 52)
        while (currentIdx !== endIdx) {
            currentIdx = (currentIdx + 1) % 52;
            const pos = BOARD_POSITIONS[currentIdx];
            if (pos) {
                waypoints.push(getCellCenter(pos.x, pos.y));
            }
            // Safety: prevent infinite loop
            if (waypoints.length > 52) break;
        }
        
        return waypoints;
    }
    
    // Fallback: direct movement
    if (oldState.is_home) {
        const homePos = HOME_POSITIONS[color][0];
        if (homePos) {
            waypoints.push(getCellCenter(homePos.x, homePos.y));
        }
    } else if (oldState.home_stretch_position > 0) {
        const pos = HOME_STRETCH[color][oldState.home_stretch_position - 1];
        if (pos) {
            waypoints.push(getCellCenter(pos.x, pos.y));
        }
    } else {
        const pos = BOARD_POSITIONS[oldState.position];
        if (pos) {
            waypoints.push(getCellCenter(pos.x, pos.y));
        }
    }
    
    // Add end position
    if (newPiece.is_finished) {
        const angle = (0 * Math.PI / 2) + (Object.keys(COLORS).indexOf(color) * Math.PI / 8);
        waypoints.push({
            x: 7.5 * CELL_SIZE + Math.cos(angle) * CELL_SIZE * 0.8,
            y: 7.5 * CELL_SIZE + Math.sin(angle) * CELL_SIZE * 0.8
        });
    } else if (newPiece.home_stretch_position > 0) {
        const pos = HOME_STRETCH[color][newPiece.home_stretch_position - 1];
        if (pos) {
            waypoints.push(getCellCenter(pos.x, pos.y));
        }
    } else {
        const pos = BOARD_POSITIONS[newPiece.position];
        if (pos) {
            waypoints.push(getCellCenter(pos.x, pos.y));
        }
    }
    
    return waypoints;
}

function updateAnimations(currentTime) {
    let hasActiveAnimations = false;
    
    animatingPieces.forEach((anim, key) => {
        const elapsed = currentTime - anim.startTime;
        const totalHops = anim.waypoints.length - 1;
        const hopIndex = Math.floor(elapsed / HOP_DURATION);
        
        if (hopIndex >= totalHops) {
            // Animation complete
            anim.currentWaypoint = totalHops - 1;
            anim.progress = 1;
            animatingPieces.delete(key);
        } else {
            anim.currentWaypoint = hopIndex;
            anim.progress = (elapsed % HOP_DURATION) / HOP_DURATION;
            hasActiveAnimations = true;
        }
    });
    
    drawBoard();
    
    if (hasActiveAnimations || animatingPieces.size > 0) {
        requestAnimationFrame(updateAnimations);
    }
}

// Initialize
console.log('🎲 Ludo Nadwa loaded!');
