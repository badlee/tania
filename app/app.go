package app

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/ghupdate"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/types"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// ==================== TYPES ====================

type Room struct {
	ID           string
	Type         string // "audio", "video", "data"
	Participants map[string]*Participant
	mu           sync.RWMutex
}

type Participant struct {
	ID          string
	PeerConn    *webrtc.PeerConnection
	DataChannel *webrtc.DataChannel
	UserID      string
}

type DataEvent struct {
	Type      string                 `msgpack:"type"`
	RoomID    string                 `msgpack:"room_id,omitempty"`
	Data      map[string]interface{} `msgpack:"data"`
	Timestamp int64                  `msgpack:"timestamp"`
}

type PubSubMessage struct {
	Topic   string                 `msgpack:"topic"`
	Payload map[string]interface{} `msgpack:"payload"`
}

// ==================== GLOBALS ====================

var (
	rooms              = make(map[string]*Room)
	roomsMutex         sync.RWMutex
	pubsub             = NewPubSub()
	scriptManager      *ScriptManager
	locationManager    *LocationManager
	userChannelManager *UserChannelManager
)

// ==================== PUB/SUB ====================

type PubSub struct {
	subscribers map[string][]chan PubSubMessage
	mu          sync.RWMutex
}

func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[string][]chan PubSubMessage),
	}
}

func (ps *PubSub) Subscribe(topic string) chan PubSubMessage {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ch := make(chan PubSubMessage, 100)
	ps.subscribers[topic] = append(ps.subscribers[topic], ch)
	return ch
}

func (ps *PubSub) Publish(topic string, msg PubSubMessage) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	for _, ch := range ps.subscribers[topic] {
		select {
		case ch <- msg:
		default:
		}
	}
}

// ==================== WEBRTC CONFIG ====================

func createPeerConnection() (*webrtc.PeerConnection, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	return webrtc.NewPeerConnection(config)
}

// ==================== ROOM MANAGEMENT ====================

func getOrCreateRoom(roomID, roomType string) *Room {
	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	if room, exists := rooms[roomID]; exists {
		return room
	}

	room := &Room{
		ID:           roomID,
		Type:         roomType,
		Participants: make(map[string]*Participant),
	}
	rooms[roomID] = room
	return room
}

func (r *Room) AddParticipant(p *Participant) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Participants[p.ID] = p

	// Broadcast join event
	r.broadcastEvent("participant_joined", map[string]interface{}{
		"participant_id": p.ID,
		"user_id":        p.UserID,
	})
}

func (r *Room) RemoveParticipant(participantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Participants, participantID)

	r.broadcastEvent("participant_left", map[string]interface{}{
		"participant_id": participantID,
	})
}

func (r *Room) broadcastEvent(eventType string, data map[string]interface{}) {
	event := DataEvent{
		Type:      eventType,
		RoomID:    r.ID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	payload, _ := msgpack.Marshal(event)

	for _, p := range r.Participants {
		if p.DataChannel != nil && p.DataChannel.ReadyState() == webrtc.DataChannelStateOpen {
			p.DataChannel.Send(payload)
		}
	}
}

// ==================== POCKETBASE MIGRATIONS ====================

func SetupCollections(app core.App) error {
	return app.RunInTransaction(func(txApp core.App) error {
		// Posts Collection
		posts := core.NewBaseCollection("posts")
		posts.Fields.Add(
			&core.RelationField{Name: "user", Required: true, CollectionId: "_pb_users_auth_", MaxSelect: 1},
			&core.SelectField{Name: "categories", MaxSelect: 5, Values: []string{"tech", "fashion", "food", "travel", "other"}},
			&core.BoolField{Name: "isPublic", Required: true},
			&core.SelectField{Name: "type", Required: true, MaxSelect: 1, Values: []string{"html", "reel", "images", "url"}},
			&core.TextField{Name: "content"},
			&core.FileField{Name: "images", MaxSelect: 3, MaxSize: 5242880, MimeTypes: []string{"image/jpeg", "image/png", "image/webp"}},
			&core.FileField{Name: "video", MaxSelect: 1, MaxSize: 52428800, MimeTypes: []string{"video/mp4"}},
			&core.RelationField{Name: "article", CollectionId: "articles", MaxSelect: 1},
			&core.SelectField{Name: "action", MaxSelect: 1, Values: []string{"none", "buy", "join", "subscribe", "read", "listen"}},
			&core.TextField{Name: "actionText"},
			&core.JSONField{Name: "dataAction"},
			&core.NumberField{Name: "likesCount", Min: types.Pointer(0.0)},
			&core.NumberField{Name: "commentsCount", Min: types.Pointer(0.0)},
		)
		if err := txApp.Save(posts); err != nil {
			return err
		}

		// Articles Collection
		articles := core.NewBaseCollection("articles")
		articles.Fields.Add(
			&core.TextField{Name: "title", Required: true},
			&core.TextField{Name: "desc"},
			&core.NumberField{Name: "prixOriginal", Min: types.Pointer(0.0)},
			&core.NumberField{Name: "prix", Required: true, Min: types.Pointer(0.0)},
			&core.NumberField{Name: "quantite", Required: true, Min: types.Pointer(0.0)},
			&core.DateField{Name: "dueDate"},
			&core.FileField{Name: "images", MaxSelect: 3, MaxSize: 5242880, MimeTypes: []string{"image/jpeg", "image/png", "image/webp"}},
			&core.RelationField{Name: "user", Required: true, CollectionId: "_pb_users_auth_", MaxSelect: 1},
		)
		if err := txApp.Save(articles); err != nil {
			return err
		}

		// VentesArticle Collection
		ventes := core.NewBaseCollection("ventesArticle")
		ventes.Fields.Add(
			&core.RelationField{Name: "article", Required: true, CollectionId: "articles", MaxSelect: 1},
			&core.NumberField{Name: "montant", Required: true, Min: types.Pointer(0.0)},
			&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"paye", "encours", "echec", "annule"}},
			&core.RelationField{Name: "user", Required: true, CollectionId: "_pb_users_auth_", MaxSelect: 1},
			&core.DateField{Name: "paiementDate"},
			&core.DateField{Name: "cancelDate"},
			&core.DateField{Name: "failDate"},
			&core.RelationField{Name: "fromPost", CollectionId: "posts", MaxSelect: 1},
		)
		if err := txApp.Save(ventes); err != nil {
			return err
		}

		// Operations Collection
		operations := core.NewBaseCollection("operations")
		operations.Fields.Add(
			&core.RelationField{Name: "user", Required: true, CollectionId: "_pb_users_auth_", MaxSelect: 1},
			&core.RelationField{Name: "vente", CollectionId: "ventesArticle", MaxSelect: 1},
			&core.NumberField{Name: "montant", Required: true},
			&core.SelectField{Name: "operation", Required: true, MaxSelect: 1, Values: []string{"cashin", "cashout"}},
			&core.TextField{Name: "desc"},
			&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"paye", "en_attente", "encours", "echec", "annule"}},
		)
		if err := txApp.Save(operations); err != nil {
			return err
		}

		// Likes Collection
		likes := core.NewBaseCollection("likes")
		likes.Fields.Add(
			&core.RelationField{Name: "user", Required: true, CollectionId: "_pb_users_auth_", MaxSelect: 1},
			&core.RelationField{Name: "post", Required: true, CollectionId: "posts", MaxSelect: 1},
			&core.SelectField{Name: "reaction", MaxSelect: 1, Values: []string{"like", "love", "fire", "wow", "sad", "angry"}},
		)
		likes.Indexes = []string{"CREATE UNIQUE INDEX idx_user_post ON likes (user, post)"}
		if err := txApp.Save(likes); err != nil {
			return err
		}

		// Comments Collection
		comments := core.NewBaseCollection("comments")
		comments.Fields.Add(
			&core.RelationField{Name: "user", Required: true, CollectionId: "_pb_users_auth_", MaxSelect: 1},
			&core.RelationField{Name: "post", Required: true, CollectionId: "posts", MaxSelect: 1},
			&core.TextField{Name: "content", Required: true},
			&core.RelationField{Name: "parentComment", CollectionId: "comments", MaxSelect: 1},
		)
		if err := txApp.Save(comments); err != nil {
			return err
		}

		// Rooms Collection (pour persistance)
		roomsColl := core.NewBaseCollection("rooms")
		roomsColl.Fields.Add(
			&core.SelectField{Name: "roomType", Required: true, MaxSelect: 1, Values: []string{"audio", "video", "data"}},
			&core.TextField{Name: "name", Required: true},
			&core.RelationField{Name: "creator", Required: true, CollectionId: "_pb_users_auth_", MaxSelect: 1},
			&core.BoolField{Name: "isPublic"},
			&core.NumberField{Name: "maxParticipants", Min: types.Pointer(2.0)},
		)
		return txApp.Save(roomsColl)
	})
}

// ==================== HTTP HANDLERS ====================

func handleCreateRoom(c *core.RequestEvent) error {
	type CreateRoomRequest struct {
		RoomType string `json:"room_type"`
		Name     string `json:"name"`
	}

	var req CreateRoomRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Check if respond_to=sse parameter is present
	respondTo := c.Request.URL.Query().Get("respond_to")

	roomID := generateID()
	room := getOrCreateRoom(roomID, req.RoomType)

	result := map[string]interface{}{
		"room_id":   room.ID,
		"room_type": room.Type,
		"name":      req.Name,
	}

	// If respond_to=sse, send response via SSE
	if respondTo == "sse" {
		userID := c.Get("userID").(string)
		requestID := c.Request.Header.Get("X-Request-ID")
		userChannelManager.SendToSSE(userID, "room_created", result, requestID)
		return c.JSON(202, map[string]string{"status": "response_sent_via_sse"})
	}

	return c.JSON(200, result)
}

func handleJoinRoom(c *core.RequestEvent) error {
	roomID := c.Request.PathValue("roomId")

	room := getOrCreateRoom(roomID, "audio")

	pc, err := createPeerConnection()
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	participantID := generateID()
	participant := &Participant{
		ID:       participantID,
		PeerConn: pc,
		UserID:   c.Get("userID").(string),
	}

	// Setup DataChannel
	dc, err := pc.CreateDataChannel("events", nil)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	participant.DataChannel = dc

	dc.OnOpen(func() {
		log.Printf("DataChannel opened for participant %s", participantID)
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var event DataEvent
		if err := msgpack.Unmarshal(msg.Data, &event); err == nil {
			handleDataEvent(room, participant, event)
		}
	})

	// Handle tracks
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Track received: %s", track.Kind())
		// Broadcast to other participants
		room.mu.RLock()
		defer room.mu.RUnlock()
		// Créer un TrackLocal pour relayer le track
		localTrack, err := webrtc.NewTrackLocalStaticRTP(
			track.Codec().RTPCodecCapability,
			track.ID(),
			track.StreamID(),
		)
		if err != nil {
			log.Printf("Error creating local track: %v", err)
			return
		}
		for _, p := range room.Participants {
			if p.ID != participantID {
				if sender, err := p.PeerConn.AddTrack(localTrack); err == nil {
					go func() {
						buf := make([]byte, 1500)
						for {
							if _, _, err := track.Read(buf); err != nil {
								return
							}
						}
					}()
					log.Printf("Track forwarded to %s", p.ID)
					_ = sender
				}
			}
		}
	})

	room.AddParticipant(participant)

	// Create offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"participant_id": participantID,
		"sdp":            offer,
	})
}

func handleAnswer(c *core.RequestEvent) error {
	roomID := c.Request.PathValue("roomId")
	participantID := c.Request.PathValue("participantId")

	var answer webrtc.SessionDescription
	if err := c.BindBody(&answer); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid SDP"})
	}

	roomsMutex.RLock()
	room, exists := rooms[roomID]
	roomsMutex.RUnlock()

	if !exists {
		return c.JSON(404, map[string]string{"error": "room not found"})
	}

	room.mu.RLock()
	participant, exists := room.Participants[participantID]
	room.mu.RUnlock()

	if !exists {
		return c.JSON(404, map[string]string{"error": "participant not found"})
	}

	if err := participant.PeerConn.SetRemoteDescription(answer); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"status": "ok"})
}

func handleDataEvent(room *Room, sender *Participant, event DataEvent) {
	switch event.Type {
	case "chat":
		room.broadcastEvent("chat", map[string]interface{}{
			"from":    sender.UserID,
			"message": event.Data["message"],
		})
	case "reaction":
		pubsub.Publish("reactions", PubSubMessage{
			Topic: "reactions",
			Payload: map[string]interface{}{
				"room_id": room.ID,
				"user_id": sender.UserID,
				"type":    event.Data["type"],
			},
		})
	}
}

// ==================== SOCIAL HANDLERS ====================

func handleLikePost(c *core.RequestEvent) error {
	app := c.App
	postID := c.Request.PathValue("postId")
	userID := c.Get("userID").(string)
	reaction := c.Request.URL.Query().Get("reaction")

	// Create like
	r, err := app.FindCollectionByNameOrId("likes")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	like := core.NewRecord(r)
	like.Set("user", userID)
	like.Set("post", postID)
	like.Set("reaction", reaction)

	if err := app.Save(like); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	// Update post likes count
	post, err := app.FindRecordById("posts", postID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "post not found"})
	}

	likesCount := post.GetInt("likesCount")
	post.Set("likesCount", likesCount+1)
	app.Save(post)

	// Broadcast event
	pubsub.Publish("post_events", PubSubMessage{
		Topic: "post_events",
		Payload: map[string]interface{}{
			"type":     "like",
			"post_id":  postID,
			"user_id":  userID,
			"reaction": reaction,
		},
	})

	return c.JSON(200, map[string]interface{}{"success": true, "like_id": like.Id})
}

func handleCommentPost(c *core.RequestEvent) error {
	app := c.App
	postID := c.Request.PathValue("postId")
	userID := c.Get("userID").(string)

	type CommentRequest struct {
		Content       string `json:"content"`
		ParentComment string `json:"parent_comment"`
	}

	var req CommentRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	r, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	comment := core.NewRecord(r)
	comment.Set("user", userID)
	comment.Set("post", postID)
	comment.Set("content", req.Content)
	if req.ParentComment != "" {
		comment.Set("parentComment", req.ParentComment)
	}

	if err := app.Save(comment); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	// Update comments count
	post, _ := app.FindRecordById("posts", postID)
	if post != nil {
		commentsCount := post.GetInt("commentsCount")
		post.Set("commentsCount", commentsCount+1)
		app.Save(post)
	}

	// Broadcast
	pubsub.Publish("post_events", PubSubMessage{
		Topic: "post_events",
		Payload: map[string]interface{}{
			"type":    "comment",
			"post_id": postID,
			"user_id": userID,
			"comment": req.Content,
		},
	})

	return c.JSON(200, map[string]interface{}{"success": true, "comment_id": comment.Id})
}

func handleBuyArticle(c *core.RequestEvent) error {
	app := c.App
	articleID := c.Request.PathValue("articleId")
	userID := c.Get("userID").(string)

	article, err := app.FindRecordById("articles", articleID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "article not found"})
	}

	quantite := article.GetInt("quantite")
	if quantite <= 0 {
		return c.JSON(400, map[string]string{"error": "out of stock"})
	}

	prix := article.GetFloat("prix")

	// Create vente
	r, err := app.FindCollectionByNameOrId("ventesArticle")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	vente := core.NewRecord(r)
	vente.Set("article", articleID)
	vente.Set("montant", prix)
	vente.Set("status", "encours")
	vente.Set("user", userID)

	if err := app.Save(vente); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	// Decrement stock
	article.Set("quantite", quantite-1)
	app.Save(article)

	// Create operation
	r, err = app.FindCollectionByNameOrId("operations")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	operation := core.NewRecord(r)
	operation.Set("user", userID)
	operation.Set("vente", vente.Id)
	operation.Set("montant", -prix)
	operation.Set("operation", "cashout")
	operation.Set("desc", "Achat article: "+article.GetString("title"))
	operation.Set("status", "encours")
	app.Save(operation)

	// Broadcast
	pubsub.Publish("sales", PubSubMessage{
		Topic: "sales",
		Payload: map[string]interface{}{
			"type":       "purchase",
			"article_id": articleID,
			"user_id":    userID,
			"amount":     prix,
		},
	})

	return c.JSON(200, map[string]interface{}{
		"success":  true,
		"vente_id": vente.Id,
	})
}

// ==================== UTILITIES ====================

var idCounter uint64

func generateID() string {
	idCounter++
	return time.Now().Format("20060102150405") + "-" + string(rune(idCounter))
}

// ==================== MAIN ====================

func Execute() {
	app := pocketbase.New()
	var migrationsDir string
	app.RootCmd.PersistentFlags().StringVar(
		&migrationsDir,
		"migrationsDir",
		"",
		"the directory with the user defined migrations",
	)

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(
		&automigrate,
		"automigrate",
		true,
		"enable/disable auto migrations",
	)

	var publicDir string
	app.RootCmd.PersistentFlags().StringVar(
		&publicDir,
		"publicDir",
		defaultPublicDir(),
		"the directory to serve static files",
	)

	var indexFallback bool
	app.RootCmd.PersistentFlags().BoolVar(
		&indexFallback,
		"indexFallback",
		true,
		"fallback the request to index.html on missing static path, e.g. when pretty urls are used with SPA",
	)

	// set commandes

	// migrate command (with js templates)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  automigrate,
		Dir:          migrationsDir,
	})

	var ghOwner string
	app.RootCmd.PersistentFlags().StringVar(
		&ghOwner,
		"github-owner",
		"badlee",
		"the owner of the repository for the selfupdate",
	)

	var ghRepo string
	app.RootCmd.PersistentFlags().StringVar(
		&ghRepo,
		"github-repo",
		"tania",
		"the name of the repository for the selfupdate",
	)

	var ghArchiveExecutable string
	app.RootCmd.PersistentFlags().StringVar(
		&ghArchiveExecutable,
		"github-archive-executable",
		"tania",
		"the name of the executable file in the release archive",
	)

	// GitHub selfupdate
	ghupdate.MustRegister(app, app.RootCmd, ghupdate.Config{
		Owner:             ghOwner,
		Repo:              ghRepo,
		ArchiveExecutable: ghArchiveExecutable,
	})
	// static route to serves files from the provided public dir
	// (if publicDir exists and the route path is not already defined)
	publicFS := os.DirFS(publicDir)
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				e.Router.GET("/{path...}", apis.Static(publicFS, indexFallback))
			}

			return e.Next()
		},
		Priority: 0x7FFFFFFF, // execute as latest as possible to allow users to provide their own route
	})

	// Setup collections on serve
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Setup collections
		if err := SetupCollections(app); err != nil {
			log.Println("Collections setup:", err)
		}

		// Setup scripts collection
		if err := SetupScriptsCollection(app); err != nil {
			log.Println("Scripts collection setup:", err)
		}

		// Setup location collections
		if err := SetupLocationCollections(app); err != nil {
			log.Println("Location collections setup:", err)
		}

		// Setup follow and room collections
		if err := SetupFollowAndRoomCollections(app); err != nil {
			log.Println("Follow/Room collections setup:", err)
		}

		// Initialiser le Location Manager
		locationManager = NewLocationManager(app)

		// Initialiser le User Channel Manager
		userChannelManager = NewUserChannelManager(app)

		// Initialiser le Script Manager
		var err error
		scriptManager, err = NewScriptManager(app, "./pb_hooks", "./pb_modules")
		if err != nil {
			log.Fatal("Failed to create script manager:", err)
		}

		// Charger tous les scripts
		if err := scriptManager.LoadAll(); err != nil {
			log.Println("Error loading scripts:", err)
		}

		// WebRTC Routes
		e.Router.POST("/api/rooms", func(c *core.RequestEvent) error {
			return handleCreateRoom(c)
		})

		e.Router.POST("/api/rooms/{roomId}/join", func(c *core.RequestEvent) error {
			return handleJoinRoom(c)
		})

		e.Router.POST("/api/rooms/{roomId}/participants/{participantId}/answer", func(c *core.RequestEvent) error {
			return handleAnswer(c)
		})

		// Social Routes (with auth middleware)
		e.Router.POST("/api/posts/{postId}/like", func(c *core.RequestEvent) error {
			return handleLikePost(c)
		}).Bind(apis.RequireAuth())

		e.Router.POST("/api/posts/{postId}/comment", func(c *core.RequestEvent) error {
			return handleCommentPost(c)
		}).Bind(apis.RequireAuth())

		e.Router.POST("/api/articles/{articleId}/buy", func(c *core.RequestEvent) error {
			return handleBuyArticle(c)
		}).Bind(apis.RequireAuth())

		// ==================== LOCATION & GEOFENCE ROUTES ====================

		// Update location
		e.Router.POST("/api/location/update", func(c *core.RequestEvent) error {
			return handleUpdateLocation(c)
		}).Bind(apis.RequireAuth())

		// Get user location
		e.Router.GET("/api/location/user/{userId}", func(c *core.RequestEvent) error {
			return handleGetLocation(c)
		}).Bind(apis.RequireAuth())

		// Find nearby users
		e.Router.POST("/api/location/nearby", func(c *core.RequestEvent) error {
			return handleFindNearby(c)
		}).Bind(apis.RequireAuth())

		// Find users in polygon
		e.Router.POST("/api/location/polygon", func(c *core.RequestEvent) error {
			return handleFindInPolygon(c)
		}).Bind(apis.RequireAuth())

		// Get users by presence
		e.Router.GET("/api/location/presence", func(c *core.RequestEvent) error {
			return handleGetByPresence(c)
		}).Bind(apis.RequireAuth())

		// Create geofence
		e.Router.POST("/api/geofences", func(c *core.RequestEvent) error {
			return handleCreateGeoFence(c)
		}).Bind(apis.RequireAuth())

		// Update geofence
		e.Router.PATCH("/api/geofences/{fenceId}", func(c *core.RequestEvent) error {
			return handleUpdateGeoFence(c)
		}).Bind(apis.RequireAuth())

		// Delete geofence
		e.Router.DELETE("/api/geofences/{fenceId}", func(c *core.RequestEvent) error {
			return handleDeleteGeoFence(c)
		}).Bind(apis.RequireAuth())

		// List geofences
		e.Router.GET("/api/geofences", func(c *core.RequestEvent) error {
			return handleListGeoFences(c)
		}).Bind(apis.RequireAuth())

		// Notify users in zone
		e.Router.POST("/api/location/notify-zone", func(c *core.RequestEvent) error {
			return handleNotifyInZone(c)
		}).Bind(apis.RequireAuth())

		// WebRTC: Broadcast location to room
		e.Router.POST("/api/rooms/{roomId}/broadcast-location", func(c *core.RequestEvent) error {
			return handleBroadcastLocationToRoom(c)
		}).Bind(apis.RequireAuth())

		// ==================== USER DEDICATED CHANNELS ====================

		// SSE channel for user
		e.Router.GET("/api/user/sse", func(c *core.RequestEvent) error {
			return handleUserSSE(c)
		}).Bind(apis.RequireAuth())

		// Connect to user's dedicated WebRTC room
		e.Router.POST("/api/user/room/connect", func(c *core.RequestEvent) error {
			return handleConnectUserRoom(c)
		}).Bind(apis.RequireAuth())

		// Answer for user room
		e.Router.POST("/api/user/room/answer", func(c *core.RequestEvent) error {
			return handleUserRoomAnswer(c)
		}).Bind(apis.RequireAuth())

		// ==================== FOLLOW/FOLLOWER ROUTES ====================

		// Get follow settings
		e.Router.GET("/api/users/{userId}/follow-settings", func(c *core.RequestEvent) error {
			return handleGetFollowSettings(c)
		})

		// Update follow settings
		e.Router.PUT("/api/user/follow-settings", func(c *core.RequestEvent) error {
			return handleUpdateFollowSettings(c)
		}).Bind(apis.RequireAuth())

		// Follow user
		e.Router.POST("/api/users/{userId}/follow", func(c *core.RequestEvent) error {
			return handleFollowUser(c)
		}).Bind(apis.RequireAuth())

		// Unfollow user
		e.Router.DELETE("/api/users/{userId}/follow", func(c *core.RequestEvent) error {
			return handleUnfollow(c)
		}).Bind(apis.RequireAuth())

		// Approve follow request
		e.Router.POST("/api/follows/{followId}/approve", func(c *core.RequestEvent) error {
			return handleApproveFollow(c)
		}).Bind(apis.RequireAuth())

		// Reject follow request
		e.Router.POST("/api/follows/{followId}/reject", func(c *core.RequestEvent) error {
			return handleRejectFollow(c)
		}).Bind(apis.RequireAuth())

		// Get followers
		e.Router.GET("/api/users/{userId}/followers", func(c *core.RequestEvent) error {
			return handleGetFollowers(c)
		})

		// Get following
		e.Router.GET("/api/users/{userId}/following", func(c *core.RequestEvent) error {
			return handleGetFollowing(c)
		})

		// Promote follower to admin
		e.Router.POST("/api/follows/{followId}/promote", func(c *core.RequestEvent) error {
			return handlePromoteFollowerToAdmin(c)
		}).Bind(apis.RequireAuth())

		// ==================== ROOM MANAGEMENT ROUTES ====================

		// Create room with settings
		e.Router.POST("/api/rooms/create", func(c *core.RequestEvent) error {
			return handleCreateRoomWithSettings(c)
		}).Bind(apis.RequireAuth())

		// Join room
		e.Router.POST("/api/rooms/{roomId}/join-request", func(c *core.RequestEvent) error {
			return handleJoinRoomRequest(c)
		}).Bind(apis.RequireAuth())

		// Approve member
		e.Router.POST("/api/room-members/{memberId}/approve", func(c *core.RequestEvent) error {
			return handleApproveRoomMember(c)
		}).Bind(apis.RequireAuth())

		// Reject member
		e.Router.POST("/api/room-members/{memberId}/reject", func(c *core.RequestEvent) error {
			return handleRejectRoomMember(c)
		}).Bind(apis.RequireAuth())

		// Promote to admin
		e.Router.POST("/api/room-members/{memberId}/promote", func(c *core.RequestEvent) error {
			return handlePromoteToRoomAdmin(c)
		}).Bind(apis.RequireAuth())

		// Demote admin
		e.Router.POST("/api/room-members/{memberId}/demote", func(c *core.RequestEvent) error {
			return handleDemoteRoomAdmin(c)
		}).Bind(apis.RequireAuth())

		// Ban member
		e.Router.POST("/api/room-members/{memberId}/ban", func(c *core.RequestEvent) error {
			return handleBanRoomMember(c)
		}).Bind(apis.RequireAuth())

		// Leave room
		e.Router.POST("/api/rooms/{roomId}/leave", func(c *core.RequestEvent) error {
			return handleLeaveRoom(c)
		}).Bind(apis.RequireAuth())

		// Get room members
		e.Router.GET("/api/rooms/{roomId}/members", func(c *core.RequestEvent) error {
			return handleGetRoomMembers(c)
		})

		// Get my rooms
		e.Router.GET("/api/user/rooms", func(c *core.RequestEvent) error {
			return handleGetMyRooms(c)
		}).Bind(apis.RequireAuth())

		// Transfer ownership
		e.Router.POST("/api/rooms/{roomId}/transfer-ownership", func(c *core.RequestEvent) error {
			return handleTransferRoomOwnership(c)
		}).Bind(apis.RequireAuth())

		// Update room settings
		e.Router.PATCH("/api/rooms/{roomId}/settings", func(c *core.RequestEvent) error {
			return handleUpdateRoomSettings(c)
		}).Bind(apis.RequireAuth())

		// Script Management Routes (optional, pour debug)
		e.Router.GET("/api/scripts/status", func(c *core.RequestEvent) error {
			if scriptManager == nil {
				return c.JSON(200, map[string]interface{}{"scripts": 0})
			}

			scriptManager.RLock()
			defer scriptManager.RUnlock()

			scripts := []map[string]interface{}{}
			for _, ctx := range scriptManager.contexts {
				ctx.mu.RLock()
				scripts = append(scripts, map[string]interface{}{
					"id":         ctx.ID,
					"name":       ctx.Name,
					"path":       ctx.FilePath,
					"is_running": ctx.isRunning,
				})
				ctx.mu.RUnlock()
			}

			return c.JSON(200, map[string]interface{}{
				"scripts": scripts,
				"count":   len(scripts),
			})
		})

		e.Router.POST("/api/scripts/{scriptId}/reload", func(c *core.RequestEvent) error {
			scriptId := c.Request.PathValue("scriptId")
			if err := scriptManager.ReloadScript(scriptId); err != nil {
				return c.JSON(400, map[string]string{"error": err.Error()})
			}
			return c.JSON(200, map[string]string{"status": "reloaded"})
		}).Bind(apis.RequireSuperuserAuth())

		// SSE for real-time events
		e.Router.GET("/api/events/{topic}", func(c *core.RequestEvent) error {
			topic := c.Request.PathValue("topic")

			c.Response.Header().Set("Content-Type", "text/event-stream")
			c.Response.Header().Set("Cache-Control", "no-cache")
			c.Response.Header().Set("Connection", "keep-alive")

			ch := pubsub.Subscribe(topic)
			defer close(ch)

			for msg := range ch {
				data, _ := json.Marshal(msg.Payload)
				c.Response.Write([]byte("data: " + string(data) + "\n\n"))
				if f, ok := c.Response.(http.Flusher); ok {
					f.Flush()
				}
				// c.Response.Flush()
			}

			return nil
		})

		// OpenAPI/Swagger endpoint
		e.Router.GET("/api/openapi", func(c *core.RequestEvent) error {
			return c.JSON(200, getOpenAPISpec())
		})

		return e.Next()
	})

	// Periodic task: check expired subscriptions (every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			checkExpiredSubscriptions(app)
		}
	}()

	// Record hooks for real-time events
	app.OnRecordAfterCreateSuccess("posts").BindFunc(func(e *core.RecordEvent) error {
		payload := map[string]interface{}{
			"type":    "new_post",
			"post_id": e.Record.Id,
			"user_id": e.Record.GetString("user"),
		}

		pubsub.Publish("post_events", PubSubMessage{
			Topic:   "post_events",
			Payload: payload,
		})

		// Also send to user's SSE channel
		userID := e.Record.GetString("user")
		userChannelManager.SendToSSE(userID, "post_created", payload, "")

		return nil
	})

	app.OnRecordAfterCreateSuccess("likes").BindFunc(func(e *core.RecordEvent) error {
		payload := map[string]interface{}{
			"type":     "like",
			"like_id":  e.Record.Id,
			"post_id":  e.Record.GetString("post"),
			"user_id":  e.Record.GetString("user"),
			"reaction": e.Record.GetString("reaction"),
		}

		pubsub.Publish("post_events", PubSubMessage{
			Topic:   "post_events",
			Payload: payload,
		})

		// Send notification to post owner
		post, _ := app.FindRecordById("posts", e.Record.GetString("post"))
		if post != nil {
			postOwner := post.GetString("user")
			userChannelManager.SendToSSE(postOwner, "post_liked", payload, "")
			userChannelManager.SendToUserRoom(postOwner, "notification", payload, "")
		}

		return nil
	})

	app.OnRecordAfterCreateSuccess("comments").BindFunc(func(e *core.RecordEvent) error {
		payload := map[string]interface{}{
			"type":       "comment",
			"comment_id": e.Record.Id,
			"post_id":    e.Record.GetString("post"),
			"user_id":    e.Record.GetString("user"),
			"content":    e.Record.GetString("content"),
		}

		pubsub.Publish("post_events", PubSubMessage{
			Topic:   "post_events",
			Payload: payload,
		})

		// Send notification to post owner
		post, _ := app.FindRecordById("posts", e.Record.GetString("post"))
		if post != nil {
			postOwner := post.GetString("user")
			userChannelManager.SendToSSE(postOwner, "post_commented", payload, "")
			userChannelManager.SendToUserRoom(postOwner, "notification", payload, "")
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// ==================== OPENAPI SPEC ====================

func getOpenAPISpec() map[string]interface{} {
	return map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "WebRTC Social Marketplace API",
			"version": "1.0.0",
		},
		"paths": map[string]interface{}{
			"/api/rooms": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Create a new room",
					"requestBody": map[string]interface{}{
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"room_type": map[string]string{"type": "string", "enum": "audio,video,data"},
										"name":      map[string]string{"type": "string"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Room created",
						},
					},
				},
			},
			"/api/posts/{postId}/like": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":  "Like a post",
					"security": []map[string][]string{{"bearerAuth": {}}},
					"parameters": []map[string]interface{}{
						{"name": "postId", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Like added"},
					},
				},
			},
			"/api/posts/{postId}/comment": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":  "Comment on a post",
					"security": []map[string][]string{{"bearerAuth": {}}},
					"parameters": []map[string]interface{}{
						{"name": "postId", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"requestBody": map[string]interface{}{
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"content":        map[string]string{"type": "string"},
										"parent_comment": map[string]string{"type": "string"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Comment added"},
					},
				},
			},
			"/api/articles/{articleId}/buy": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":  "Buy an article",
					"security": []map[string][]string{{"bearerAuth": {}}},
					"parameters": []map[string]interface{}{
						{"name": "articleId", "in": "path", "required": true, "schema": map[string]string{"type": "string"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Purchase initiated"},
					},
				},
			},
			"/api/events/{topic}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Subscribe to real-time events via SSE",
					"parameters": []map[string]interface{}{
						{"name": "topic", "in": "path", "required": true, "schema": map[string]string{"type": "string", "enum": "post_events,sales,reactions"}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Event stream",
							"content": map[string]interface{}{
								"text/event-stream": map[string]interface{}{},
							},
						},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":   "http",
					"scheme": "bearer",
				},
			},
		},
	}
}
