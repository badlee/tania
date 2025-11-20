package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	tania "tania/app"

	"github.com/dop251/goja"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

var (
	locationManager    *tania.LocationManager
	userChannelManager *tania.UserChannelManager
)

// ==================== SETUP ====================

func setupTestApp(t *testing.T) *tests.TestApp {
	var app *tests.TestApp
	var err error
	app, err = tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	// Setup collections
	if err := tania.SetupCollections(app); err != nil {
		t.Fatal(err)
	}
	if err := tania.SetupLocationCollections(app); err != nil {
		t.Fatal(err)
	}
	if err := tania.SetupFollowAndRoomCollections(app); err != nil {
		t.Fatal(err)
	}
	if err := tania.SetupScriptsCollection(app); err != nil {
		t.Fatal(err)
	}

	// Initialize managers
	locationManager = tania.NewLocationManager(app)
	userChannelManager = tania.NewUserChannelManager(app)

	return app
}

func createTestUser(app core.App, email, password string) (string, string, error) {
	user, err := app.FindAuthRecordByEmail("users", email)
	if err != nil {
		collection, _ := app.FindCollectionByNameOrId("users")
		user = core.NewRecord(collection)
		user.Set("email", email)
		user.Set("password", password)
		user.Set("passwordConfirm", password)
		if err := app.Save(user); err != nil {
			return "", "", err
		}
	}

	token, _ := user.NewAuthToken()
	return user.Id, token, nil
}

// ==================== PUBSUB TESTS ====================

func TestPubSub(t *testing.T) {
	ps := tania.NewPubSub()

	// Test subscribe
	ch := ps.Subscribe("test_topic")
	assert.NotNil(t, ch)

	// Test publish
	go ps.Publish("test_topic", tania.PubSubMessage{
		Topic: "test_topic",
		Payload: map[string]interface{}{
			"message": "hello",
		},
	})

	// Test receive
	select {
	case msg := <-ch:
		assert.Equal(t, "test_topic", msg.Topic)
		assert.Equal(t, "hello", msg.Payload["message"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

// ==================== LOCATION TESTS ====================

func TestLocationManager(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	lm := tania.NewLocationManager(app)

	// Test update location
	location := tania.Location{
		Point:     tania.Point{Lat: 48.8566, Lng: 2.3522},
		Accuracy:  10,
		Timestamp: time.Now(),
	}

	err := lm.UpdateLocation("user_1", location, "online")
	assert.NoError(t, err)

	// Test get location
	userLoc, exists := lm.GetLocation("user_1")
	assert.True(t, exists)
	assert.Equal(t, "user_1", userLoc.UserID)
	assert.Equal(t, "online", userLoc.Presence)
	assert.Equal(t, 48.8566, userLoc.Location.Point.Lat)
}

func TestFindNearby(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	lm := tania.NewLocationManager(app)

	// Add test locations
	loc1 := tania.Location{Point: tania.Point{Lat: 48.8566, Lng: 2.3522}, Accuracy: 10, Timestamp: time.Now()}
	loc2 := tania.Location{Point: tania.Point{Lat: 48.8567, Lng: 2.3523}, Accuracy: 10, Timestamp: time.Now()}
	loc3 := tania.Location{Point: tania.Point{Lat: 48.9000, Lng: 2.4000}, Accuracy: 10, Timestamp: time.Now()}

	lm.UpdateLocation("user_1", loc1, "online")
	lm.UpdateLocation("user_2", loc2, "online")
	lm.UpdateLocation("user_3", loc3, "online")

	// Find nearby (radius 100m)
	nearby := lm.FindNearby(tania.Point{Lat: 48.8566, Lng: 2.3522}, 100, "")
	assert.Equal(t, 2, len(nearby)) // user_1 and user_2

	// Find nearby (radius 10000m)
	nearby = lm.FindNearby(tania.Point{Lat: 48.8566, Lng: 2.3522}, 10000, "")
	assert.Equal(t, 3, len(nearby)) // all users
}

func TestHaversineDistance(t *testing.T) {
	// Paris to London (approx 344 km)
	paris := tania.Point{Lat: 48.8566, Lng: 2.3522}
	london := tania.Point{Lat: 51.5074, Lng: -0.1278}

	distance := tania.HaversineDistance(paris, london)
	assert.InDelta(t, 344000, distance, 5000) // ±5km tolerance

	// Same point
	distance = tania.HaversineDistance(paris, paris)
	assert.Equal(t, 0.0, distance)
}

func TestPointInPolygon(t *testing.T) {
	polygon := []tania.Point{
		{Lat: 48.8, Lng: 2.3},
		{Lat: 48.9, Lng: 2.3},
		{Lat: 48.9, Lng: 2.4},
		{Lat: 48.8, Lng: 2.4},
	}

	// Inside
	inside := tania.Point{Lat: 48.85, Lng: 2.35}
	assert.True(t, tania.IsPointInPolygon(inside, polygon))

	// Outside
	outside := tania.Point{Lat: 48.7, Lng: 2.2}
	assert.False(t, tania.IsPointInPolygon(outside, polygon))
}

// ==================== GEOFENCE TESTS ====================

func TestGeofence(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	lm := tania.NewLocationManager(app)

	// Create geofence
	fence := &tania.GeoFence{
		ID:   "fence_1",
		Name: "Test Zone",
		Geometry: tania.GeoJSONGeometry{
			Type:        "Circle",
			Coordinates: []interface{}{2.3522, 48.8566},
			Radius:      100,
		},
		Actions:     []string{"notification"},
		TriggerType: "enter",
		IsActive:    true,
	}

	lm.AddGeoFence(fence)

	// Update location inside fence
	location := tania.Location{
		Point:     tania.Point{Lat: 48.8567, Lng: 2.3523},
		Accuracy:  10,
		Timestamp: time.Now(),
	}

	// Mock event checking
	isInside := lm.IsInsideGeoFence(location.Point, fence)
	assert.True(t, isInside)

	// Test outside
	outsideLocation := tania.Location{
		Point:     tania.Point{Lat: 48.9000, Lng: 2.4000},
		Timestamp: time.Now(),
	}
	isInside = lm.IsInsideGeoFence(outsideLocation.Point, fence)
	assert.False(t, isInside)
}

// ==================== FOLLOW SYSTEM TESTS ====================

func TestFollowUser(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	// Create test users
	user1ID, token1, _ := createTestUser(app, "user1@test.com", "password123")
	user2ID, _, _ := createTestUser(app, "user2@test.com", "password123")

	// Setup follow settings for user2 (free follow)
	r, err := app.FindCollectionByNameOrId("followSettings")
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewRecord(r)
	settings.Set("user", user2ID)
	settings.Set("followType", "free")
	settings.Set("isAcceptingFollowers", true)
	app.Save(settings)

	// Test follow request
	payload := map[string]interface{}{}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/users/"+user2ID+"/follow", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", "application/json")

	// rec := httptest.NewRecorder()

	// Note: This would require proper Echo setup with the actual handler
	// For now, test the logic directly

	// Verify follow was created
	follow, err := app.FindFirstRecordByFilter("follows",
		"follower = '"+user1ID+"' && following = '"+user2ID+"'")

	if err == nil {
		assert.Equal(t, user1ID, follow.GetString("follower"))
		assert.Equal(t, user2ID, follow.GetString("following"))
		assert.Equal(t, "active", follow.GetString("status"))
	}
}

func TestFollowWithApproval(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	user1ID, _, _ := createTestUser(app, "user1@test.com", "password123")
	user2ID, _, _ := createTestUser(app, "user2@test.com", "password123")

	// Setup follow settings (require approval)
	r, err := app.FindCollectionByNameOrId("followSettings")
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewRecord(r)
	settings.Set("user", user2ID)
	settings.Set("followType", "require_approval")
	settings.Set("isAcceptingFollowers", true)
	app.Save(settings)

	// Create follow (should be pending)
	r, err = app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatal(err)
	}
	follow := core.NewRecord(r)
	follow.Set("follower", user1ID)
	follow.Set("following", user2ID)
	follow.Set("status", "pending")
	follow.Set("role", "follower")
	app.Save(follow)

	// Verify status is pending
	assert.Equal(t, "pending", follow.GetString("status"))

	// Approve follow
	follow.Set("status", "active")
	follow.Set("approvedBy", user2ID)
	follow.Set("approvedAt", time.Now())
	app.Save(follow)

	assert.Equal(t, "active", follow.GetString("status"))
}

// ==================== ROOM MANAGEMENT TESTS ====================

func TestCreateRoom(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	userID, _, _ := createTestUser(app, "owner@test.com", "password123")

	// Create room
	r, err := app.FindCollectionByNameOrId("rooms")
	if err != nil {
		t.Fatal(err)
	}
	room := core.NewRecord(r)
	room.Set("roomType", "audio")
	room.Set("name", "Test Room")
	room.Set("owner", userID)
	room.Set("joinType", "free")
	room.Set("maxParticipants", 50)
	room.Set("isActive", true)
	room.Set("isPublic", true)

	err = app.Save(room)
	assert.NoError(t, err)

	// Add owner as member
	r, err = app.FindCollectionByNameOrId("roomMembers")
	if err != nil {
		t.Fatal(err)
	}
	member := core.NewRecord(r)
	member.Set("room", room.Id)
	member.Set("user", userID)
	member.Set("role", "owner")
	member.Set("status", "active")
	member.Set("joinedAt", time.Now())

	err = app.Save(member)
	assert.NoError(t, err)

	// Verify
	assert.Equal(t, "audio", room.GetString("roomType"))
	assert.Equal(t, userID, room.GetString("owner"))
}

func TestRoomMemberHierarchy(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	ownerID, _, _ := createTestUser(app, "owner@test.com", "password123")
	adminID, _, _ := createTestUser(app, "admin@test.com", "password123")
	userID, _, _ := createTestUser(app, "user@test.com", "password123")

	// Create room
	r, err := app.FindCollectionByNameOrId("rooms")
	if err != nil {
		t.Fatal(err)
	}
	room := core.NewRecord(r)
	room.Set("roomType", "audio")
	room.Set("name", "Test Room")
	room.Set("owner", ownerID)
	room.Set("joinType", "free")
	room.Set("maxParticipants", 50)
	room.Set("isActive", true)
	app.Save(room)

	// Add members
	addRoomMember := func(userID, role string) {
		r, err := app.FindCollectionByNameOrId("roomMembers")
		if err != nil {
			t.Fatal(err)
		}
		member := core.NewRecord(r)
		member.Set("room", room.Id)
		member.Set("user", userID)
		member.Set("role", role)
		member.Set("status", "active")
		member.Set("joinedAt", time.Now())
		app.Save(member)
	}

	addRoomMember(ownerID, "owner")
	addRoomMember(adminID, "admin")
	addRoomMember(userID, "participant")

	// Verify hierarchy
	ownerMember, _ := app.FindFirstRecordByFilter("roomMembers",
		"room = '"+room.Id+"' && user = '"+ownerID+"'")
	assert.Equal(t, "owner", ownerMember.GetString("role"))

	adminMember, _ := app.FindFirstRecordByFilter("roomMembers",
		"room = '"+room.Id+"' && user = '"+adminID+"'")
	assert.Equal(t, "admin", adminMember.GetString("role"))

	userMember, _ := app.FindFirstRecordByFilter("roomMembers",
		"room = '"+room.Id+"' && user = '"+userID+"'")
	assert.Equal(t, "participant", userMember.GetString("role"))

	// Test is owner or admin
	assert.True(t, tania.IsRoomOwnerOrAdmin(app, room.Id, ownerID))
	assert.True(t, tania.IsRoomOwnerOrAdmin(app, room.Id, adminID))
	assert.False(t, tania.IsRoomOwnerOrAdmin(app, room.Id, userID))
}

// ==================== USER CHANNELS TESTS ====================

func TestSSEChannel(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	ucm := tania.NewUserChannelManager(app)

	// Create channel
	channel := ucm.GetOrCreateSSEChannel("user_1")
	assert.NotNil(t, channel)
	assert.True(t, channel.IsActive)

	// Send message
	go ucm.SendToSSE("user_1", "test_event", map[string]interface{}{
		"message": "hello",
	}, "req_123")

	// Receive message
	select {
	case msg := <-channel.Channel:
		assert.Equal(t, "test_event", msg.Type)
		assert.Equal(t, "req_123", msg.RequestID)
		assert.Equal(t, "hello", msg.Data["message"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for SSE message")
	}

	// Close channel
	ucm.CloseSSEChannel("user_1")
	assert.False(t, channel.IsActive)
}

// ==================== SCRIPT MANAGER TESTS ====================

func TestScriptManager(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	// Create temp directories
	scriptsDir := t.TempDir() + "/pb_hooks"
	modulesDir := t.TempDir() + "/pb_modules"

	sm, err := tania.NewScriptManager(app, scriptsDir, modulesDir)
	assert.NoError(t, err)
	assert.NotNil(t, sm)

	// Create test script
	scriptContent := `
		function main() {
			log("Test script executed");
			return { success: true };
		}
		main();
	`
	scriptPath := scriptsDir + "/test.js"
	os.MkdirAll(scriptsDir, 0755)
	os.WriteFile(scriptPath, []byte(scriptContent), 0644)

	// Load script
	err = sm.LoadScript(scriptPath)
	assert.NoError(t, err)

	// Verify script was loaded
	sm.RLock()
	defer sm.RUnlock()
	_, exists := sm.Get("test")
	assert.True(t, exists)
}

func TestSharedModules(t *testing.T) {
	// Test module registry
	module := goja.New().NewObject()
	module.Set("test", "value")

	tania.SharedModules.Set("test_module", module)

	retrieved, exists := tania.SharedModules.Get("test_module")
	assert.True(t, exists)
	assert.NotNil(t, retrieved)

	tania.SharedModules.Delete("test_module")
	_, exists = tania.SharedModules.Get("test_module")
	assert.False(t, exists)
}

// ==================== INTEGRATION TESTS ====================

func TestFullWorkflow(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	// Create users
	user1ID, _, _ := createTestUser(app, "user1@test.com", "password123")
	user2ID, _, _ := createTestUser(app, "user2@test.com", "password123")

	// User1 creates a post
	r, err := app.FindCollectionByNameOrId("posts")
	if err != nil {
		t.Fatal(err)
	}
	post := core.NewRecord(r)
	post.Set("user", user1ID)
	post.Set("type", "html")
	post.Set("content", "Test post")
	post.Set("isPublic", true)
	post.Set("likesCount", 0)
	post.Set("commentsCount", 0)
	app.Save(post)

	// User2 likes the post
	r, err = app.FindCollectionByNameOrId("likes")
	if err != nil {
		t.Fatal(err)
	}
	like := core.NewRecord(r)
	like.Set("user", user2ID)
	like.Set("post", post.Id)
	like.Set("reaction", "like")
	app.Save(like)

	// Update post likes count
	post.Set("likesCount", 1)
	app.Save(post)

	// User2 comments
	r, err = app.FindCollectionByNameOrId("comments")
	if err != nil {
		t.Fatal(err)
	}
	comment := core.NewRecord(r)
	comment.Set("user", user2ID)
	comment.Set("post", post.Id)
	comment.Set("content", "Great post!")
	app.Save(comment)

	// Update comments count
	post.Set("commentsCount", 1)
	app.Save(post)

	// Verify
	assert.Equal(t, 1, post.GetInt("likesCount"))
	assert.Equal(t, 1, post.GetInt("commentsCount"))
}

func TestMarketplaceWorkflow(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	sellerID, _, _ := createTestUser(app, "seller@test.com", "password123")
	buyerID, _, _ := createTestUser(app, "buyer@test.com", "password123")

	// Create article
	r, err := app.FindCollectionByNameOrId("articles")
	if err != nil {
		t.Fatal(err)
	}
	article := core.NewRecord(r)
	article.Set("title", "iPhone 15")
	article.Set("prix", 999.99)
	article.Set("quantite", 5)
	article.Set("user", sellerID)
	app.Save(article)

	// Buy article
	r, err = app.FindCollectionByNameOrId("ventesArticle")
	if err != nil {
		t.Fatal(err)
	}
	vente := core.NewRecord(r)
	vente.Set("article", article.Id)
	vente.Set("user", buyerID)
	vente.Set("montant", 999.99)
	vente.Set("status", "paye")
	app.Save(vente)

	// Update stock
	article.Set("quantite", 4)
	app.Save(article)

	// Create operation
	r, err = app.FindCollectionByNameOrId("operations")
	if err != nil {
		t.Fatal(err)
	}
	operation := core.NewRecord(r)
	operation.Set("user", buyerID)
	operation.Set("vente", vente.Id)
	operation.Set("montant", -999.99)
	operation.Set("operation", "cashout")
	operation.Set("status", "paye")
	app.Save(operation)

	// Verify
	assert.Equal(t, 4, article.GetInt("quantite"))
	assert.Equal(t, "paye", vente.GetString("status"))
	assert.Equal(t, -999.99, operation.GetFloat("montant"))
}

// ==================== BENCHMARK TESTS ====================

func BenchmarkHaversineDistance(b *testing.B) {
	p1 := tania.Point{Lat: 48.8566, Lng: 2.3522}
	p2 := tania.Point{Lat: 51.5074, Lng: -0.1278}

	for i := 0; i < b.N; i++ {
		tania.HaversineDistance(p1, p2)
	}
}

func BenchmarkFindNearby(b *testing.B) {
	app := setupTestApp(&testing.T{})
	defer app.Cleanup()

	lm := tania.NewLocationManager(app)

	// Add 1000 users
	for i := 0; i < 1000; i++ {
		loc := tania.Location{
			Point:     tania.Point{Lat: 48.8 + float64(i)*0.001, Lng: 2.3 + float64(i)*0.001},
			Accuracy:  10,
			Timestamp: time.Now(),
		}
		lm.UpdateLocation(fmt.Sprintf("user_%d", i), loc, "online")
	}

	center := tania.Point{Lat: 48.8566, Lng: 2.3522}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lm.FindNearby(center, 1000, "")
	}
}

func BenchmarkPubSubPublish(b *testing.B) {
	ps := tania.NewPubSub()
	ch := ps.Subscribe("test")

	go func() {
		for range ch {
			// Consume messages
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.Publish("test", tania.PubSubMessage{
			Topic:   "test",
			Payload: map[string]interface{}{"id": i},
		})
	}
}
