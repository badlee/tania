import FollowRoomClient from "./follow_room";
// ==================== EXEMPLES FOLLOW/FOLLOWERS ====================

// ==================== 1. CONFIGURATION DES PARAMÈTRES DE FOLLOW ====================

// Profil gratuit (tout le monde peut follow)
await fetch('/api/user/follow-settings', {
  method: 'PUT',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    follow_type: 'free',
    is_accepting_followers: true,
    description: 'Follow me for free content!'
  })
});

// Profil avec approbation
await fetch('/api/user/follow-settings', {
  method: 'PUT',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    follow_type: 'require_approval',
    is_accepting_followers: true,
    description: 'I approve follows manually'
  })
});

// Profil payant par période (abonnement mensuel)
await fetch('/api/user/follow-settings', {
  method: 'PUT',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    follow_type: 'paid_period',
    price: 9.99,
    period_days: 30,
    is_accepting_followers: true,
    description: 'Exclusive content - $9.99/month'
  })
});

// Profil payant à vie
await fetch('/api/user/follow-settings', {
  method: 'PUT',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    follow_type: 'paid_lifetime',
    price: 99.99,
    is_accepting_followers: true,
    description: 'Lifetime access - $99.99 one-time'
  })
});

// ==================== 2. FOLLOW UN UTILISATEUR ====================

// Follow gratuit
const followResponse = await fetch('/api/users/user_123/follow', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  }
});

const followData = await followResponse.json();
console.log('Follow status:', followData.status); // "active" ou "pending"

// ==================== 3. GÉRER LES DEMANDES DE FOLLOW ====================

// Récupérer les demandes en attente
const pendingFollows = await fetch('/api/users/me/followers?status=pending', {
  headers: { 'Authorization': 'Bearer TOKEN' }
});

const pending = await pendingFollows.json();
console.log('Pending requests:', pending.followers);

// Approuver une demande
await fetch('/api/follows/follow_123/approve', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// Rejeter une demande
await fetch('/api/follows/follow_123/reject', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// ==================== 4. PROMOUVOIR UN FOLLOWER EN ADMIN ====================

// Seul le compte suivi peut promouvoir ses followers
await fetch('/api/follows/follow_123/promote', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// L'admin peut maintenant:
// - Approuver/rejeter des demandes de follow
// - Gérer les followers
// - Modérer les commentaires

// ==================== 5. RÉCUPÉRER FOLLOWERS ET FOLLOWING ====================

// Mes followers
const followers = await fetch('/api/users/me/followers?status=active', {
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// Qui je suis
const following = await fetch('/api/users/me/following', {
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// ==================== 6. UNFOLLOW ====================

await fetch('/api/users/user_123/follow', {
  method: 'DELETE',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// ==================== EXEMPLES ROOM MANAGEMENT ====================

// ==================== 1. CRÉER UNE ROOM AVEC PARAMÈTRES ====================

// Room gratuite
const freeRoom = await fetch('/api/rooms/create', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    room_type: 'audio',
    name: 'Casual Voice Chat',
    description: 'Open voice chat for everyone',
    is_public: true,
    max_participants: 50,
    join_type: 'free'
  })
});

// Room avec approbation
const approvalRoom = await fetch('/api/rooms/create', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    room_type: 'video',
    name: 'Private Mastermind',
    description: 'Approved members only',
    is_public: false,
    max_participants: 10,
    join_type: 'require_approval'
  })
});

// Room payante mensuelle
const paidRoom = await fetch('/api/rooms/create', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    room_type: 'audio',
    name: 'VIP Podcast Studio',
    description: 'Monthly subscription for exclusive access',
    is_public: true,
    max_participants: 100,
    join_type: 'paid_period',
    price: 19.99,
    period_days: 30
  })
});

// Room à vie
const lifetimeRoom = await fetch('/api/rooms/create', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    room_type: 'video',
    name: 'Lifetime Coaching',
    description: 'One-time payment for lifetime access',
    is_public: true,
    max_participants: 20,
    join_type: 'paid_lifetime',
    price: 299.99
  })
});

// ==================== 2. REJOINDRE UNE ROOM ====================

const joinResponse = await fetch('/api/rooms/room_123/join-request', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  }
});

const joinData = await joinResponse.json();
console.log('Join status:', joinData.status); // "active", "pending"

// Si payant, le paiement est automatiquement traité
// et une opération est créée

// ==================== 3. GÉRER LES DEMANDES D'ADHÉSION ====================

// Récupérer les membres en attente (owner/admin)
const pendingMembers = await fetch('/api/rooms/room_123/members?status=pending', {
  headers: { 'Authorization': 'Bearer TOKEN' }
});

const pending = await pendingMembers.json();

// Approuver
await fetch('/api/room-members/member_123/approve', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// Rejeter
await fetch('/api/room-members/member_123/reject', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// ==================== 4. GESTION DES RÔLES ====================

// Promouvoir en admin (owner uniquement)
await fetch('/api/room-members/member_123/promote', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// L'admin peut maintenant:
// - Approuver/rejeter des membres
// - Bannir des membres
// - Modérer la room

// Rétrograder un admin (owner uniquement)
await fetch('/api/room-members/member_123/demote', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// ==================== 5. BANNIR UN MEMBRE ====================

// Owner ou admin peut bannir
await fetch('/api/room-members/member_123/ban', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// Le membre est immédiatement expulsé de la room WebRTC

// ==================== 6. TRANSFÉRER LA PROPRIÉTÉ ====================

// Seul l'owner peut transférer
await fetch('/api/rooms/room_123/transfer-ownership', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    new_owner_id: 'user_456'
  })
});

// L'ancien owner devient admin
// Le nouveau owner a tous les droits

// ==================== 7. METTRE À JOUR LES PARAMÈTRES ====================

await fetch('/api/rooms/room_123/settings', {
  method: 'PATCH',
  headers: {
    'Authorization': 'Bearer TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    name: 'New Room Name',
    max_participants: 200,
    join_type: 'paid_period',
    price: 29.99,
    period_days: 30,
    is_active: true
  })
});

// ==================== 8. QUITTER UNE ROOM ====================

await fetch('/api/rooms/room_123/leave', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer TOKEN' }
});

// L'owner ne peut pas quitter sans transférer d'abord

// ==================== 9. RÉCUPÉRER MES ROOMS ====================

const myRooms = await fetch('/api/user/rooms', {
  headers: { 'Authorization': 'Bearer TOKEN' }
});

const rooms = await myRooms.json();
console.log('My rooms:', rooms.rooms);

// Chaque entrée contient:
// - member: info du membre (role, status, expires_at)
// - room: info de la room


// ==================== UTILISATION COMPLÈTE ====================

const client = new FollowRoomClient('http://localhost:8090', 'TOKEN');

// Configuration du profil
await client.updateFollowSettings({
  follow_type: 'paid_period',
  price: 9.99,
  period_days: 30,
  description: 'Premium content',
  is_accepting_followers: true
});

// Créer une room payante
const room = await client.createRoom({
  room_type: 'audio',
  name: 'VIP Lounge',
  description: 'Exclusive audio room',
  is_public: true,
  max_participants: 50,
  join_type: 'paid_period',
  price: 19.99,
  period_days: 30
});

console.log('Room created:', room.room_id);

// Rejoindre une room
const join = await client.joinRoom(room.room_id);
console.log('Join status:', join.status);

// Gérer les membres
const members = await client.getRoomMembers(room.room_id);
console.log('Members:', members.count);

// Promouvoir un membre
if (members.members.length > 0) {
  await client.promoteToRoomAdmin(members.members[0].id);
}

// Follow un utilisateur
await client.followUser('user_123');

// Voir mes followers
const followers = await client.getFollowers('me');
console.log('Followers:', followers.count);

// Mes rooms
const myRooms = await client.getMyRooms();
console.log('My rooms:', myRooms.count);
