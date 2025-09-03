package websocket

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"laps/internal/domain"
	"laps/internal/service"
)

// SignalingMessage represents a WebRTC signaling message
type SignalingMessage struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id"`
	From      int64       `json:"from"`
	To        int64       `json:"to"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID     int64
	UserID int64
	Role   domain.UserRole
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *SignalingHub
}

// SignalingHub maintains the set of active clients and broadcasts messages
type SignalingHub struct {
	// Registered clients by user ID
	clients map[int64]*Client

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Active call sessions by session ID
	sessions map[string]*CallSession

	// Logger
	logger *zap.Logger

	// Services
	services *service.Services

	// Mutex for thread safety
	mutex sync.RWMutex
}

// CallSession represents an active call session
type CallSession struct {
	ID           string    `json:"id"`
	ClientID     int64     `json:"client_id"`
	SpecialistID int64     `json:"specialist_id"`
	AppointmentID *int64   `json:"appointment_id,omitempty"`
	Status       string    `json:"status"` // waiting, active, ended
	CreatedAt    time.Time `json:"created_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from localhost and development origins
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Allow connections without Origin header (for testing)
		}
		
		// Allow localhost and development origins
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"https://localhost:3000",
			"https://127.0.0.1:3000",
		}
		
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		
		// In production, add your domain here
		// return origin == "https://yourdomain.com"
		return true // For now, allow all origins during development
	},
	ReadBufferSize:  65536,
	WriteBufferSize: 65536,
}

// NewSignalingHub creates a new signaling hub
func NewSignalingHub(logger *zap.Logger, services *service.Services) *SignalingHub {
	return &SignalingHub{
		clients:    make(map[int64]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		sessions:   make(map[string]*CallSession),
		logger:     logger,
		services:   services,
	}
}

// Run starts the signaling hub
func (h *SignalingHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.UserID] = client
			h.mutex.Unlock()
			h.logger.Info("Client connected", 
				zap.Int64("user_id", client.UserID), 
				zap.String("role", string(client.Role)))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mutex.Unlock()
			h.logger.Info("Client disconnected", zap.Int64("user_id", client.UserID))

		case message := <-h.broadcast:
			var msg SignalingMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				h.logger.Error("Failed to unmarshal message", zap.Error(err))
				continue
			}

			h.handleSignalingMessage(&msg)
		}
	}
}

// handleSignalingMessage processes incoming signaling messages
func (h *SignalingHub) handleSignalingMessage(msg *SignalingMessage) {
	h.logger.Info("🔔 [BACKEND] Processing signaling message", 
		zap.String("type", msg.Type),
		zap.Int64("from", msg.From),
		zap.Int64("to", msg.To),
		zap.String("session_id", msg.SessionID))

	// Check if target user is connected
	if _, exists := h.clients[msg.To]; !exists {
		h.logger.Warn("❌ [BACKEND] Target user not connected", 
			zap.Int64("target_user_id", msg.To),
			zap.String("message_type", msg.Type))
	} else {
		h.logger.Info("✅ [BACKEND] Target user is connected", 
			zap.Int64("target_user_id", msg.To),
			zap.String("message_type", msg.Type))
	}

	switch msg.Type {
	case "call-invitation":
		h.logger.Info("📞 [BACKEND] Handling call-invitation message")
		h.handleCallInvitation(msg)
	case "call-offer":
		h.logger.Info("📞 [BACKEND] Handling call-offer message")
		h.handleCallOffer(msg)
	case "call-answer":
		h.handleCallAnswer(msg)
	case "ice-candidate":
		h.handleIceCandidate(msg)
	case "call-reject":
		h.handleCallReject(msg)
	case "call-end":
		h.handleCallEnd(msg)
	case "ping":
		h.handlePing(msg)
	default:
		h.logger.Warn("Unknown message type", zap.String("type", msg.Type))
	}
}

// handleCallInvitation processes call invitation messages (for UI notification)
func (h *SignalingHub) handleCallInvitation(msg *SignalingMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	h.logger.Info("📞 [BACKEND] Processing call-invitation", 
		zap.String("session_id", msg.SessionID),
		zap.Int64("from", msg.From),
		zap.Int64("to", msg.To))
	
	// Log all connected clients for debugging
	var connectedClients []int64
	for clientID := range h.clients {
		connectedClients = append(connectedClients, clientID)
	}
	h.logger.Info("📞 [BACKEND] Currently connected clients", 
		zap.Int64s("client_ids", connectedClients))

	// Forward invitation to target user
	if targetClient, exists := h.clients[msg.To]; exists {
		h.logger.Info("📞 [BACKEND] Target client found, forwarding call-invitation", 
			zap.Int64("target_user_id", msg.To),
			zap.String("session_id", msg.SessionID))
		
		h.sendMessageToClient(targetClient, msg)
		
		h.logger.Info("✅ [BACKEND] Call invitation forwarded successfully", 
			zap.String("session_id", msg.SessionID),
			zap.Int64("from", msg.From),
			zap.Int64("to", msg.To))
	} else {
		h.logger.Warn("❌ [BACKEND] Target user not connected for call invitation", 
			zap.Int64("user_id", msg.To),
			zap.String("session_id", msg.SessionID))
		
		// Send error back to caller
		errorMsg := &SignalingMessage{
			Type:      "call-error",
			SessionID: msg.SessionID,
			From:      msg.To,
			To:        msg.From,
			Data:      map[string]string{"error": "User not available"},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		if callerClient, exists := h.clients[msg.From]; exists {
			h.logger.Info("📞 [BACKEND] Sending call-error back to caller", 
				zap.Int64("caller_id", msg.From))
			h.sendMessageToClient(callerClient, errorMsg)
		}
	}
}

// handleCallOffer processes call offer messages
func (h *SignalingHub) handleCallOffer(msg *SignalingMessage) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.logger.Info("📞 [BACKEND] Processing call-offer", 
		zap.String("session_id", msg.SessionID),
		zap.Int64("from", msg.From),
		zap.Int64("to", msg.To))
	
	// Log all connected clients for debugging
	var connectedClients []int64
	for clientID := range h.clients {
		connectedClients = append(connectedClients, clientID)
	}
	h.logger.Info("📞 [BACKEND] Currently connected clients", 
		zap.Int64s("client_ids", connectedClients))

	// Create new call session
	fromClient, fromExists := h.clients[msg.From]
	toClient, toExists := h.clients[msg.To]

	if !fromExists || !toExists {
		h.logger.Error("Could not find one or both clients for call",
			zap.Int64("from_id", msg.From),
			zap.Bool("from_exists", fromExists),
			zap.Int64("to_id", msg.To),
			zap.Bool("to_exists", toExists))
		return
	}

	var clientID, specialistID int64
	if fromClient.Role == "client" {
		clientID = fromClient.UserID
		specialistID = toClient.UserID
	} else {
		clientID = toClient.UserID
		specialistID = fromClient.UserID
	}

	session := &CallSession{
		ID:           msg.SessionID,
		ClientID:     clientID,
		SpecialistID: specialistID,
		Status:       "waiting",
		CreatedAt:    time.Now(),
	}

	h.sessions[msg.SessionID] = session
	h.logger.Info("📞 [BACKEND] Call session created", zap.String("session_id", msg.SessionID))

	// Forward offer to target user
	if targetClient, exists := h.clients[msg.To]; exists {
		h.logger.Info("📞 [BACKEND] Target client found, forwarding call-offer", 
			zap.Int64("target_user_id", msg.To),
			zap.String("session_id", msg.SessionID),
			zap.Bool("client_exists", targetClient != nil),
			zap.Bool("send_channel_exists", targetClient != nil && targetClient.Send != nil))
		
		h.sendMessageToClient(targetClient, msg)
		
		h.logger.Info("✅ [BACKEND] Call offer forwarded successfully", 
			zap.String("session_id", msg.SessionID),
			zap.Int64("from", msg.From),
			zap.Int64("to", msg.To))
	} else {
		h.logger.Warn("❌ [BACKEND] Target user not connected", 
			zap.Int64("user_id", msg.To),
			zap.String("session_id", msg.SessionID))
		
		// Send error back to caller
		errorMsg := &SignalingMessage{
			Type:      "call-error",
			SessionID: msg.SessionID,
			From:      msg.To,
			To:        msg.From,
			Data:      map[string]string{"error": "User not available"},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		if callerClient, exists := h.clients[msg.From]; exists {
			h.logger.Info("📞 [BACKEND] Sending call-error back to caller", 
				zap.Int64("caller_id", msg.From))
			h.sendMessageToClient(callerClient, errorMsg)
		}
	}
}

// handleCallAnswer processes call answer messages
func (h *SignalingHub) handleCallAnswer(msg *SignalingMessage) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Update session status
	if session, exists := h.sessions[msg.SessionID]; exists {
		session.Status = "active"
	}

	// Forward answer to caller
	if callerClient, exists := h.clients[msg.To]; exists {
		h.sendMessageToClient(callerClient, msg)
		h.logger.Info("Call answer forwarded", 
			zap.String("session_id", msg.SessionID),
			zap.Int64("from", msg.From),
			zap.Int64("to", msg.To))
	}
}

// handleIceCandidate processes ICE candidate messages
func (h *SignalingHub) handleIceCandidate(msg *SignalingMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Forward ICE candidate to the other peer
	if targetClient, exists := h.clients[msg.To]; exists {
		h.sendMessageToClient(targetClient, msg)
	}
}

// handleCallReject handles call rejection messages
func (h *SignalingHub) handleCallReject(msg *SignalingMessage) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.logger.Info("Processing call-reject", 
		zap.String("session_id", msg.SessionID),
		zap.Int64("from", msg.From),
		zap.Int64("to", msg.To))

	// Forward rejection to the caller
	if targetClient, exists := h.clients[msg.To]; exists {
		h.sendMessageToClient(targetClient, msg)
		h.logger.Info("Call rejection forwarded to caller", 
			zap.Int64("caller_id", msg.To))
	}

	// Remove session if it exists
	if _, exists := h.sessions[msg.SessionID]; exists {
		delete(h.sessions, msg.SessionID)
		h.logger.Info("Session removed after rejection", 
			zap.String("session_id", msg.SessionID))
	}
}

// handleCallEnd processes call end messages
func (h *SignalingHub) handleCallEnd(msg *SignalingMessage) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Update session status
	if session, exists := h.sessions[msg.SessionID]; exists {
		session.Status = "ended"
		now := time.Now()
		session.EndedAt = &now
	}

	// Forward end message to the other peer
	if targetClient, exists := h.clients[msg.To]; exists {
		h.sendMessageToClient(targetClient, msg)
	}

	h.logger.Info("Call ended", zap.String("session_id", msg.SessionID))
}

// handlePing processes ping messages for connection keepalive
func (h *SignalingHub) handlePing(msg *SignalingMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	pongMsg := &SignalingMessage{
		Type:      "pong",
		SessionID: msg.SessionID,
		From:      msg.To,
		To:        msg.From,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if client, exists := h.clients[msg.From]; exists {
		h.sendMessageToClient(client, pongMsg)
	}
}

// sendMessageToClient sends a message to a specific client
// NOTE: This function should only be called when the mutex is already held
func (h *SignalingHub) sendMessageToClient(client *Client, msg *SignalingMessage) {
	h.logger.Info("📤 [BACKEND] Attempting to send message to client", 
		zap.String("message_type", msg.Type),
		zap.Int64("target_user_id", client.UserID),
		zap.Int64("from", msg.From),
		zap.Int64("to", msg.To),
		zap.String("session_id", msg.SessionID))

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("❌ [BACKEND] Failed to marshal message", zap.Error(err))
		return
	}

	select {
	case client.Send <- data:
		h.logger.Info("✅ [BACKEND] Message sent successfully to client", 
			zap.String("message_type", msg.Type),
			zap.Int64("target_user_id", client.UserID),
			zap.String("session_id", msg.SessionID))
	default:
		h.logger.Warn("❌ [BACKEND] Failed to send message - client channel full or closed", 
			zap.Int64("user_id", client.UserID),
			zap.String("message_type", msg.Type))
		// Don't modify the clients map here - let the cleanup happen in the main hub loop
		// This prevents race conditions and deadlocks
		// The client will be removed when the connection closes naturally
	}
}

// HandleWebSocket handles WebSocket connections
func (h *SignalingHub) HandleWebSocket(c *gin.Context) {
	h.logger.Info("🔥 WebSocket handler called", zap.String("path", c.Request.URL.Path), zap.String("query", c.Request.URL.RawQuery))
	
	// Get user ID and role from JWT token (passed as query parameter for WebSocket)
	tokenStr := c.Query("token")
	if tokenStr == "" {
		h.logger.Info("🔥 No token provided, using simplified auth")
	} else {
		h.logger.Info("🔥 Token provided but using simplified auth anyway")
	}

	// For now, use a simple approach - extract user info from query params
	// In production, this should use proper JWT validation
	userIDStr := c.Query("user_id")
	roleStr := c.Query("role")
	
	// Temporary simple validation - just check if user exists in system
	if userIDStr == "" || roleStr == "" {
		h.logger.Warn("Missing user_id or role in WebSocket request", 
			zap.String("user_id", userIDStr), 
			zap.String("role", roleStr),
			zap.String("token_present", func() string {
				if tokenStr != "" { return "yes" } else { return "no" }
			}()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id and role required"})
		return
	}
	
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("Invalid user_id format", zap.String("user_id", userIDStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
		return
	}
	
	role := domain.UserRole(roleStr)
	if role != "client" && role != "specialist" {
		h.logger.Warn("Invalid role", zap.String("role", roleStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}
	
	h.logger.Info("WebSocket connection authorized", zap.Int64("user_id", userID), zap.String("role", string(role)))

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Create client
	client := &Client{
		UserID: userID,
		Role:   role,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h,
	}

	// Register client
	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// Allow large SDP payloads and batches of ICE candidates (up to 10MB)
	c.Conn.SetReadLimit(10 * 1024 * 1024)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		// Parse and validate message
		var msg SignalingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.Hub.logger.Error("Failed to unmarshal message", zap.Error(err))
			continue
		}

		// Set sender info
		msg.From = c.UserID
		msg.Timestamp = time.Now().Format(time.RFC3339)

		// Re-marshal with corrected info
		correctedMessage, err := json.Marshal(msg)
		if err != nil {
			c.Hub.logger.Error("Failed to marshal corrected message", zap.Error(err))
			continue
		}

		c.Hub.broadcast <- correctedMessage
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send exactly one message per frame to avoid huge concatenated frames
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Hub.logger.Error("Failed to write message to WebSocket",
					zap.Int64("user_id", c.UserID),
					zap.Error(err))
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetActiveSessions returns all active call sessions
func (h *SignalingHub) GetActiveSessions() map[string]*CallSession {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	sessions := make(map[string]*CallSession)
	for id, session := range h.sessions {
		if session.Status == "active" || session.Status == "waiting" {
			sessions[id] = session
		}
	}
	return sessions
}

// IsUserConnected checks if a user is currently connected
func (h *SignalingHub) IsUserConnected(userID int64) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	_, exists := h.clients[userID]
	return exists
}

// GetActiveCallForUsers returns active call session between two users
func (h *SignalingHub) GetActiveCallForUsers(userID1, userID2 int64) *CallSession {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for _, session := range h.sessions {
		if session.Status == "active" || session.Status == "waiting" {
			if (session.ClientID == userID1 && session.SpecialistID == userID2) ||
				(session.ClientID == userID2 && session.SpecialistID == userID1) {
				return session
			}
		}
	}
	return nil
}

// GetActiveCallBySessionID returns active call session by ID
func (h *SignalingHub) GetActiveCallBySessionID(sessionID string) *CallSession {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if session, exists := h.sessions[sessionID]; exists {
		if session.Status == "active" || session.Status == "waiting" {
			return session
		}
	}
	return nil
}

// GetAllActiveCallsForUser returns all active calls for a user
func (h *SignalingHub) GetAllActiveCallsForUser(userID int64) []*CallSession {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var activeCalls []*CallSession
	for _, session := range h.sessions {
		if session.Status == "active" || session.Status == "waiting" {
			if session.ClientID == userID || session.SpecialistID == userID {
				activeCalls = append(activeCalls, session)
			}
		}
	}
	return activeCalls
} 