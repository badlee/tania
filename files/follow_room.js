// ==================== CLIENT COMPLET ====================

export default class FollowRoomClient {
  constructor(serverUrl, authToken) {
    this.serverUrl = serverUrl;
    this.authToken = authToken;
  }

  // ========== FOLLOW ==========

  async updateFollowSettings(settings) {
    const res = await fetch(`${this.serverUrl}/api/user/follow-settings`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(settings)
    });
    return res.json();
  }

  async followUser(userId) {
    const res = await fetch(`${this.serverUrl}/api/users/${userId}/follow`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async unfollowUser(userId) {
    const res = await fetch(`${this.serverUrl}/api/users/${userId}/follow`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async approveFollow(followId) {
    const res = await fetch(`${this.serverUrl}/api/follows/${followId}/approve`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async getFollowers(userId, status = 'active') {
    const res = await fetch(`${this.serverUrl}/api/users/${userId}/followers?status=${status}`, {
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async promoteFollowerToAdmin(followId) {
    const res = await fetch(`${this.serverUrl}/api/follows/${followId}/promote`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  // ========== ROOMS ==========

  async createRoom(roomData) {
    const res = await fetch(`${this.serverUrl}/api/rooms/create`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(roomData)
    });
    return res.json();
  }

  async joinRoom(roomId) {
    const res = await fetch(`${this.serverUrl}/api/rooms/${roomId}/join-request`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async leaveRoom(roomId) {
    const res = await fetch(`${this.serverUrl}/api/rooms/${roomId}/leave`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async getRoomMembers(roomId, status = 'active') {
    const res = await fetch(`${this.serverUrl}/api/rooms/${roomId}/members?status=${status}`, {
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async promoteToRoomAdmin(memberId) {
    const res = await fetch(`${this.serverUrl}/api/room-members/${memberId}/promote`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async banRoomMember(memberId) {
    const res = await fetch(`${this.serverUrl}/api/room-members/${memberId}/ban`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }

  async transferRoomOwnership(roomId, newOwnerId) {
    const res = await fetch(`${this.serverUrl}/api/rooms/${roomId}/transfer-ownership`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ new_owner_id: newOwnerId })
    });
    return res.json();
  }

  async updateRoomSettings(roomId, settings) {
    const res = await fetch(`${this.serverUrl}/api/rooms/${roomId}/settings`, {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(settings)
    });
    return res.json();
  }

  async getMyRooms() {
    const res = await fetch(`${this.serverUrl}/api/user/rooms`, {
      headers: { 'Authorization': `Bearer ${this.authToken}` }
    });
    return res.json();
  }
}