// ==================== EXEMPLES GÉOLOCALISATION ====================

// ==================== HOOK: pb_hooks/location-tracker.js ====================
// Suivi en temps réel des localisations et déclenchement d'actions

const logger = require("logger");
const cache = require("cache");

function main() {
  logger.info("Location tracker initialized");

  // Écouter les mises à jour de localisation
  pubsub.subscribe("location_updates", function(eventData) {
    const event = utils.jsonDecode(eventData);
    handleLocationUpdate(event);
  });

  // Écouter les événements de géofences
  pubsub.subscribe("geo_events", function(eventData) {
    const event = utils.jsonDecode(eventData);
    handleGeoEvent(event);
  });

  logger.info("Listening for location updates and geo events...");
}

function handleLocationUpdate(event) {
  const userId = event.user_id;
  const location = event.location;
  const presence = event.presence;

  logger.info(`📍 User ${userId} location updated: (${location.point.lat}, ${location.point.lng}), presence: ${presence}`);

  // Enregistrer dans l'historique
  db.create("locationHistory", {
    user: userId,
    location: utils.jsonEncode(location),
    presence: presence,
    accuracy: location.accuracy
  });

  // Vérifier les utilisateurs à proximité
  const nearby = location.findNearby(location.point.lat, location.point.lng, 100); // 100m
  
  if (nearby.length > 0) {
    logger.info(`👥 Found ${nearby.length} users nearby`);
    
    // Notifier l'utilisateur
    pubsub.publish("notifications", {
      type: "nearby_users",
      user_id: userId,
      count: nearby.length,
      users: nearby
    });
  }

  // Mettre en cache la dernière position
  cache.set(`last_location:${userId}`, {
    lat: location.point.lat,
    lng: location.point.lng,
    timestamp: Date.now()
  }, 300); // 5 minutes
}

function handleGeoEvent(event) {
  logger.info(`🎯 Geo event: ${event.type} - User ${event.user_id}`);

  if (event.type === "user_entered") {
    handleUserEnteredFence(event);
  } else if (event.type === "user_exited") {
    handleUserExitedFence(event);
  }
}

function handleUserEnteredFence(event) {
  const fence = event.fence;
  logger.info(`User ${event.user_id} entered fence: ${fence.name}`);

  // Créer une notification personnalisée
  db.create("notifications", {
    user: event.user_id,
    type: "geofence_enter",
    title: `Bienvenue à ${fence.name}!`,
    message: fence.metadata.welcome_message || "Vous êtes entré dans une zone spéciale",
    data: utils.jsonEncode({
      fence_id: fence.id,
      fence_name: fence.name,
      location: event.location
    }),
    isRead: false
  });

  // Statistiques
  const key = `fence_entries:${fence.id}`;
  const count = cache.get(key) || 0;
  cache.set(key, count + 1, 86400); // 24h
}

function handleUserExitedFence(event) {
  logger.info(`User ${event.user_id} exited fence: ${event.fence_id}`);
  
  // Enregistrer la durée du séjour
  const entryKey = `fence_entry_time:${event.user_id}:${event.fence_id}`;
  const entryTime = cache.get(entryKey);
  
  if (entryTime) {
    const duration = Date.now() - entryTime;
    logger.info(`User stayed for ${Math.round(duration / 1000)} seconds`);
    
    cache.del(entryKey);
  }
}

main();

// ==================== HOOK: pb_hooks/proximity-chat.js ====================
// Système de chat de proximité automatique

const logger = require("logger");
const cache = require("cache");

const PROXIMITY_THRESHOLD = 50; // 50 mètres

function main() {
  logger.info("Proximity chat system initialized");

  // Vérifier les proximités toutes les 30 secondes
  cron.schedule(30, function() {
    checkProximityMatches();
  });
}

function checkProximityMatches() {
  const onlineUsers = location.getUsersByPresence("online");
  
  if (onlineUsers.length < 2) {
    return;
  }

  logger.info(`Checking proximity for ${onlineUsers.length} online users`);

  for (let i = 0; i < onlineUsers.length; i++) {
    for (let j = i + 1; j < onlineUsers.length; j++) {
      const user1 = onlineUsers[i];
      const user2 = onlineUsers[j];

      const distance = location.distance(
        user1.lat, user1.lng,
        user2.lat, user2.lng
      );

      if (distance <= PROXIMITY_THRESHOLD) {
        handleProximityMatch(user1.user_id, user2.user_id, distance);
      }
    }
  }
}

function handleProximityMatch(userId1, userId2, distance) {
  const matchKey = `proximity_match:${userId1}:${userId2}`;
  
  // Éviter les notifications répétées
  if (cache.has(matchKey)) {
    return;
  }

  logger.info(`👫 Proximity match: ${userId1} and ${userId2} (${Math.round(distance)}m apart)`);

  // Créer une room de chat automatique
  const roomId = `proximity_${Date.now()}_${userId1}_${userId2}`;
  
  db.create("rooms", {
    roomType: "data",
    name: `Chat de proximité (${Math.round(distance)}m)`,
    creator: userId1,
    isPublic: false,
    maxParticipants: 2
  });

  // Inviter les deux utilisateurs
  pubsub.publish("chat_invites", {
    type: "proximity_chat",
    room_id: roomId,
    user_ids: [userId1, userId2],
    distance: Math.round(distance),
    message: `Quelqu'un est proche de vous! Voulez-vous discuter?`
  });

  // Marquer pour éviter répétition (5 minutes)
  cache.set(matchKey, true, 300);
  cache.set(`proximity_match:${userId2}:${userId1}`, true, 300);
}

main();

// ==================== HOOK: pb_hooks/geofence-ads.js ====================
// Publicité ciblée basée sur la localisation

const logger = require("logger");
const cache = require("cache");

function main() {
  logger.info("Geofence ads system initialized");

  // Écouter les entrées dans les zones
  pubsub.subscribe("geo_events", function(eventData) {
    const event = utils.jsonDecode(eventData);
    
    if (event.type === "user_entered") {
      showTargetedAd(event);
    }
  });
}

function showTargetedAd(event) {
  const fence = event.fence;
  
  // Vérifier si la fence a des données publicitaires
  if (!fence.metadata || !fence.metadata.ad_data) {
    return;
  }

  const userId = event.user_id;
  const adKey = `ad_shown:${userId}:${fence.id}`;

  // Ne pas montrer la même pub plusieurs fois par jour
  if (cache.has(adKey)) {
    return;
  }

  logger.info(`📢 Showing ad to user ${userId} in fence ${fence.name}`);

  const adData = fence.metadata.ad_data;

  // Créer une notification publicitaire
  db.create("notifications", {
    user: userId,
    type: "advertisement",
    title: adData.title || "Offre spéciale près de vous!",
    message: adData.message,
    data: utils.jsonEncode({
      ad_id: adData.id,
      fence_id: fence.id,
      image_url: adData.image_url,
      action_url: adData.action_url,
      discount: adData.discount
    }),
    isRead: false
  });

  // Enregistrer l'impression
  db.create("adImpressions", {
    user: userId,
    adId: adData.id,
    fenceId: fence.id,
    location: utils.jsonEncode(event.location)
  });

  // Marquer comme montré (24h)
  cache.set(adKey, true, 86400);

  // Statistiques
  const statsKey = `ad_impressions:${adData.id}`;
  const impressions = cache.get(statsKey) || 0;
  cache.set(statsKey, impressions + 1, 86400);

  logger.info(`Ad impressions for ${adData.id}: ${impressions + 1}`);
}

main();

// ==================== HOOK: pb_hooks/emergency-alert.js ====================
// Système d'alerte d'urgence géolocalisé

const logger = require("logger");

function main() {
  logger.info("Emergency alert system initialized");

  // Écouter les événements d'urgence via pubsub
  pubsub.subscribe("emergency_alerts", function(eventData) {
    const alert = utils.jsonDecode(eventData);
    handleEmergencyAlert(alert);
  });
}

function handleEmergencyAlert(alert) {
  logger.warn(`🚨 EMERGENCY ALERT: ${alert.type} at (${alert.lat}, ${alert.lng})`);

  // Trouver tous les utilisateurs dans un rayon de 5km
  const affectedUsers = location.findNearby(alert.lat, alert.lng, 5000);

  logger.warn(`Found ${affectedUsers.length} users in danger zone`);

  // Notifier tous les utilisateurs affectés
  for (const user of affectedUsers) {
    const distance = Math.round(user.distance);
    
    // Créer notification d'urgence
    db.create("notifications", {
      user: user.user_id,
      type: "emergency",
      title: `🚨 ALERTE: ${alert.type}`,
      message: `Incident à ${distance}m de votre position. ${alert.message}`,
      data: utils.jsonEncode({
        alert_type: alert.type,
        distance: distance,
        severity: alert.severity,
        instructions: alert.instructions,
        lat: alert.lat,
        lng: alert.lng
      }),
      isRead: false,
      priority: "high"
    });

    // Publier notification push
    pubsub.publish("push_notifications", {
      user_id: user.user_id,
      title: `🚨 ALERTE: ${alert.type}`,
      message: `Incident à ${distance}m`,
      priority: "high",
      sound: "emergency"
    });

    logger.warn(`Emergency notification sent to user ${user.user_id}`);
  }

  // Enregistrer l'alerte
  db.create("emergencyAlerts", {
    type: alert.type,
    location: utils.jsonEncode({ lat: alert.lat, lng: alert.lng }),
    message: alert.message,
    severity: alert.severity,
    affectedUsers: affectedUsers.length
  });
}

main();

// ==================== HOOK: pb_hooks/presence-manager.js ====================
// Gestion de la présence des utilisateurs

const logger = require("logger");
const cache = require("cache");

function main() {
  logger.info("Presence manager initialized");

  // Vérifier les présences toutes les minutes
  cron.schedule(60, function() {
    updatePresenceStatus();
  });

  // Écouter les mises à jour de localisation
  pubsub.subscribe("location_updates", function(eventData) {
    const event = utils.jsonDecode(eventData);
    handlePresenceUpdate(event);
  });
}

function updatePresenceStatus() {
  // Récupérer tous les utilisateurs
  const users = db.findAll("users", "", "-lastSeen", 1000);

  let onlineCount = 0;
  let awayCount = 0;
  let offlineCount = 0;

  for (const user of users) {
    if (!user.lastSeen) continue;

    const lastSeenTime = new Date(user.lastSeen).getTime();
    const now = Date.now();
    const minutesSinceLastSeen = (now - lastSeenTime) / 1000 / 60;

    let newPresence = user.presence;

    if (minutesSinceLastSeen > 30) {
      newPresence = "offline";
      offlineCount++;
    } else if (minutesSinceLastSeen > 5) {
      newPresence = "away";
      awayCount++;
    } else {
      onlineCount++;
    }

    // Mettre à jour si changement
    if (newPresence !== user.presence) {
      db.update("users", user.id, { presence: newPresence });
      
      // Publier l'événement
      pubsub.publish("presence_changes", {
        user_id: user.id,
        old_presence: user.presence,
        new_presence: newPresence
      });
    }
  }

  logger.info(`Presence stats: ${onlineCount} online, ${awayCount} away, ${offlineCount} offline`);

  // Mettre en cache les stats
  cache.set("presence_stats", {
    online: onlineCount,
    away: awayCount,
    offline: offlineCount,
    total: users.length,
    timestamp: Date.now()
  }, 60);
}

function handlePresenceUpdate(event) {
  const userId = event.user_id;
  const presence = event.presence;

  logger.info(`User ${userId} presence: ${presence}`);

  // Notifier les amis/contacts
  const friendships = db.findAll("friendships", `user1 = '${userId}' || user2 = '${userId}'`);

  for (const friendship of friendships) {
    const friendId = friendship.user1 === userId ? friendship.user2 : friendship.user1;
    
    pubsub.publish("notifications", {
      type: "friend_presence",
      user_id: friendId,
      friend_id: userId,
      presence: presence
    });
  }
}

main();

// ==================== EXEMPLE CLIENT: Utilisation de l'API ====================

/*
// 1. Mettre à jour sa localisation
fetch('/api/location/update', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    location: {
      point: { lat: 48.8566, lng: 2.3522 },
      accuracy: 10,
      altitude: 35,
      speed: 0,
      heading: 0
    },
    presence: 'online'
  })
});

// 2. Trouver des utilisateurs à proximité
fetch('/api/location/nearby', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    point: { lat: 48.8566, lng: 2.3522 },
    radius: 1000 // 1km
  })
});

// 3. Créer une geofence
fetch('/api/geofences', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    name: "Zone commerciale",
    geometry: {
      type: "Circle",
      coordinates: [2.3522, 48.8566], // [lng, lat]
      radius: 500
    },
    actions: ["notification", "ads"],
    trigger_type: "enter",
    metadata: {
      notification_message: "Bienvenue dans la zone commerciale!",
      ad_data: {
        id: "ad_123",
        title: "20% de réduction!",
        message: "Offre spéciale aujourd'hui"
      }
    }
  })
});

// 4. Notifier tous les utilisateurs dans une zone
fetch('/api/location/notify-zone', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    point: { lat: 48.8566, lng: 2.3522 },
    radius: 2000,
    title: "Événement spécial",
    message: "Concert gratuit ce soir au parc!"
  })
});

// 5. S'abonner aux événements de localisation (SSE)
const eventSource = new EventSource('/api/events/location_updates');
eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Location update:', data);
};

// 6. Broadcaster sa position dans une room WebRTC
fetch('/api/rooms/room_123/broadcast-location', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    location: {
      point: { lat: 48.8566, lng: 2.3522 },
      accuracy: 10
    }
  })
});
*/
