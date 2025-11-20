package app

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// ==================== GEO TYPES ====================

type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Location struct {
	Point     Point     `json:"point"`
	Accuracy  float64   `json:"accuracy"` // meters
	Altitude  float64   `json:"altitude"` // meters
	Speed     float64   `json:"speed"`    // m/s
	Heading   float64   `json:"heading"`  // degrees
	Timestamp time.Time `json:"timestamp"`
}

type UserLocation struct {
	UserID    string    `json:"user_id"`
	Location  Location  `json:"location"`
	Presence  string    `json:"presence"` // online, away, busy, offline
	UpdatedAt time.Time `json:"updated_at"`
}

type GeoJSONGeometry struct {
	Type        string      `json:"type"` // Point, Polygon, Circle
	Coordinates interface{} `json:"coordinates"`
	Radius      float64     `json:"radius,omitempty"` // For Circle type (meters)
}

type GeoFence struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Geometry    GeoJSONGeometry        `json:"geometry"`
	Actions     []string               `json:"actions"`      // chat, notification, call, ads
	TriggerType string                 `json:"trigger_type"` // enter, exit, dwell
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedBy   string                 `json:"created_by"`
	IsActive    bool                   `json:"is_active"`
}

type GeoEvent struct {
	Type      string                 `json:"type"` // user_entered, user_exited, user_nearby
	UserID    string                 `json:"user_id"`
	FenceID   string                 `json:"fence_id,omitempty"`
	Location  Location               `json:"location"`
	Distance  float64                `json:"distance,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ==================== LOCATION MANAGER ====================

type LocationManager struct {
	locations  map[string]*UserLocation // userID -> location
	fences     map[string]*GeoFence     // fenceID -> geofence
	userFences map[string][]string      // userID -> fenceIDs user is inside
	app        core.App
	mu         sync.RWMutex
}

func NewLocationManager(app core.App) *LocationManager {
	return &LocationManager{
		locations:  make(map[string]*UserLocation),
		fences:     make(map[string]*GeoFence),
		userFences: make(map[string][]string),
		app:        app,
	}
}

// Update user location
func (lm *LocationManager) UpdateLocation(userID string, location Location, presence string) error {
	lm.mu.Lock()

	userLoc := &UserLocation{
		UserID:    userID,
		Location:  location,
		Presence:  presence,
		UpdatedAt: time.Now(),
	}

	oldLocation := lm.locations[userID]
	lm.locations[userID] = userLoc

	lm.mu.Unlock()

	// Check geofences
	go lm.checkGeofences(userID, userLoc, oldLocation)

	// Broadcast location update
	pubsub.Publish("location_updates", PubSubMessage{
		Topic: "location_updates",
		Payload: map[string]interface{}{
			"user_id":  userID,
			"location": location,
			"presence": presence,
		},
	})

	// Save to database
	go lm.saveLocationToDB(userID, location, presence)

	return nil
}

// Get user location
func (lm *LocationManager) GetLocation(userID string) (*UserLocation, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	loc, exists := lm.locations[userID]
	return loc, exists
}

// Find users nearby
func (lm *LocationManager) FindNearby(point Point, radiusMeters float64, excludeUserID string) []*UserLocation {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	nearby := []*UserLocation{}

	for userID, userLoc := range lm.locations {
		if userID == excludeUserID {
			continue
		}

		// Check if stale (older than 5 minutes)
		if time.Since(userLoc.UpdatedAt) > 5*time.Minute {
			continue
		}

		distance := HaversineDistance(point, userLoc.Location.Point)
		if distance <= radiusMeters {
			nearby = append(nearby, userLoc)
		}
	}

	return nearby
}

// Find users in polygon
func (lm *LocationManager) FindInPolygon(polygon []Point) []*UserLocation {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	inside := []*UserLocation{}

	for _, userLoc := range lm.locations {
		if time.Since(userLoc.UpdatedAt) > 5*time.Minute {
			continue
		}

		if IsPointInPolygon(userLoc.Location.Point, polygon) {
			inside = append(inside, userLoc)
		}
	}

	return inside
}

// Get all users with specific presence
func (lm *LocationManager) GetUsersByPresence(presence string) []*UserLocation {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	users := []*UserLocation{}
	for _, userLoc := range lm.locations {
		if userLoc.Presence == presence {
			users = append(users, userLoc)
		}
	}

	return users
}

// Add geofence
func (lm *LocationManager) AddGeoFence(fence *GeoFence) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.fences[fence.ID] = fence
}

// Remove geofence
func (lm *LocationManager) RemoveGeoFence(fenceID string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	delete(lm.fences, fenceID)
}

// Check if user entered/exited geofences
func (lm *LocationManager) checkGeofences(userID string, newLoc *UserLocation, oldLoc *UserLocation) {
	lm.mu.RLock()
	activeFences := make([]*GeoFence, 0, len(lm.fences))
	for _, fence := range lm.fences {
		if fence.IsActive {
			activeFences = append(activeFences, fence)
		}
	}
	previousFences := lm.userFences[userID]
	lm.mu.RUnlock()

	currentFences := []string{}

	for _, fence := range activeFences {
		isInside := lm.IsInsideGeoFence(newLoc.Location.Point, fence)

		if isInside {
			currentFences = append(currentFences, fence.ID)
		}

		wasInside := contains(previousFences, fence.ID)

		// User entered fence
		if isInside && !wasInside && (fence.TriggerType == "enter" || fence.TriggerType == "") {
			lm.triggerGeoEvent(GeoEvent{
				Type:      "user_entered",
				UserID:    userID,
				FenceID:   fence.ID,
				Location:  newLoc.Location,
				Timestamp: time.Now(),
			}, fence)
		}

		// User exited fence
		if !isInside && wasInside && fence.TriggerType == "exit" {
			lm.triggerGeoEvent(GeoEvent{
				Type:      "user_exited",
				UserID:    userID,
				FenceID:   fence.ID,
				Location:  newLoc.Location,
				Timestamp: time.Now(),
			}, fence)
		}
	}

	lm.mu.Lock()
	lm.userFences[userID] = currentFences
	lm.mu.Unlock()
}

// Check if point is inside geofence
func (lm *LocationManager) IsInsideGeoFence(point Point, fence *GeoFence) bool {
	switch fence.Geometry.Type {
	case "Point":
		// Check circle around point
		coords := fence.Geometry.Coordinates.([]interface{})
		center := Point{
			Lat: coords[1].(float64),
			Lng: coords[0].(float64),
		}
		distance := HaversineDistance(point, center)
		return distance <= fence.Geometry.Radius

	case "Circle":
		coords := fence.Geometry.Coordinates.([]interface{})
		center := Point{
			Lat: coords[1].(float64),
			Lng: coords[0].(float64),
		}
		distance := HaversineDistance(point, center)
		return distance <= fence.Geometry.Radius

	case "Polygon":
		coords := fence.Geometry.Coordinates.([]interface{})
		polygon := make([]Point, len(coords))
		for i, coord := range coords {
			c := coord.([]interface{})
			polygon[i] = Point{
				Lat: c[1].(float64),
				Lng: c[0].(float64),
			}
		}
		return IsPointInPolygon(point, polygon)
	}

	return false
}

// Trigger geofence event
func (lm *LocationManager) triggerGeoEvent(event GeoEvent, fence *GeoFence) {
	log.Printf("🎯 Geo Event: %s - User %s %s fence %s", event.Type, event.UserID, event.Type, fence.Name)

	// Publish event
	pubsub.Publish("geo_events", PubSubMessage{
		Topic: "geo_events",
		Payload: map[string]interface{}{
			"type":     event.Type,
			"user_id":  event.UserID,
			"fence_id": event.FenceID,
			"fence":    fence,
			"location": event.Location,
		},
	})

	// Execute actions
	for _, action := range fence.Actions {
		lm.executeAction(action, event, fence)
	}
}

// Execute geofence action
func (lm *LocationManager) executeAction(action string, event GeoEvent, fence *GeoFence) {
	switch action {
	case "notification":
		lm.sendNotification(event, fence)
	case "chat":
		lm.initiateChat(event, fence)
	case "ads":
		lm.showAds(event, fence)
	case "call":
		lm.initiateCall(event, fence)
	default:
		log.Printf("Unknown action: %s", action)
	}
}

func (lm *LocationManager) sendNotification(event GeoEvent, fence *GeoFence) {
	pubsub.Publish("notifications", PubSubMessage{
		Topic: "notifications",
		Payload: map[string]interface{}{
			"type":    "geofence",
			"user_id": event.UserID,
			"title":   fmt.Sprintf("You entered %s", fence.Name),
			"message": fence.Metadata["notification_message"],
			"data": map[string]interface{}{
				"fence_id": fence.ID,
				"event":    event.Type,
			},
		},
	})

	log.Printf("📬 Notification sent to user %s", event.UserID)
}

func (lm *LocationManager) initiateChat(event GeoEvent, fence *GeoFence) {
	// Create a room or send chat invite
	pubsub.Publish("chat_invites", PubSubMessage{
		Topic: "chat_invites",
		Payload: map[string]interface{}{
			"user_id":  event.UserID,
			"fence_id": fence.ID,
			"message":  fence.Metadata["chat_message"],
		},
	})

	log.Printf("💬 Chat initiated for user %s", event.UserID)
}

func (lm *LocationManager) showAds(event GeoEvent, fence *GeoFence) {
	pubsub.Publish("ads", PubSubMessage{
		Topic: "ads",
		Payload: map[string]interface{}{
			"user_id":  event.UserID,
			"fence_id": fence.ID,
			"ad_data":  fence.Metadata["ad_data"],
		},
	})

	log.Printf("📢 Ads shown to user %s", event.UserID)
}

func (lm *LocationManager) initiateCall(event GeoEvent, fence *GeoFence) {
	pubsub.Publish("call_invites", PubSubMessage{
		Topic: "call_invites",
		Payload: map[string]interface{}{
			"user_id":  event.UserID,
			"fence_id": fence.ID,
			"call_to":  fence.Metadata["call_to"],
		},
	})

	log.Printf("📞 Call initiated for user %s", event.UserID)
}

// Save location to database
func (lm *LocationManager) saveLocationToDB(userID string, location Location, presence string) {
	user, err := lm.app.FindRecordById("users", userID)
	if err != nil {
		log.Printf("Error finding user %s: %v", userID, err)
		return
	}

	locationData := map[string]interface{}{
		"lat":       location.Point.Lat,
		"lng":       location.Point.Lng,
		"accuracy":  location.Accuracy,
		"altitude":  location.Altitude,
		"speed":     location.Speed,
		"heading":   location.Heading,
		"timestamp": location.Timestamp.Format(time.RFC3339),
	}

	user.Set("location", locationData)
	user.Set("presence", presence)
	user.Set("lastSeen", time.Now())

	if err := lm.app.Save(user); err != nil {
		log.Printf("Error saving location for user %s: %v", userID, err)
	}
}

// ==================== GEO MATH FUNCTIONS ====================

// Haversine distance in meters
func HaversineDistance(p1, p2 Point) float64 {
	const R = 6371000 // Earth radius in meters

	lat1 := p1.Lat * math.Pi / 180
	lat2 := p2.Lat * math.Pi / 180
	deltaLat := (p2.Lat - p1.Lat) * math.Pi / 180
	deltaLng := (p2.Lng - p1.Lng) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// Check if point is inside polygon (ray casting algorithm)
func IsPointInPolygon(point Point, polygon []Point) bool {
	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		if (polygon[i].Lat > point.Lat) != (polygon[j].Lat > point.Lat) &&
			point.Lng < (polygon[j].Lng-polygon[i].Lng)*(point.Lat-polygon[i].Lat)/(polygon[j].Lat-polygon[i].Lat)+polygon[i].Lng {
			inside = !inside
		}
		j = i
	}

	return inside
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ==================== HTTP HANDLERS ====================

// Update location
func handleUpdateLocation(c *core.RequestEvent) error {
	userID := c.Get("userID").(string)

	var req struct {
		Location Location `json:"location"`
		Presence string   `json:"presence"`
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	if req.Presence == "" {
		req.Presence = "online"
	}

	req.Location.Timestamp = time.Now()

	if err := locationManager.UpdateLocation(userID, req.Location, req.Presence); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success":  true,
		"user_id":  userID,
		"location": req.Location,
		"presence": req.Presence,
	})
}

// Get user location
func handleGetLocation(c *core.RequestEvent) error {
	targetUserID := c.Request.PathValue("userId")

	userLoc, exists := locationManager.GetLocation(targetUserID)
	if !exists {
		return c.JSON(404, map[string]string{"error": "location not found"})
	}

	return c.JSON(200, userLoc)
}

// Find nearby users
func handleFindNearby(c *core.RequestEvent) error {
	userID := c.Get("userID").(string)

	var req struct {
		Point  Point   `json:"point"`
		Radius float64 `json:"radius"` // meters
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	nearby := locationManager.FindNearby(req.Point, req.Radius, userID)

	return c.JSON(200, map[string]interface{}{
		"count": len(nearby),
		"users": nearby,
	})
}

// Find users in polygon
func handleFindInPolygon(c *core.RequestEvent) error {
	var req struct {
		Polygon []Point `json:"polygon"`
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	users := locationManager.FindInPolygon(req.Polygon)

	return c.JSON(200, map[string]interface{}{
		"count": len(users),
		"users": users,
	})
}

// Get users by presence
func handleGetByPresence(c *core.RequestEvent) error {
	presence := c.Request.URL.Query().Get("presence")

	if presence == "" {
		presence = "online"
	}

	users := locationManager.GetUsersByPresence(presence)

	return c.JSON(200, map[string]interface{}{
		"presence": presence,
		"count":    len(users),
		"users":    users,
	})
}

// Create geofence
func handleCreateGeoFence(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)

	var fence GeoFence
	if err := c.BindBody(&fence); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	fence.ID = generateID()
	fence.CreatedBy = userID
	fence.IsActive = true

	locationManager.AddGeoFence(&fence)

	// Save to database
	r, err := app.FindCollectionByNameOrId("geofences")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	geofenceRecord := core.NewRecord(r)
	geofenceRecord.Set("fenceId", fence.ID)
	geofenceRecord.Set("name", fence.Name)

	geometryJSON, _ := json.Marshal(fence.Geometry)
	geofenceRecord.Set("geometry", string(geometryJSON))

	actionsJSON, _ := json.Marshal(fence.Actions)
	geofenceRecord.Set("actions", string(actionsJSON))

	geofenceRecord.Set("triggerType", fence.TriggerType)

	metadataJSON, _ := json.Marshal(fence.Metadata)
	geofenceRecord.Set("metadata", string(metadataJSON))

	geofenceRecord.Set("createdBy", userID)
	geofenceRecord.Set("isActive", true)

	if err := app.Save(geofenceRecord); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, fence)
}

// Update geofence
func handleUpdateGeoFence(c *core.RequestEvent) error {
	fenceID := c.Request.PathValue("fenceId")

	var updates struct {
		IsActive *bool                  `json:"is_active,omitempty"`
		Actions  []string               `json:"actions,omitempty"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.BindBody(&updates); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	locationManager.mu.Lock()
	fence, exists := locationManager.fences[fenceID]
	if !exists {
		locationManager.mu.Unlock()
		return c.JSON(404, map[string]string{"error": "geofence not found"})
	}

	if updates.IsActive != nil {
		fence.IsActive = *updates.IsActive
	}
	if updates.Actions != nil {
		fence.Actions = updates.Actions
	}
	if updates.Metadata != nil {
		fence.Metadata = updates.Metadata
	}

	locationManager.mu.Unlock()

	return c.JSON(200, fence)
}

// Delete geofence
func handleDeleteGeoFence(c *core.RequestEvent) error {
	app := c.App
	fenceID := c.Request.PathValue("fenceId")

	locationManager.RemoveGeoFence(fenceID)

	// Delete from database
	geofences, _ := app.FindRecordsByFilter("geofences", fmt.Sprintf("fenceId = '%s'", fenceID), "", 1, 0)
	if len(geofences) > 0 {
		app.Delete(geofences[0])
	}

	return c.JSON(200, map[string]interface{}{
		"success":  true,
		"fence_id": fenceID,
	})
}

// List geofences
func handleListGeoFences(c *core.RequestEvent) error {
	locationManager.mu.RLock()
	defer locationManager.mu.RUnlock()

	fences := make([]*GeoFence, 0, len(locationManager.fences))
	for _, fence := range locationManager.fences {
		fences = append(fences, fence)
	}

	return c.JSON(200, map[string]interface{}{
		"count":  len(fences),
		"fences": fences,
	})
}

// Notify users in zone
func handleNotifyInZone(c *core.RequestEvent) error {
	var req struct {
		Point   *Point                 `json:"point,omitempty"`
		Radius  float64                `json:"radius,omitempty"`
		Polygon []Point                `json:"polygon,omitempty"`
		Message string                 `json:"message"`
		Title   string                 `json:"title"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	var users []*UserLocation

	if req.Point != nil {
		users = locationManager.FindNearby(*req.Point, req.Radius, "")
	} else if req.Polygon != nil {
		users = locationManager.FindInPolygon(req.Polygon)
	} else {
		return c.JSON(400, map[string]string{"error": "point or polygon required"})
	}

	// Send notifications
	notifiedCount := 0
	for _, user := range users {
		pubsub.Publish("notifications", PubSubMessage{
			Topic: "notifications",
			Payload: map[string]interface{}{
				"type":    "zone_notification",
				"user_id": user.UserID,
				"title":   req.Title,
				"message": req.Message,
				"data":    req.Data,
			},
		})
		notifiedCount++
	}

	return c.JSON(200, map[string]interface{}{
		"success":        true,
		"notified_count": notifiedCount,
	})
}

// WebRTC: Broadcast location to room
func handleBroadcastLocationToRoom(c *core.RequestEvent) error {
	roomID := c.Request.PathValue("roomId")
	userID := c.Get("userID").(string)

	var req struct {
		Location Location `json:"location"`
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	roomsMutex.RLock()
	room, exists := rooms[roomID]
	roomsMutex.RUnlock()

	if !exists {
		return c.JSON(404, map[string]string{"error": "room not found"})
	}

	// Broadcast location to all participants
	room.broadcastEvent("location_update", map[string]interface{}{
		"user_id":  userID,
		"location": req.Location,
	})

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"room_id": roomID,
	})
}

// ==================== SETUP COLLECTIONS ====================

func SetupLocationCollections(app core.App) error {
	return app.RunInTransaction(func(txApp core.App) error {
		// Update users collection to add location and presence
		users, err := txApp.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.Fields.Add(
			// Add location field (JSON)
			&core.JSONField{
				Name: "location",
			},
			// Add presence field
			&core.SelectField{
				Name:      "presence",
				MaxSelect: 1,
				Values:    []string{"online", "away", "busy", "offline"},
			},

			// Add lastSeen field
			&core.DateField{
				Name: "lastSeen",
			})

		if err := txApp.Save(users); err != nil {
			return err
		}

		// Geofences collection
		geofences := &core.Collection{}
		geofences.Name = "geofences"
		geofences.Type = core.CollectionTypeBase
		geofences.Fields.Add(
			&core.TextField{Name: "fenceId", Required: true},
			&core.TextField{Name: "name", Required: true},
			&core.JSONField{Name: "geometry", Required: true},
			&core.JSONField{Name: "actions"},
			&core.TextField{Name: "triggerType"},
			&core.JSONField{Name: "metadata"},
			&core.RelationField{Name: "createdBy", CollectionId: "_pb_users_auth_", MaxSelect: 1},
			&core.BoolField{Name: "isActive"},
		)
		geofences.Indexes = []string{"CREATE UNIQUE INDEX idx_fence_id ON geofences (fenceId)"}

		if err := txApp.Save(geofences); err != nil {
			return err
		}

		// Location history collection (optional)
		locationHistory := &core.Collection{}
		locationHistory.Name = "locationHistory"
		locationHistory.Type = core.CollectionTypeBase
		locationHistory.Fields.Add(
			&core.RelationField{Name: "user", Required: true, CollectionId: "_pb_users_auth_", MaxSelect: 1},
			&core.JSONField{Name: "location", Required: true},
			&core.TextField{Name: "presence"},
			&core.NumberField{Name: "accuracy"},
		)
		locationHistory.Indexes = []string{"CREATE INDEX idx_user_created ON locationHistory (user, created)"}

		return txApp.Save(locationHistory)
	})
}
