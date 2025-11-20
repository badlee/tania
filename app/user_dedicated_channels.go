package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	webrtc "github.com/pion/webrtc/v3"
	"github.com/pocketbase/pocketbase/core"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// ==================== USER CHANNEL TYPES ====================

// SSEChannel - Canal SSE pour chaque utilisateur
type SSEChannel struct {
	UserID   string
	Channel  chan SSEMessage
	IsActive bool
	mu       sync.RWMutex
}

type SSEMessage struct {
	Type      string                 `json:"type"`
	RequestID string                 `json:"request_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// UserRoom - Room WebRTC dédiée à chaque utilisateur
type UserRoom struct {
	UserID       string
	Participant  *Participant
	DataChannel  *webrtc.DataChannel
	IsConnected  bool
	LastActivity time.Time
	mu           sync.RWMutex
}

// APIRequest - Requête API via WebRTC DataChannel
type APIRequest struct {
	RequestID string                 `msgpack:"request_id"`
	Method    string                 `msgpack:"method"` // GET, POST, PATCH, DELETE
	Endpoint  string                 `msgpack:"endpoint"`
	Body      map[string]interface{} `msgpack:"body,omitempty"`
	Query     map[string]string      `msgpack:"query,omitempty"`
}

type APIResponse struct {
	RequestID  string                 `msgpack:"request_id"`
	StatusCode int                    `msgpack:"status_code"`
	Data       map[string]interface{} `msgpack:"data"`
	Error      string                 `msgpack:"error,omitempty"`
	Timestamp  int64                  `msgpack:"timestamp"`
}

// ==================== USER CHANNEL MANAGER ====================

type UserChannelManager struct {
	sseChannels map[string]*SSEChannel
	userRooms   map[string]*UserRoom
	app         core.App
	mu          sync.RWMutex
}

func NewUserChannelManager(app core.App) *UserChannelManager {
	return &UserChannelManager{
		sseChannels: make(map[string]*SSEChannel),
		userRooms:   make(map[string]*UserRoom),
		app:         app,
	}
}

// ==================== SSE CHANNEL MANAGEMENT ====================

// Create or get SSE channel for user
func (ucm *UserChannelManager) GetOrCreateSSEChannel(userID string) *SSEChannel {
	ucm.mu.Lock()
	defer ucm.mu.Unlock()

	if channel, exists := ucm.sseChannels[userID]; exists {
		return channel
	}

	channel := &SSEChannel{
		UserID:   userID,
		Channel:  make(chan SSEMessage, 100),
		IsActive: true,
	}

	ucm.sseChannels[userID] = channel
	log.Printf("📡 SSE Channel created for user: %s", userID)

	return channel
}

// Send message to user's SSE channel
func (ucm *UserChannelManager) SendToSSE(userID string, msgType string, data map[string]interface{}, requestID string) {
	ucm.mu.RLock()
	channel, exists := ucm.sseChannels[userID]
	ucm.mu.RUnlock()

	if !exists || !channel.IsActive {
		return
	}

	message := SSEMessage{
		Type:      msgType,
		RequestID: requestID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	select {
	case channel.Channel <- message:
	default:
		log.Printf("⚠️  SSE channel full for user %s", userID)
	}
}

// Close SSE channel
func (ucm *UserChannelManager) CloseSSEChannel(userID string) {
	ucm.mu.Lock()
	defer ucm.mu.Unlock()

	if channel, exists := ucm.sseChannels[userID]; exists {
		channel.IsActive = false
		close(channel.Channel)
		delete(ucm.sseChannels, userID)
		log.Printf("🔌 SSE Channel closed for user: %s", userID)
	}
}

// ==================== USER ROOM (WebRTC) MANAGEMENT ====================

// Create or get user's dedicated room
func (ucm *UserChannelManager) GetOrCreateUserRoom(userID string) (*UserRoom, error) {
	ucm.mu.Lock()
	defer ucm.mu.Unlock()

	if room, exists := ucm.userRooms[userID]; exists {
		return room, nil
	}

	room := &UserRoom{
		UserID:       userID,
		IsConnected:  false,
		LastActivity: time.Now(),
	}

	ucm.userRooms[userID] = room
	log.Printf("🎯 User Room created for: %s", userID)

	return room, nil
}

// Connect user to their dedicated room
func (ucm *UserChannelManager) ConnectToUserRoom(userID string) (*webrtc.SessionDescription, error) {
	room, err := ucm.GetOrCreateUserRoom(userID)
	if err != nil {
		return nil, err
	}

	// Create peer connection
	pc, err := createPeerConnection()
	if err != nil {
		return nil, err
	}

	participantID := generateID()
	participant := &Participant{
		ID:       participantID,
		PeerConn: pc,
		UserID:   userID,
	}

	// Create DataChannel
	dc, err := pc.CreateDataChannel("api", nil)
	if err != nil {
		return nil, err
	}

	participant.DataChannel = dc

	// Setup DataChannel handlers
	dc.OnOpen(func() {
		log.Printf("✅ User Room DataChannel opened for: %s", userID)
		room.mu.Lock()
		room.IsConnected = true
		room.LastActivity = time.Now()
		room.mu.Unlock()

		// Send welcome message
		ucm.SendToUserRoom(userID, "welcome", map[string]interface{}{
			"message": "Connected to your dedicated room",
			"user_id": userID,
		}, "")

		// Update presence to online
		if locationManager != nil {
			userLoc, exists := locationManager.GetLocation(userID)
			if exists {
				locationManager.UpdateLocation(userID, userLoc.Location, "online")
			}
		}
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		room.mu.Lock()
		room.LastActivity = time.Now()
		room.mu.Unlock()

		ucm.handleUserRoomMessage(userID, msg.Data)
	})

	dc.OnClose(func() {
		log.Printf("🔌 User Room DataChannel closed for: %s", userID)
		room.mu.Lock()
		room.IsConnected = false
		room.mu.Unlock()

		// Update presence to offline
		if locationManager != nil {
			userLoc, exists := locationManager.GetLocation(userID)
			if exists {
				locationManager.UpdateLocation(userID, userLoc.Location, "offline")
			}
		}
	})

	room.mu.Lock()
	room.Participant = participant
	room.DataChannel = dc
	room.mu.Unlock()

	// Create offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		return nil, err
	}

	return &offer, nil
}

// Handle answer from client
func (ucm *UserChannelManager) HandleUserRoomAnswer(userID string, answer webrtc.SessionDescription) error {
	ucm.mu.RLock()
	room, exists := ucm.userRooms[userID]
	ucm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("user room not found")
	}

	room.mu.RLock()
	pc := room.Participant.PeerConn
	room.mu.RUnlock()

	return pc.SetRemoteDescription(answer)
}

// Send message to user's room via DataChannel
func (ucm *UserChannelManager) SendToUserRoom(userID string, msgType string, data map[string]interface{}, requestID string) {
	ucm.mu.RLock()
	room, exists := ucm.userRooms[userID]
	ucm.mu.RUnlock()

	if !exists || !room.IsConnected {
		return
	}

	room.mu.RLock()
	dc := room.DataChannel
	room.mu.RUnlock()

	if dc == nil || dc.ReadyState() != webrtc.DataChannelStateOpen {
		return
	}

	response := map[string]interface{}{
		"type":       msgType,
		"request_id": requestID,
		"data":       data,
		"timestamp":  time.Now().Unix(),
	}

	payload, err := msgpack.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return
	}

	if err := dc.Send(payload); err != nil {
		log.Printf("Error sending to user room: %v", err)
	}
}

// Handle incoming messages from user room
func (ucm *UserChannelManager) handleUserRoomMessage(userID string, data []byte) {
	var req APIRequest
	if err := msgpack.Unmarshal(data, &req); err != nil {
		log.Printf("Error unmarshaling request: %v", err)
		return
	}

	log.Printf("📨 API Request via WebRTC from %s: %s %s", userID, req.Method, req.Endpoint)

	// Execute API request
	go ucm.executeAPIRequest(userID, req)
}

// Execute API request received via WebRTC
func (ucm *UserChannelManager) executeAPIRequest(userID string, req APIRequest) {
	response := APIResponse{
		RequestID: req.RequestID,
		Timestamp: time.Now().Unix(),
	}

	// Route to appropriate handler
	result, statusCode, err := ucm.routeAPIRequest(userID, req)

	response.StatusCode = statusCode
	if err != nil {
		response.Error = err.Error()
	} else {
		response.Data = result
	}

	// Send response
	payload, _ := msgpack.Marshal(response)

	ucm.mu.RLock()
	room, exists := ucm.userRooms[userID]
	ucm.mu.RUnlock()

	if exists && room.IsConnected && room.DataChannel != nil {
		room.DataChannel.Send(payload)
	}
}

// Route API requests to appropriate handlers
func (ucm *UserChannelManager) routeAPIRequest(userID string, req APIRequest) (map[string]interface{}, int, error) {
	switch req.Endpoint {
	// Location APIs
	case "/location/update":
		return ucm.handleLocationUpdate(userID, req.Body)
	case "/location/nearby":
		return ucm.handleLocationNearby(userID, req.Body)
	case "/location/polygon":
		return ucm.handleLocationPolygon(userID, req.Body)

	// Social APIs
	case "/posts":
		if req.Method == "GET" {
			return ucm.handleGetPosts(userID, req.Query)
		} else if req.Method == "POST" {
			return ucm.handleCreatePost(userID, req.Body)
		}
	case "/posts/like":
		return ucm.handleLikePost(userID, req.Body)
	case "/posts/comment":
		return ucm.handleCommentPost(userID, req.Body)

	// Marketplace APIs
	case "/articles/buy":
		return ucm.handleBuyArticle(userID, req.Body)
	case "/articles":
		return ucm.handleGetArticles(userID, req.Query)

	// Presence
	case "/presence/update":
		return ucm.handlePresenceUpdate(userID, req.Body)

	// Rooms
	case "/rooms/join":
		return ucm.handleJoinRoom(userID, req.Body)
	case "/rooms/leave":
		return ucm.handleLeaveRoom(userID, req.Body)

	default:
		return nil, 404, fmt.Errorf("endpoint not found: %s", req.Endpoint)
	}

	return nil, 500, fmt.Errorf("unhandled request")
}

// ==================== API HANDLERS (WebRTC) ====================

func (ucm *UserChannelManager) handleLocationUpdate(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	locationData, ok := body["location"].(map[string]interface{})
	if !ok {
		return nil, 400, fmt.Errorf("invalid location data")
	}

	point, ok := locationData["point"].(map[string]interface{})
	if !ok {
		return nil, 400, fmt.Errorf("invalid point data")
	}

	location := Location{
		Point: Point{
			Lat: point["lat"].(float64),
			Lng: point["lng"].(float64),
		},
		Accuracy:  getFloat64(locationData, "accuracy"),
		Altitude:  getFloat64(locationData, "altitude"),
		Speed:     getFloat64(locationData, "speed"),
		Heading:   getFloat64(locationData, "heading"),
		Timestamp: time.Now(),
	}

	presence := getString(body, "presence", "online")

	if err := locationManager.UpdateLocation(userID, location, presence); err != nil {
		return nil, 500, err
	}

	return map[string]interface{}{
		"success": true,
		"user_id": userID,
	}, 200, nil
}

func (ucm *UserChannelManager) handleLocationNearby(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	point, ok := body["point"].(map[string]interface{})
	if !ok {
		return nil, 400, fmt.Errorf("invalid point")
	}

	radius := getFloat64(body, "radius")

	p := Point{
		Lat: point["lat"].(float64),
		Lng: point["lng"].(float64),
	}

	nearby := locationManager.FindNearby(p, radius, userID)

	users := make([]map[string]interface{}, len(nearby))
	for i, u := range nearby {
		users[i] = map[string]interface{}{
			"user_id":  u.UserID,
			"location": u.Location,
			"presence": u.Presence,
		}
	}

	return map[string]interface{}{
		"count": len(nearby),
		"users": users,
	}, 200, nil
}

func (ucm *UserChannelManager) handleLocationPolygon(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	polygonData, ok := body["polygon"].([]interface{})
	if !ok {
		return nil, 400, fmt.Errorf("invalid polygon")
	}

	polygon := make([]Point, len(polygonData))
	for i, p := range polygonData {
		point := p.(map[string]interface{})
		polygon[i] = Point{
			Lat: point["lat"].(float64),
			Lng: point["lng"].(float64),
		}
	}

	users := locationManager.FindInPolygon(polygon)

	result := make([]map[string]interface{}, len(users))
	for i, u := range users {
		result[i] = map[string]interface{}{
			"user_id":  u.UserID,
			"location": u.Location,
			"presence": u.Presence,
		}
	}

	return map[string]interface{}{
		"count": len(users),
		"users": result,
	}, 200, nil
}

func (ucm *UserChannelManager) handleGetPosts(userID string, query map[string]string) (map[string]interface{}, int, error) {
	filter := query["filter"]
	if filter == "" {
		filter = "isPublic = true"
	}

	posts, err := ucm.app.FindRecordsByFilter("posts", filter, "-created", 20, 0)
	if err != nil {
		return nil, 500, err
	}

	result := make([]map[string]interface{}, len(posts))
	for i, post := range posts {
		result[i] = recordToMap(post)
	}

	return map[string]interface{}{
		"posts": result,
		"count": len(posts),
	}, 200, nil
}

func (ucm *UserChannelManager) handleCreatePost(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	r, err := ucm.app.FindCollectionByNameOrId("posts")
	if err != nil {
		return nil, 500, err
	}
	post := core.NewRecord(r)

	post.Set("user", userID)
	post.Set("type", getString(body, "type", "html"))
	post.Set("content", body["content"])
	post.Set("isPublic", getBool(body, "isPublic", true))
	post.Set("likesCount", 0)
	post.Set("commentsCount", 0)

	if err := ucm.app.Save(post); err != nil {
		return nil, 500, err
	}

	// Broadcast event
	pubsub.Publish("post_events", PubSubMessage{
		Topic: "post_events",
		Payload: map[string]interface{}{
			"type":    "new_post",
			"post_id": post.Id,
			"user_id": userID,
		},
	})

	return map[string]interface{}{
		"success": true,
		"post_id": post.Id,
		"post":    recordToMap(post),
	}, 201, nil
}

func (ucm *UserChannelManager) handleLikePost(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	postID := getString(body, "post_id", "")
	reaction := getString(body, "reaction", "like")

	if postID == "" {
		return nil, 400, fmt.Errorf("post_id required")
	}
	r, err := ucm.app.FindCollectionByNameOrId("likes")
	if err != nil {
		return nil, 500, err
	}

	like := core.NewRecord(r)
	like.Set("user", userID)
	like.Set("post", postID)
	like.Set("reaction", reaction)

	if err := ucm.app.Save(like); err != nil {
		return nil, 500, err
	}

	// Update post likes count
	post, _ := ucm.app.FindRecordById("posts", postID)
	if post != nil {
		count := post.GetInt("likesCount")
		post.Set("likesCount", count+1)
		ucm.app.Save(post)
	}

	return map[string]interface{}{
		"success": true,
		"like_id": like.Id,
	}, 200, nil
}

func (ucm *UserChannelManager) handleCommentPost(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	postID := getString(body, "post_id", "")
	content := getString(body, "content", "")

	if postID == "" || content == "" {
		return nil, 400, fmt.Errorf("post_id and content required")
	}

	r, err := ucm.app.FindCollectionByNameOrId("comments")
	if err != nil {
		return nil, 500, err
	}
	comment := core.NewRecord(r)
	comment.Set("user", userID)
	comment.Set("post", postID)
	comment.Set("content", content)

	if err := ucm.app.Save(comment); err != nil {
		return nil, 500, err
	}

	// Update comments count
	post, _ := ucm.app.FindRecordById("posts", postID)
	if post != nil {
		count := post.GetInt("commentsCount")
		post.Set("commentsCount", count+1)
		ucm.app.Save(post)
	}

	return map[string]interface{}{
		"success":    true,
		"comment_id": comment.Id,
	}, 201, nil
}

func (ucm *UserChannelManager) handleBuyArticle(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	articleID := getString(body, "article_id", "")

	if articleID == "" {
		return nil, 400, fmt.Errorf("article_id required")
	}

	article, err := ucm.app.FindRecordById("articles", articleID)
	if err != nil {
		return nil, 404, fmt.Errorf("article not found")
	}

	quantite := article.GetInt("quantite")
	if quantite <= 0 {
		return nil, 400, fmt.Errorf("out of stock")
	}

	prix := article.GetFloat("prix")

	// Create vente

	r, err := ucm.app.FindCollectionByNameOrId("ventesArticle")
	if err != nil {
		return nil, 500, err
	}

	vente := core.NewRecord(r)
	vente.Set("article", articleID)
	vente.Set("montant", prix)
	vente.Set("status", "encours")
	vente.Set("user", userID)

	if err := ucm.app.Save(vente); err != nil {
		return nil, 500, err
	}

	// Update stock
	article.Set("quantite", quantite-1)
	ucm.app.Save(article)

	return map[string]interface{}{
		"success":  true,
		"vente_id": vente.Id,
		"amount":   prix,
	}, 200, nil
}

func (ucm *UserChannelManager) handleGetArticles(userID string, query map[string]string) (map[string]interface{}, int, error) {
	filter := query["filter"]
	if filter == "" {
		filter = "quantite > 0"
	}

	articles, err := ucm.app.FindRecordsByFilter("articles", filter, "-created", 20, 0)
	if err != nil {
		return nil, 500, err
	}

	result := make([]map[string]interface{}, len(articles))
	for i, article := range articles {
		result[i] = recordToMap(article)
	}

	return map[string]interface{}{
		"articles": result,
		"count":    len(articles),
	}, 200, nil
}

func (ucm *UserChannelManager) handlePresenceUpdate(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	presence := getString(body, "presence", "online")

	user, err := ucm.app.FindRecordById("users", userID)
	if err != nil {
		return nil, 404, err
	}

	user.Set("presence", presence)
	user.Set("lastSeen", time.Now())

	if err := ucm.app.Save(user); err != nil {
		return nil, 500, err
	}

	return map[string]interface{}{
		"success":  true,
		"presence": presence,
	}, 200, nil
}

func (ucm *UserChannelManager) handleJoinRoom(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	roomID := getString(body, "room_id", "")
	if roomID == "" {
		return nil, 400, fmt.Errorf("room_id required")
	}

	// Logic to join room (simplified)
	return map[string]interface{}{
		"success": true,
		"room_id": roomID,
		"message": "Joined room",
	}, 200, nil
}

func (ucm *UserChannelManager) handleLeaveRoom(userID string, body map[string]interface{}) (map[string]interface{}, int, error) {
	roomID := getString(body, "room_id", "")
	if roomID == "" {
		return nil, 400, fmt.Errorf("room_id required")
	}

	return map[string]interface{}{
		"success": true,
		"room_id": roomID,
		"message": "Left room",
	}, 200, nil
}

// ==================== HTTP HANDLERS ====================

// SSE endpoint for user
func handleUserSSE(c *core.RequestEvent) error {
	userID := c.Get("userID").(string)

	c.Response.Header().Set("Content-Type", "text/event-stream")
	c.Response.Header().Set("Cache-Control", "no-cache")
	c.Response.Header().Set("Connection", "keep-alive")
	c.Response.Header().Set("X-Accel-Buffering", "no")

	channel := userChannelManager.GetOrCreateSSEChannel(userID)

	log.Printf("📡 SSE connection established for user: %s", userID)

	// Send initial connection message
	data, _ := json.Marshal(map[string]interface{}{
		"type":    "connected",
		"user_id": userID,
		"message": "SSE channel ready",
	})
	fmt.Fprintf(c.Response, "data: %s\n\n", data)
	if f, ok := c.Response.(http.Flusher); ok {
		f.Flush()
	}

	// Listen for messages
	for msg := range channel.Channel {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		fmt.Fprintf(c.Response, "data: %s\n\n", data)
		if f, ok := c.Response.(http.Flusher); ok {
			f.Flush()
		}
	}

	userChannelManager.CloseSSEChannel(userID)
	return nil
}

// Connect to user's dedicated WebRTC room
func handleConnectUserRoom(c *core.RequestEvent) error {
	userID := c.Get("userID").(string)

	offer, err := userChannelManager.ConnectToUserRoom(userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"user_id": userID,
		"sdp":     offer,
	})
}

// Handle answer from client
func handleUserRoomAnswer(c *core.RequestEvent) error {
	userID := c.Get("userID").(string)

	var answer webrtc.SessionDescription
	if err := c.BindBody(&answer); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid SDP"})
	}

	if err := userChannelManager.HandleUserRoomAnswer(userID, answer); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "connected"})
}

// Utility functions
func getString(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultVal
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0
}

func getBool(m map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultVal
}
