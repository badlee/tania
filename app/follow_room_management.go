package app

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// ==================== FOLLOW/FOLLOWER TYPES ====================

type FollowType string

const (
	FollowTypeFree            FollowType = "free"
	FollowTypeRequireApproval FollowType = "require_approval"
	FollowTypePaidPeriod      FollowType = "paid_period"
	FollowTypePaidLifetime    FollowType = "paid_lifetime"
)

type FollowStatus string

const (
	FollowStatusPending  FollowStatus = "pending"
	FollowStatusActive   FollowStatus = "active"
	FollowStatusRejected FollowStatus = "rejected"
	FollowStatusExpired  FollowStatus = "expired"
)

type FollowerRole string

const (
	FollowerRoleFollower FollowerRole = "follower"
	FollowerRoleAdmin    FollowerRole = "admin"
)

// ==================== ROOM MANAGEMENT TYPES ====================

type RoomRole string

const (
	RoomRoleOwner       RoomRole = "owner"
	RoomRoleAdmin       RoomRole = "admin"
	RoomRoleParticipant RoomRole = "participant"
)

type RoomJoinType string

const (
	RoomJoinFree            RoomJoinType = "free"
	RoomJoinRequireApproval RoomJoinType = "require_approval"
	RoomJoinPaidPeriod      RoomJoinType = "paid_period"
	RoomJoinPaidLifetime    RoomJoinType = "paid_lifetime"
)

type RoomMemberStatus string

const (
	RoomMemberStatusPending  RoomMemberStatus = "pending"
	RoomMemberStatusActive   RoomMemberStatus = "active"
	RoomMemberStatusRejected RoomMemberStatus = "rejected"
	RoomMemberStatusExpired  RoomMemberStatus = "expired"
	RoomMemberStatusBanned   RoomMemberStatus = "banned"
)

// ==================== SETUP COLLECTIONS ====================

func SetupFollowAndRoomCollections(app core.App) error {
	return app.RunInTransaction(func(txApp core.App) error {

		// ==================== FOLLOW SYSTEM ====================

		// followSettings - Configuration du profil utilisateur
		followSettings := &core.Collection{}
		followSettings.Name = "followSettings"
		followSettings.Type = core.CollectionTypeBase
		followSettings.Fields.Add(
			&core.RelationField{
				Name:         "user",
				Required:     true,
				CollectionId: "_pb_users_auth_", MaxSelect: 1,
			},
			&core.SelectField{
				Name:      "followType",
				Required:  true,
				MaxSelect: 1, Values: []string{"free", "require_approval", "paid_period", "paid_lifetime"},
			},
			&core.NumberField{
				Name: "price",
				Min:  types.Pointer(float64(0)),
			},
			&core.NumberField{
				Name: "periodDays",
				Min:  types.Pointer(float64(1)),
			},
			&core.TextField{
				Name: "description",
			},
			&core.BoolField{
				Name: "isAcceptingFollowers",
			},
		)
		followSettings.Indexes = []string{"CREATE UNIQUE INDEX idx_follow_settings_user ON followSettings (user)"}
		if err := txApp.Save(followSettings); err != nil {
			return err
		}

		// follows - Relations follow/follower
		follows := &core.Collection{}
		follows.Name = "follows"
		follows.Type = core.CollectionTypeBase
		follows.Fields.Add(
			&core.RelationField{
				Name:         "follower",
				Required:     true,
				CollectionId: "_pb_users_auth_", MaxSelect: 1,
			},
			&core.RelationField{
				Name:         "following",
				Required:     true,
				CollectionId: "_pb_users_auth_", MaxSelect: 1,
			},
			&core.SelectField{
				Name:      "status",
				Required:  true,
				MaxSelect: 1, Values: []string{"pending", "active", "rejected", "expired"},
			},
			&core.SelectField{
				Name:      "role",
				Required:  true,
				MaxSelect: 1, Values: []string{"follower", "admin"},
			},
			&core.DateField{
				Name: "expiresAt",
			},
			&core.NumberField{
				Name: "paidAmount",
				Min:  types.Pointer(float64(0)),
			},
			&core.DateField{
				Name: "paidAt",
			},
			&core.DateField{
				Name: "approvedAt",
			},
			&core.RelationField{
				Name:         "approvedBy",
				CollectionId: "_pb_users_auth_", MaxSelect: 1,
			},
		)
		follows.Indexes = []string{
			"CREATE UNIQUE INDEX idx_follows_pair ON follows (follower, following)",
			"CREATE INDEX idx_follows_follower ON follows (follower)",
			"CREATE INDEX idx_follows_following ON follows (following)",
		}
		if err := txApp.Save(follows); err != nil {
			return err
		}

		// ==================== ROOM MANAGEMENT ====================

		// Update rooms collection
		roomsColl, err := txApp.FindCollectionByNameOrId("rooms")
		if err != nil {
			roomsColl = &core.Collection{}
			roomsColl.Name = "rooms"
			roomsColl.Type = core.CollectionTypeBase
		}

		roomsColl.Fields.Add(
			&core.SelectField{
				Name:      "roomType",
				Required:  true,
				MaxSelect: 1, Values: []string{"audio", "video", "data"},
			},
			&core.TextField{
				Name:     "name",
				Required: true,
			},
			&core.RelationField{
				Name:         "owner",
				Required:     true,
				CollectionId: "_pb_users_auth_", MaxSelect: 1,
			},
			&core.TextField{
				Name: "description",
			},
			&core.BoolField{
				Name: "isPublic",
			},
			&core.NumberField{
				Name: "maxParticipants",
				Min:  types.Pointer(float64(2)),
			},
			&core.SelectField{
				Name:      "joinType",
				Required:  true,
				MaxSelect: 1, Values: []string{"free", "require_approval", "paid_period", "paid_lifetime"},
			},
			&core.NumberField{
				Name: "price",
				Min:  types.Pointer(float64(0)),
			},
			&core.NumberField{
				Name: "periodDays",
				Min:  types.Pointer(float64(1)),
			},
			&core.BoolField{
				Name: "isActive",
			},
			&core.JSONField{
				Name: "metadata",
			},
		)

		if err := txApp.Save(roomsColl); err != nil {
			return err
		}

		// roomMembers - Membres des rooms
		roomMembers := &core.Collection{}
		roomMembers.Name = "roomMembers"
		roomMembers.Type = core.CollectionTypeBase
		roomMembers.Fields.Add(
			&core.RelationField{
				Name:         "room",
				Required:     true,
				CollectionId: "rooms", MaxSelect: 1,
			},
			&core.RelationField{
				Name:         "user",
				Required:     true,
				CollectionId: "_pb_users_auth_", MaxSelect: 1,
			},
			&core.SelectField{
				Name:      "role",
				Required:  true,
				MaxSelect: 1, Values: []string{"owner", "admin", "participant"},
			},
			&core.SelectField{
				Name:      "status",
				Required:  true,
				MaxSelect: 1, Values: []string{"pending", "active", "rejected", "expired", "banned"},
			},
			&core.DateField{
				Name: "expiresAt",
			},
			&core.NumberField{
				Name: "paidAmount",
				Min:  types.Pointer(float64(0)),
			},
			&core.DateField{
				Name: "paidAt",
			},
			&core.DateField{
				Name: "joinedAt",
			},
			&core.RelationField{
				Name:         "approvedBy",
				CollectionId: "_pb_users_auth_", MaxSelect: 1,
			},
			&core.JSONField{
				Name: "permissions",
			},
		)
		roomMembers.Indexes = []string{
			"CREATE UNIQUE INDEX idx_room_members_pair ON roomMembers (room, user)",
			"CREATE INDEX idx_room_members_room ON roomMembers (room)",
			"CREATE INDEX idx_room_members_user ON roomMembers (user)",
		}

		return txApp.Save(roomMembers)
	})
}

// ==================== FOLLOW HANDLERS ====================

func handleGetFollowSettings(c *core.RequestEvent) error {
	app := c.App
	targetUserID := c.Request.PathValue("userId")

	settings, err := app.FindFirstRecordByFilter("followSettings", fmt.Sprintf("user = '%s'", targetUserID))
	if err != nil {
		// Return default settings
		return c.JSON(200, map[string]interface{}{
			"user":                   targetUserID,
			"follow_type":            "free",
			"is_accepting_followers": true,
		})
	}

	return c.JSON(200, recordToMap(settings))
}

func handleUpdateFollowSettings(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)

	var req struct {
		FollowType           string  `json:"follow_type"`
		Price                float64 `json:"price"`
		PeriodDays           int     `json:"period_days"`
		Description          string  `json:"description"`
		IsAcceptingFollowers bool    `json:"is_accepting_followers"`
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Find or create settings
	settings, err := app.FindFirstRecordByFilter("followSettings", fmt.Sprintf("user = '%s'", userID))
	if err != nil {
		r, err := app.FindCollectionByNameOrId("followSettings")
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		settings = core.NewRecord(r)
		settings.Set("user", userID)
	}

	settings.Set("followType", req.FollowType)
	settings.Set("price", req.Price)
	settings.Set("periodDays", req.PeriodDays)
	settings.Set("description", req.Description)
	settings.Set("isAcceptingFollowers", req.IsAcceptingFollowers)

	if err := app.Save(settings); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, recordToMap(settings))
}

func handleFollowUser(c *core.RequestEvent) error {
	app := c.App
	followerID := c.Get("userID").(string)
	followingID := c.Request.PathValue("userId")

	if followerID == followingID {
		return c.JSON(400, map[string]string{"error": "cannot follow yourself"})
	}

	// Check if already following
	existing, _ := app.FindFirstRecordByFilter("follows",
		fmt.Sprintf("follower = '%s' && following = '%s'", followerID, followingID))
	if existing != nil {
		return c.JSON(400, map[string]string{"error": "already following"})
	}

	// Get follow settings
	settings, err := app.FindFirstRecordByFilter("followSettings",
		fmt.Sprintf("user = '%s'", followingID))

	followType := "free"
	price := 0.0
	periodDays := 0

	if err == nil {
		if !settings.GetBool("isAcceptingFollowers") {
			return c.JSON(403, map[string]string{"error": "user not accepting followers"})
		}
		followType = settings.GetString("followType")
		price = settings.GetFloat("price")
		periodDays = settings.GetInt("periodDays")
	}

	// Create follow record
	r, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	follow := core.NewRecord(r)
	follow.Set("follower", followerID)
	follow.Set("following", followingID)
	follow.Set("role", "follower")

	var status string
	var expiresAt *time.Time

	switch followType {
	case "free":
		status = "active"

	case "require_approval":
		status = "pending"

	case "paid_period":
		// Check payment (simplified)
		if price <= 0 {
			return c.JSON(400, map[string]string{"error": "invalid price"})
		}
		status = "active"
		expires := time.Now().AddDate(0, 0, periodDays)
		expiresAt = &expires
		follow.Set("paidAmount", price)
		follow.Set("paidAt", time.Now())

		// Create operation
		if err := createPaymentOperation(app, followerID, price,
			fmt.Sprintf("Follow subscription: %d days", periodDays), nil); err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
	case "paid_lifetime":
		if price <= 0 {
			return c.JSON(400, map[string]string{"error": "invalid price"})
		}
		status = "active"
		follow.Set("paidAmount", price)
		follow.Set("paidAt", time.Now())

		err := createPaymentOperation(app, followerID, price, "Follow lifetime subscription", nil)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
	}

	follow.Set("status", status)
	if expiresAt != nil {
		follow.Set("expiresAt", *expiresAt)
	}

	if err := app.Save(follow); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Notify
	if status == "pending" {
		userChannelManager.SendToSSE(followingID, "follow_request", map[string]interface{}{
			"follower_id": followerID,
			"follow_id":   follow.Id,
		}, "")
		userChannelManager.SendToUserRoom(followingID, "notification", map[string]interface{}{
			"type":    "follow_request",
			"user_id": followerID,
		}, "")
	} else if status == "active" {
		userChannelManager.SendToSSE(followingID, "new_follower", map[string]interface{}{
			"follower_id": followerID,
			"follow_id":   follow.Id,
		}, "")
	}

	return c.JSON(200, map[string]interface{}{
		"success":   true,
		"follow_id": follow.Id,
		"status":    status,
	})
}

func handleApproveFollow(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	followID := c.Request.PathValue("followId")

	follow, err := app.FindRecordById("follows", followID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "follow not found"})
	}

	// Check permission (must be the following user or admin)
	if follow.GetString("following") != userID {
		// Check if user is admin
		adminFollow, _ := app.FindFirstRecordByFilter("follows",
			fmt.Sprintf("follower = '%s' && following = '%s' && role = 'admin' && status = 'active'",
				userID, follow.GetString("following")))
		if adminFollow == nil {
			return c.JSON(403, map[string]string{"error": "permission denied"})
		}
	}

	follow.Set("status", "active")
	follow.Set("approvedAt", time.Now())
	follow.Set("approvedBy", userID)

	if err := app.Save(follow); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Notify follower
	followerID := follow.GetString("follower")
	userChannelManager.SendToSSE(followerID, "follow_approved", map[string]interface{}{
		"following_id": follow.GetString("following"),
		"follow_id":    followID,
	}, "")

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleRejectFollow(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	followID := c.Request.PathValue("followId")

	follow, err := app.FindRecordById("follows", followID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "follow not found"})
	}

	if follow.GetString("following") != userID {
		return c.JSON(403, map[string]string{"error": "permission denied"})
	}

	follow.Set("status", "rejected")

	if err := app.Save(follow); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleUnfollow(c *core.RequestEvent) error {
	app := c.App
	followerID := c.Get("userID").(string)
	followingID := c.Request.PathValue("userId")

	follow, err := app.FindFirstRecordByFilter("follows",
		fmt.Sprintf("follower = '%s' && following = '%s'", followerID, followingID))

	if err != nil {
		return c.JSON(404, map[string]string{"error": "not following"})
	}

	if err := app.Delete(follow); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleGetFollowers(c *core.RequestEvent) error {
	app := c.App
	userID := c.Request.PathValue("userId")
	status := c.Request.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}

	filter := fmt.Sprintf("following = '%s' && status = '%s'", userID, status)
	follows, err := app.FindRecordsByFilter("follows", filter, "-created", 100, 0)

	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	result := make([]map[string]interface{}, len(follows))
	for i, follow := range follows {
		result[i] = recordToMap(follow)
	}

	return c.JSON(200, map[string]interface{}{
		"followers": result,
		"count":     len(result),
	})
}

func handleGetFollowing(c *core.RequestEvent) error {
	app := c.App
	userID := c.Request.PathValue("userId")

	follows, err := app.FindRecordsByFilter("follows",
		fmt.Sprintf("follower = '%s' && status = 'active'", userID), "-created", 100, 0)

	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	result := make([]map[string]interface{}, len(follows))
	for i, follow := range follows {
		result[i] = recordToMap(follow)
	}

	return c.JSON(200, map[string]interface{}{
		"following": result,
		"count":     len(result),
	})
}

func handlePromoteFollowerToAdmin(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	followID := c.Request.PathValue("followId")

	follow, err := app.FindRecordById("follows", followID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "follow not found"})
	}

	// Only the following user can promote
	if follow.GetString("following") != userID {
		return c.JSON(403, map[string]string{"error": "permission denied"})
	}

	follow.Set("role", "admin")

	if err := app.Save(follow); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Notify
	followerID := follow.GetString("follower")
	userChannelManager.SendToSSE(followerID, "promoted_to_admin", map[string]interface{}{
		"following_id": userID,
	}, "")

	return c.JSON(200, map[string]interface{}{"success": true})
}

// ==================== ROOM HANDLERS ====================

func handleCreateRoomWithSettings(c *core.RequestEvent) error {
	app := c.App
	ownerID := c.Get("userID").(string)

	var req struct {
		RoomType        string  `json:"room_type"`
		Name            string  `json:"name"`
		Description     string  `json:"description"`
		IsPublic        bool    `json:"is_public"`
		MaxParticipants int     `json:"max_participants"`
		JoinType        string  `json:"join_type"`
		Price           float64 `json:"price"`
		PeriodDays      int     `json:"period_days"`
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Create room
	r, err := app.FindCollectionByNameOrId("rooms")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	room := core.NewRecord(r)
	room.Set("roomType", req.RoomType)
	room.Set("name", req.Name)
	room.Set("description", req.Description)
	room.Set("owner", ownerID)
	room.Set("isPublic", req.IsPublic)
	room.Set("maxParticipants", req.MaxParticipants)
	room.Set("joinType", req.JoinType)
	room.Set("price", req.Price)
	room.Set("periodDays", req.PeriodDays)
	room.Set("isActive", true)

	if err := app.Save(room); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Add owner as member
	r, err = app.FindCollectionByNameOrId("roomMembers")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	member := core.NewRecord(r)
	member.Set("room", room.Id)
	member.Set("user", ownerID)
	member.Set("role", "owner")
	member.Set("status", "active")
	member.Set("joinedAt", time.Now())

	if err := app.Save(member); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"room_id": room.Id,
		"room":    recordToMap(room),
	})
}

func handleJoinRoomRequest(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	roomID := c.Request.PathValue("roomId")

	room, err := app.FindRecordById("rooms", roomID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "room not found"})
	}

	if !room.GetBool("isActive") {
		return c.JSON(403, map[string]string{"error": "room is not active"})
	}

	// Check if already member
	existing, _ := app.FindFirstRecordByFilter("roomMembers",
		fmt.Sprintf("room = '%s' && user = '%s'", roomID, userID))
	if existing != nil {
		return c.JSON(400, map[string]string{"error": "already a member"})
	}

	// Check max participants
	memberCount, _ := app.FindRecordsByFilter("roomMembers",
		fmt.Sprintf("room = '%s' && status = 'active'", roomID), "", 9999, 0)
	if len(memberCount) >= room.GetInt("maxParticipants") {
		return c.JSON(403, map[string]string{"error": "room is full"})
	}

	joinType := room.GetString("joinType")
	price := room.GetFloat("price")
	periodDays := room.GetInt("periodDays")

	r, err := app.FindCollectionByNameOrId("roomMembers")
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	member := core.NewRecord(r)
	member.Set("room", roomID)
	member.Set("user", userID)
	member.Set("role", "participant")

	var status string
	var expiresAt *time.Time

	switch joinType {
	case "free":
		status = "active"
		member.Set("joinedAt", time.Now())

	case "require_approval":
		status = "pending"

	case "paid_period":
		if price <= 0 {
			return c.JSON(400, map[string]string{"error": "invalid price"})
		}
		status = "active"
		expires := time.Now().AddDate(0, 0, periodDays)
		expiresAt = &expires
		member.Set("paidAmount", price)
		member.Set("paidAt", time.Now())
		member.Set("joinedAt", time.Now())

		err := createPaymentOperation(app, userID, price,
			fmt.Sprintf("Room access: %s (%d days)", room.GetString("name"), periodDays), &roomID)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}

	case "paid_lifetime":
		if price <= 0 {
			return c.JSON(400, map[string]string{"error": "invalid price"})
		}
		status = "active"
		member.Set("paidAmount", price)
		member.Set("paidAt", time.Now())
		member.Set("joinedAt", time.Now())

		err := createPaymentOperation(app, userID, price,
			fmt.Sprintf("Room lifetime access: %s", room.GetString("name")), &roomID)
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
	}

	member.Set("status", status)
	if expiresAt != nil {
		member.Set("expiresAt", *expiresAt)
	}

	if err := app.Save(member); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Notify owner/admins if pending
	if status == "pending" {
		notifyRoomAdmins(app, roomID, "join_request", map[string]interface{}{
			"user_id": userID,
			"room_id": roomID,
		})
	}

	return c.JSON(200, map[string]interface{}{
		"success":   true,
		"member_id": member.Id,
		"status":    status,
	})
}

func handleApproveRoomMember(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	memberID := c.Request.PathValue("memberId")

	member, err := app.FindRecordById("roomMembers", memberID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "member not found"})
	}

	roomID := member.GetString("room")

	// Check if user is owner or admin
	if !IsRoomOwnerOrAdmin(app, roomID, userID) {
		return c.JSON(403, map[string]string{"error": "permission denied"})
	}

	member.Set("status", "active")
	member.Set("approvedBy", userID)
	member.Set("joinedAt", time.Now())

	if err := app.Save(member); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Notify member
	targetUserID := member.GetString("user")
	userChannelManager.SendToSSE(targetUserID, "room_join_approved", map[string]interface{}{
		"room_id": roomID,
	}, "")

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleRejectRoomMember(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	memberID := c.Request.PathValue("memberId")

	member, err := app.FindRecordById("roomMembers", memberID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "member not found"})
	}

	roomID := member.GetString("room")

	if !IsRoomOwnerOrAdmin(app, roomID, userID) {
		return c.JSON(403, map[string]string{"error": "permission denied"})
	}

	member.Set("status", "rejected")

	if err := app.Save(member); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handlePromoteToRoomAdmin(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	memberID := c.Request.PathValue("memberId")

	member, err := app.FindRecordById("roomMembers", memberID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "member not found"})
	}

	roomID := member.GetString("room")
	room, _ := app.FindRecordById("rooms", roomID)

	// Only owner can promote
	if room.GetString("owner") != userID {
		return c.JSON(403, map[string]string{"error": "only owner can promote"})
	}

	member.Set("role", "admin")

	if err := app.Save(member); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Notify
	targetUserID := member.GetString("user")
	userChannelManager.SendToSSE(targetUserID, "promoted_to_room_admin", map[string]interface{}{
		"room_id": roomID,
	}, "")

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleDemoteRoomAdmin(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	memberID := c.Request.PathValue("memberId")

	member, err := app.FindRecordById("roomMembers", memberID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "member not found"})
	}

	roomID := member.GetString("room")
	room, _ := app.FindRecordById("rooms", roomID)

	if room.GetString("owner") != userID {
		return c.JSON(403, map[string]string{"error": "only owner can demote"})
	}

	member.Set("role", "participant")

	if err := app.Save(member); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleBanRoomMember(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	memberID := c.Request.PathValue("memberId")

	member, err := app.FindRecordById("roomMembers", memberID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "member not found"})
	}

	roomID := member.GetString("room")

	if !IsRoomOwnerOrAdmin(app, roomID, userID) {
		return c.JSON(403, map[string]string{"error": "permission denied"})
	}

	// Cannot ban owner
	if member.GetString("role") == "owner" {
		return c.JSON(403, map[string]string{"error": "cannot ban owner"})
	}

	member.Set("status", "banned")

	if err := app.Save(member); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Kick from active WebRTC room
	roomsMutex.RLock()
	room, exists := rooms[roomID]
	roomsMutex.RUnlock()

	if exists {
		targetUserID := member.GetString("user")
		room.mu.RLock()
		for _, p := range room.Participants {
			if p.UserID == targetUserID {
				room.mu.RUnlock()
				room.RemoveParticipant(p.ID)
				break
			}
		}
		room.mu.RUnlock()
	}

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleLeaveRoom(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	roomID := c.Request.PathValue("roomId")

	member, err := app.FindFirstRecordByFilter("roomMembers",
		fmt.Sprintf("room = '%s' && user = '%s'", roomID, userID))

	if err != nil {
		return c.JSON(404, map[string]string{"error": "not a member"})
	}

	// Owner cannot leave (must transfer ownership first)
	if member.GetString("role") == "owner" {
		return c.JSON(403, map[string]string{"error": "owner cannot leave, transfer ownership first"})
	}

	if err := app.Delete(member); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleGetRoomMembers(c *core.RequestEvent) error {
	app := c.App
	roomID := c.Request.PathValue("roomId")
	status := c.Request.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}

	filter := fmt.Sprintf("room = '%s' && status = '%s'", roomID, status)
	members, err := app.FindRecordsByFilter("roomMembers", filter, "-created", 1000, 0)

	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	result := make([]map[string]interface{}, len(members))
	for i, member := range members {
		result[i] = recordToMap(member)
	}

	return c.JSON(200, map[string]interface{}{
		"members": result,
		"count":   len(result),
	})
}

func handleGetMyRooms(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)

	members, err := app.FindRecordsByFilter("roomMembers",
		fmt.Sprintf("user = '%s' && status = 'active'", userID), "-created", 100, 0)

	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	result := make([]map[string]interface{}, len(members))
	for i, member := range members {
		roomID := member.GetString("room")
		room, _ := app.FindRecordById("rooms", roomID)

		result[i] = map[string]interface{}{
			"member": recordToMap(member),
			"room":   recordToMap(room),
		}
	}

	return c.JSON(200, map[string]interface{}{
		"rooms": result,
		"count": len(result),
	})
}

func handleTransferRoomOwnership(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	roomID := c.Request.PathValue("roomId")

	var req struct {
		NewOwnerID string `json:"new_owner_id"`
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	room, err := app.FindRecordById("rooms", roomID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "room not found"})
	}

	// Must be current owner
	if room.GetString("owner") != userID {
		return c.JSON(403, map[string]string{"error": "only owner can transfer"})
	}

	// New owner must be a member
	newOwnerMember, err := app.FindFirstRecordByFilter("roomMembers",
		fmt.Sprintf("room = '%s' && user = '%s' && status = 'active'", roomID, req.NewOwnerID))

	if err != nil {
		return c.JSON(400, map[string]string{"error": "new owner must be an active member"})
	}

	// Update room owner
	room.Set("owner", req.NewOwnerID)
	if err := app.Save(room); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Update old owner to admin
	oldOwnerMember, _ := app.FindFirstRecordByFilter("roomMembers",
		fmt.Sprintf("room = '%s' && user = '%s'", roomID, userID))
	if oldOwnerMember != nil {
		oldOwnerMember.Set("role", "admin")
		app.Save(oldOwnerMember)
	}

	// Update new owner role
	newOwnerMember.Set("role", "owner")
	app.Save(newOwnerMember)

	// Notify
	userChannelManager.SendToSSE(req.NewOwnerID, "room_ownership_transferred", map[string]interface{}{
		"room_id": roomID,
	}, "")

	return c.JSON(200, map[string]interface{}{"success": true})
}

func handleUpdateRoomSettings(c *core.RequestEvent) error {
	app := c.App
	userID := c.Get("userID").(string)
	roomID := c.Request.PathValue("roomId")

	room, err := app.FindRecordById("rooms", roomID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "room not found"})
	}

	// Must be owner
	if room.GetString("owner") != userID {
		return c.JSON(403, map[string]string{"error": "only owner can update settings"})
	}

	var req struct {
		Name            *string  `json:"name,omitempty"`
		Description     *string  `json:"description,omitempty"`
		IsPublic        *bool    `json:"is_public,omitempty"`
		MaxParticipants *int     `json:"max_participants,omitempty"`
		JoinType        *string  `json:"join_type,omitempty"`
		Price           *float64 `json:"price,omitempty"`
		PeriodDays      *int     `json:"period_days,omitempty"`
		IsActive        *bool    `json:"is_active,omitempty"`
	}

	if err := c.BindBody(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	if req.Name != nil {
		room.Set("name", *req.Name)
	}
	if req.Description != nil {
		room.Set("description", *req.Description)
	}
	if req.IsPublic != nil {
		room.Set("isPublic", *req.IsPublic)
	}
	if req.MaxParticipants != nil {
		room.Set("maxParticipants", *req.MaxParticipants)
	}
	if req.JoinType != nil {
		room.Set("joinType", *req.JoinType)
	}
	if req.Price != nil {
		room.Set("price", *req.Price)
	}
	if req.PeriodDays != nil {
		room.Set("periodDays", *req.PeriodDays)
	}
	if req.IsActive != nil {
		room.Set("isActive", *req.IsActive)
	}

	if err := app.Save(room); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, recordToMap(room))
}

// ==================== UTILITY FUNCTIONS ====================

func IsRoomOwnerOrAdmin(app core.App, roomID, userID string) bool {
	member, err := app.FindFirstRecordByFilter("roomMembers",
		fmt.Sprintf("room = '%s' && user = '%s' && status = 'active'", roomID, userID))

	if err != nil {
		return false
	}

	role := member.GetString("role")
	return role == "owner" || role == "admin"
}

func notifyRoomAdmins(app core.App, roomID string, eventType string, data map[string]interface{}) {
	admins, _ := app.FindRecordsByFilter("roomMembers",
		fmt.Sprintf("room = '%s' && (role = 'owner' || role = 'admin') && status = 'active'", roomID), "", 100, 0)

	for _, admin := range admins {
		adminUserID := admin.GetString("user")
		userChannelManager.SendToSSE(adminUserID, eventType, data, "")
		userChannelManager.SendToUserRoom(adminUserID, "notification", map[string]interface{}{
			"type": eventType,
			"data": data,
		}, "")
	}
}

func createPaymentOperation(app core.App, userID string, amount float64, description string, relatedID *string) error {
	r, err := app.FindCollectionByNameOrId("operations")
	if err != nil {
		return err
	}
	operation := core.NewRecord(r)
	operation.Set("user", userID)
	operation.Set("montant", -amount)
	operation.Set("operation", "cashout")
	operation.Set("desc", description)
	operation.Set("status", "paye")

	if relatedID != nil {
		// Could link to room or other entity
		metadata := map[string]interface{}{
			"related_id": *relatedID,
		}
		operation.Set("metadata", metadata)
	}

	return app.Save(operation)
}

// Check and expire subscriptions (run periodically)
func checkExpiredSubscriptions(app core.App) {
	now := time.Now().Format(time.RFC3339)

	// Expire follows
	expiredFollows, _ := app.FindRecordsByFilter("follows",
		fmt.Sprintf("status = 'active' && expiresAt != '' && expiresAt <= '%s'", now), "", 1000, 0)

	for _, follow := range expiredFollows {
		follow.Set("status", "expired")
		app.Save(follow)

		// Notify
		followerID := follow.GetString("follower")
		followingID := follow.GetString("following")
		userChannelManager.SendToSSE(followerID, "follow_expired", map[string]interface{}{
			"following_id": followingID,
		}, "")
	}

	// Expire room memberships
	expiredMembers, _ := app.FindRecordsByFilter("roomMembers",
		fmt.Sprintf("status = 'active' && expiresAt != '' && expiresAt <= '%s'", now), "", 1000, 0)

	for _, member := range expiredMembers {
		member.Set("status", "expired")
		app.Save(member)

		// Notify
		userID := member.GetString("user")
		roomID := member.GetString("room")
		userChannelManager.SendToSSE(userID, "room_membership_expired", map[string]interface{}{
			"room_id": roomID,
		}, "")
	}

	log.Printf("Expired %d follows and %d room memberships", len(expiredFollows), len(expiredMembers))
}
