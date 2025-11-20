// Tests pour le client JavaScript avec Bun
// Run with: bun test --timeout 10000

import { describe, test, expect, beforeEach, mock, spyOn } from "bun:test";
import UserChannelsClient from "../files/user_channels_client"
import FollowRoomClient from "../files/follow_room";

// Mock global objects
global.EventSource = class EventSource {
  constructor(url) {
    this.url = url;
    this.onopen = null;
    this.onmessage = null;
    this.onerror = null;
  }
  close() {}
};

global.RTCPeerConnection = class RTCPeerConnection {
  constructor(config) {
    this.config = config;
    this.localDescription = null;
    this.remoteDescription = null;
  }
  createDataChannel(label) {
    return {
      label,
      readyState: 'open',
      send: mock(() => {}),
      onopen: null,
      onmessage: null,
      onclose: null,
      onerror: null
    };
  }
  async createOffer() {
    return { type: 'offer', sdp: 'mock_sdp' };
  }
  async createAnswer() {
    return { type: 'answer', sdp: 'mock_sdp' };
  }
  async setLocalDescription(desc) {
    this.localDescription = desc;
  }
  async setRemoteDescription(desc) {
    this.remoteDescription = desc;
  }
  close() {}
};

// Mock fetch
global.fetch = mock((url, options) => {
  return Promise.resolve({
    ok: true,
    status: 200,
    json: async () => ({ success: true })
  });
});

// // Import clients (adjust paths as needed)
// class UserChannelsClient {
//   constructor(serverUrl, authToken, reconnectionTTL) {
//     this.serverUrl = serverUrl;
//     this.authToken = authToken;
//     this.sseConnection = null;
//     this.sseHandlers = new Map();
//     this.userRoom = null;
//     this.dataChannel = null;
//     this.peerConnection = null;
//     this.requestCallbacks = new Map();
//     this.requestIdCounter = 0;
//     this.reconnectionTTL = reconnectionTTL ?? 4e3;
//     if(this.reconnectionTTL < 0){
//       this.reconnectionTTL = 0
//     }
//     this.reconnectionTimeToWait = this.reconnectionTTL;

//   }

//   connectSSE() {
//     const url = `${this.serverUrl}/api/user/sse?token=${this.authToken}`;
//     this.sseConnection = new EventSource(url);
    
//     this.sseConnection.onopen = () => {
//       this.reconnectionTTL = this.reconnectionTimeToWait;
//       console.log('\u2705 SSE Channel connected');
//     };
    
//     this.sseConnection.onmessage = (event) => {
//       const data = JSON.parse(event.data);
//       this.handleSSEMessage(data);
//     };
    
//     this.sseConnection.onerror = (error) => {
//       console.error('\u274c SSE error:', error);
//       if(this.reconnectionTTL > 0){
//         setTimeout(() => {
//           this.sseConnection.close()
//           this.connectSSE()
//         }, this.reconnectionTTL);
//         this.reconnectionTTL += this.reconnectionTTL * 0.05;
//       }
//     };
//   }

//   handleSSEMessage(message) {
//     if (message.request_id && this.requestCallbacks.has(message.request_id)) {
//       const callback = this.requestCallbacks.get(message.request_id);
//       callback(null, message.data);
//       this.requestCallbacks.delete(message.request_id);
//       return;
//     }

//     const handler = this.sseHandlers.get(message.type);
//     if (handler) {
//       handler(message.data);
//     }
//   }

//   onSSE(type, handler) {
//     this.sseHandlers.set(type, handler);
//   }

//   disconnectSSE() {
//     if (this.sseConnection) {
//       this.sseConnection.close();
//       this.sseConnection = null;
//     }
//   }

//   async connectUserRoom() {
//     this.peerConnection = new RTCPeerConnection({
//       iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
//     });

//     const response = await fetch(`${this.serverUrl}/api/user/room/connect`, {
//       method: 'POST',
//       headers: {
//         'Authorization': `Bearer ${this.authToken}`,
//         'Content-Type': 'application/json'
//       }
//     });

//     const { sdp } = await response.json();
//     await this.peerConnection.setRemoteDescription(sdp);

//     const answer = await this.peerConnection.createAnswer();
//     await this.peerConnection.setLocalDescription(answer);

//     await fetch(`${this.serverUrl}/api/user/room/answer`, {
//       method: 'POST',
//       headers: {
//         'Authorization': `Bearer ${this.authToken}`,
//         'Content-Type': 'application/json'
//       },
//       body: JSON.stringify(answer)
//     });

//     this.dataChannel = this.peerConnection.createDataChannel('api');
//     this.setupDataChannel();
//   }

//   setupDataChannel() {
//     this.dataChannel.onmessage = async (event) => {
//       const data = JSON.parse(event.data);
//       this.handleRoomMessage(data);
//     };
//   }

//   handleRoomMessage(message) {
//     if (message.request_id && this.requestCallbacks.has(message.request_id)) {
//       const callback = this.requestCallbacks.get(message.request_id);
//       callback(null, message.data);
//       this.requestCallbacks.delete(message.request_id);
//     }
//   }

//   async callAPI(method, endpoint, body = null, query = null) {
//     if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
//       throw new Error('User room not connected');
//     }

//     const requestId = `req_${++this.requestIdCounter}_${Date.now()}`;
//     const request = {
//       request_id: requestId,
//       method: method,
//       endpoint: endpoint,
//       body: body,
//       query: query
//     };

//     this.dataChannel.send(JSON.stringify(request));

//     return new Promise((resolve, reject) => {
//       const timeout = setTimeout(() => {
//         this.requestCallbacks.delete(requestId);
//         reject(new Error('Request timeout'));
//       }, 30000);

//       this.requestCallbacks.set(requestId, (error, data) => {
//         clearTimeout(timeout);
//         if (error) {
//           reject(error);
//         } else {
//           resolve(data);
//         }
//       });
//     });
//   }

//   async updateLocation(lat, lng, accuracy = 10, presence = 'online') {
//     return this.callAPI('POST', '/location/update', {
//       location: {
//         point: { lat, lng },
//         accuracy: accuracy,
//         altitude: 0,
//         speed: 0,
//         heading: 0
//       },
//       presence: presence
//     });
//   }

//   async findNearby(lat, lng, radius) {
//     return this.callAPI('POST', '/location/nearby', {
//       point: { lat, lng },
//       radius: radius
//     });
//   }

//   async createPost(content, type = 'html', isPublic = true) {
//     return this.callAPI('POST', '/posts', {
//       content: content,
//       type: type,
//       isPublic: isPublic
//     });
//   }

//   async likePost(postId, reaction = 'like') {
//     return this.callAPI('POST', '/posts/like', {
//       post_id: postId,
//       reaction: reaction
//     });
//   }

//   async commentPost(postId, content) {
//     return this.callAPI('POST', '/posts/comment', {
//       post_id: postId,
//       content: content
//     });
//   }

//   async getArticles(filter = 'quantite > 0') {
//     return this.callAPI('GET', '/articles', null, { filter });
//   }

//   async buyArticle(articleId) {
//     return this.callAPI('POST', '/articles/buy', {
//       article_id: articleId
//     });
//   }
// }

// class FollowRoomClient {
//   constructor(serverUrl, authToken) {
//     this.serverUrl = serverUrl;
//     this.authToken = authToken;
//   }

//   async updateFollowSettings(settings) {
//     const res = await fetch(`${this.serverUrl}/api/user/follow-settings`, {
//       method: 'PUT',
//       headers: {
//         'Authorization': `Bearer ${this.authToken}`,
//         'Content-Type': 'application/json'
//       },
//       body: JSON.stringify(settings)
//     });
//     return res.json();
//   }

//   async followUser(userId) {
//     const res = await fetch(`${this.serverUrl}/api/users/${userId}/follow`, {
//       method: 'POST',
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }

//   async unfollowUser(userId) {
//     const res = await fetch(`${this.serverUrl}/api/users/${userId}/follow`, {
//       method: 'DELETE',
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }

//   async approveFollow(followId) {
//     const res = await fetch(`${this.serverUrl}/api/follows/${followId}/approve`, {
//       method: 'POST',
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }

//   async getFollowers(userId, status = 'active') {
//     const res = await fetch(`${this.serverUrl}/api/users/${userId}/followers?status=${status}`, {
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }

//   async createRoom(roomData) {
//     const res = await fetch(`${this.serverUrl}/api/rooms/create`, {
//       method: 'POST',
//       headers: {
//         'Authorization': `Bearer ${this.authToken}`,
//         'Content-Type': 'application/json'
//       },
//       body: JSON.stringify(roomData)
//     });
//     return res.json();
//   }

//   async joinRoom(roomId) {
//     const res = await fetch(`${this.serverUrl}/api/rooms/${roomId}/join-request`, {
//       method: 'POST',
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }

//   async leaveRoom(roomId) {
//     const res = await fetch(`${this.serverUrl}/api/rooms/${roomId}/leave`, {
//       method: 'POST',
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }

//   async getRoomMembers(roomId, status = 'active') {
//     const res = await fetch(`${this.serverUrl}/api/rooms/${roomId}/members?status=${status}`, {
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }

//   async promoteToRoomAdmin(memberId) {
//     const res = await fetch(`${this.serverUrl}/api/room-members/${memberId}/promote`, {
//       method: 'POST',
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }

//   async banRoomMember(memberId) {
//     const res = await fetch(`${this.serverUrl}/api/room-members/${memberId}/ban`, {
//       method: 'POST',
//       headers: { 'Authorization': `Bearer ${this.authToken}` }
//     });
//     return res.json();
//   }
// }

// ==================== TESTS ====================

describe('UserChannelsClient', () => {
  let client;
  const serverUrl = 'http://localhost:8090';
  const authToken = 'test_token';

  beforeEach(() => {
    client = new UserChannelsClient(serverUrl, authToken);
    global.fetch.mockClear();
  });

  describe('SSE Connection', () => {
    test('should connect to SSE endpoint', () => {
      client.connectSSE();
      
      expect(client.sseConnection).toBeDefined();
      expect(client.sseConnection.url).toBe(`${serverUrl}/api/user/sse?token=${authToken}`);
    });

    test('should handle SSE messages', async () => {
      const mockMessage = {
        type: 'notification',
        data: { message: 'Test notification' }
      };

      let receivedMessage;
      client.onSSE('notification', (data) => {
        receivedMessage = data;
      });

      client.handleSSEMessage(mockMessage);

      expect(receivedMessage).toEqual({ message: 'Test notification' });
    });

    test('should handle request callbacks', async () => {
      const requestId = 'req_123';
      const responseData = { success: true, post_id: 'post_123' };

      const promise = new Promise((resolve) => {
        client.requestCallbacks.set(requestId, (error, data) => {
          resolve(data);
        });
      });

      client.handleSSEMessage({
        request_id: requestId,
        data: responseData
      });

      const result = await promise;
      expect(result).toEqual(responseData);
    });

    test('should reconnect on error after delay', async () => {
      let connectCallCount = 0;
      
      // Mock EventSource to track reconnections
      const OriginalEventSource = global.EventSource;
      global.EventSource = class MockEventSource extends OriginalEventSource {
        constructor(url) {
          super(url);
          connectCallCount++;
        }
      };
      
      client.connectSSE();
      expect(connectCallCount).toBe(1);
      
      // Simulate error without throwing
      const errorHandler = client.sseConnection.onerror;
      if (errorHandler) {
        // expect.unreachable(errorHandler.toString())
        // Suppress console.error during test
        const originalError = console.error;
        console.error = () => {};
        
        // Trigger error
        try {
          errorHandler(new Error('Connection lost'));
        } catch (e) {
          // Expected
        }
        
        console.error = originalError;
      }else{
        expect().fail("onError n'est pas defini")
      }

      // Wait for reconnection
      await Bun.sleep(4e3+100);

      expect(connectCallCount).toBe(2);
      
      // Restore
      global.EventSource = OriginalEventSource;
      client.disconnectSSE();
    });
  });

  describe('WebRTC User Room', () => {
    test('should connect to user room', async () => {
      global.fetch.mockImplementation((url) => {
        if (url.includes('/connect')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({ sdp: { type: 'offer', sdp: 'mock_sdp' } })
          });
        }
        return Promise.resolve({
          ok: true,
          json: async () => ({ status: 'connected' })
        });
      });

      await client.connectUserRoom();

      expect(client.peerConnection).toBeDefined();
      expect(client.dataChannel).toBeDefined();
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });

    test('should setup DataChannel handlers', () => {
      client.peerConnection = new RTCPeerConnection();
      client.dataChannel = client.peerConnection.createDataChannel('api');
      
      client.setupDataChannel();

      expect(client.dataChannel.onmessage).toBeDefined();
    });

    test('should handle DataChannel messages', () => {
      const mockMessage = {
        type: 'notification',
        request_id: 'req_123',
        data: { message: 'Test' }
      };

      let callbackCalled = false;
      client.requestCallbacks.set('req_123', (error, data) => {
        callbackCalled = true;
      });

      client.handleRoomMessage(mockMessage);

      expect(callbackCalled).toBe(true);
    });
  });

  describe('API Calls via WebRTC', () => {
    beforeEach(() => {
      client.dataChannel = {
        readyState: 'open',
        send: mock(() => {})
      };
    });

    test('should send API request via DataChannel', async () => {
      const responsePromise = client.callAPI('POST', '/posts', {
        content: 'Test post'
      });

      // Simulate response
      setTimeout(() => {
        const callbacks = Array.from(client.requestCallbacks.entries());
        if (callbacks.length > 0) {
          const [requestId, callback] = callbacks[0];
          callback(null, { success: true, post_id: 'post_123' });
        }
      }, 10);

      const result = await responsePromise;
      expect(result.success).toBe(true);
      expect(result.post_id).toBe('post_123');
    });

    test('should throw error if DataChannel not open', async () => {
      client.dataChannel.readyState = 'closed';

      expect(async () => {
        await client.callAPI('GET', '/posts');
      }).toThrow('User room not connected');
    });

    test('should timeout on no response', async () => {
      const promise = client.callAPI('GET', '/posts');

      // Fast-forward time
      try {
        await promise;
        throw new Error('Should have timed out');
      } catch (error) {
        expect(error.message).toBe('Request timeout');
      }
    },30100);
  });

  describe('Convenience Methods', () => {
    beforeEach(() => {
      client.callAPI = mock(() => Promise.resolve({ success: true }));
    });

    test('should update location', async () => {
      await client.updateLocation(48.8566, 2.3522, 10, 'online');

      expect(client.callAPI).toHaveBeenCalledWith('POST', '/location/update', {
        location: {
          point: { lat: 48.8566, lng: 2.3522 },
          accuracy: 10,
          altitude: 0,
          speed: 0,
          heading: 0
        },
        presence: 'online'
      });
    });

    test('should find nearby users', async () => {
      await client.findNearby(48.8566, 2.3522, 1000);

      expect(client.callAPI).toHaveBeenCalledWith('POST', '/location/nearby', {
        point: { lat: 48.8566, lng: 2.3522 },
        radius: 1000
      });
    });

    test('should create post', async () => {
      await client.createPost('Test content', 'html', true);

      expect(client.callAPI).toHaveBeenCalledWith('POST', '/posts', {
        content: 'Test content',
        type: 'html',
        isPublic: true
      });
    });

    test('should like post', async () => {
      await client.likePost('post_123', 'fire');

      expect(client.callAPI).toHaveBeenCalledWith('POST', '/posts/like', {
        post_id: 'post_123',
        reaction: 'fire'
      });
    });

    test('should comment on post', async () => {
      await client.commentPost('post_123', 'Great post!');

      expect(client.callAPI).toHaveBeenCalledWith('POST', '/posts/comment', {
        post_id: 'post_123',
        content: 'Great post!'
      });
    });

    test('should get articles', async () => {
      await client.getArticles('quantite > 0');

      expect(client.callAPI).toHaveBeenCalledWith('GET', '/articles', null, {
        filter: 'quantite > 0'
      });
    });

    test('should buy article', async () => {
      await client.buyArticle('article_123');

      expect(client.callAPI).toHaveBeenCalledWith('POST', '/articles/buy', {
        article_id: 'article_123'
      });
    });
  });
});

describe('FollowRoomClient', () => {
  let client;
  const serverUrl = 'http://localhost:8090';
  const authToken = 'test_token';

  beforeEach(() => {
    client = new FollowRoomClient(serverUrl, authToken);
    global.fetch.mockClear();
  });

  describe('Follow Operations', () => {
    test('should update follow settings', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      });

      const settings = {
        follow_type: 'paid_period',
        price: 9.99,
        period_days: 30
      };

      const result = await client.updateFollowSettings(settings);

      expect(result.success).toBe(true);
      expect(global.fetch).toHaveBeenCalledWith(
        `${serverUrl}/api/user/follow-settings`,
        expect.objectContaining({
          method: 'PUT'
        })
      );
    });

    test('should follow user', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true, follow_id: 'follow_123' })
      });

      const result = await client.followUser('user_456');

      expect(result.success).toBe(true);
      expect(result.follow_id).toBe('follow_123');
    });

    test('should unfollow user', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      });

      const result = await client.unfollowUser('user_456');

      expect(result.success).toBe(true);
    });

    test('should approve follow', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      });

      const result = await client.approveFollow('follow_123');

      expect(result.success).toBe(true);
    });

    test('should get followers', async () => {
      const mockFollowers = {
        followers: [
          { id: 'follow_1', follower: 'user_1' },
          { id: 'follow_2', follower: 'user_2' }
        ],
        count: 2
      };

      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => mockFollowers
      });

      const result = await client.getFollowers('user_123');

      expect(result.count).toBe(2);
      expect(result.followers).toHaveLength(2);
    });
  });

  describe('Room Operations', () => {
    test('should create room', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ 
          success: true, 
          room_id: 'room_123' 
        })
      });

      const roomData = {
        room_type: 'audio',
        name: 'Test Room',
        join_type: 'free',
        max_participants: 50
      };

      const result = await client.createRoom(roomData);

      expect(result.room_id).toBe('room_123');
    });

    test('should join room', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ 
          success: true, 
          status: 'active' 
        })
      });

      const result = await client.joinRoom('room_123');

      expect(result.success).toBe(true);
      expect(result.status).toBe('active');
    });

    test('should leave room', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      });

      const result = await client.leaveRoom('room_123');

      expect(result.success).toBe(true);
    });

    test('should get room members', async () => {
      const mockMembers = {
        members: [
          { id: 'member_1', role: 'owner' },
          { id: 'member_2', role: 'participant' }
        ],
        count: 2
      };

      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => mockMembers
      });

      const result = await client.getRoomMembers('room_123');

      expect(result.count).toBe(2);
      expect(result.members[0].role).toBe('owner');
    });

    test('should promote to room admin', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      });

      const result = await client.promoteToRoomAdmin('member_123');

      expect(result.success).toBe(true);
    });

    test('should ban room member', async () => {
      global.fetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      });

      const result = await client.banRoomMember('member_123');

      expect(result.success).toBe(true);
    });
  });
});

// ==================== Integration Tests ====================

describe('Integration Tests', () => {
  test('complete user flow', async () => {
    const client = new UserChannelsClient('http://localhost:8090', 'token');
    
    let callCount = 0;
    client.callAPI = mock(() => {
      callCount++;
      if (callCount === 1) return Promise.resolve({ success: true, post_id: 'post_123' });
      if (callCount === 2) return Promise.resolve({ success: true });
      if (callCount === 3) return Promise.resolve({ success: true });
    });

    // Create post
    const post = await client.createPost('Hello World!');
    expect(post.post_id).toBe('post_123');

    // Like post
    await client.likePost('post_123', 'like');

    // Comment on post
    await client.commentPost('post_123', 'Great post!');

    expect(callCount).toBe(3);
  });

  test('complete marketplace flow', async () => {
    const client = new UserChannelsClient('http://localhost:8090', 'token');
    
    let callCount = 0;
    client.callAPI = mock(() => {
      callCount++;
      if (callCount === 1) {
        return Promise.resolve({ 
          articles: [{ id: 'article_123', price: 99.99 }] 
        });
      }
      if (callCount === 2) {
        return Promise.resolve({ 
          success: true, 
          vente_id: 'vente_123' 
        });
      }
    });

    // Get articles
    const articles = await client.getArticles();
    expect(articles.articles).toHaveLength(1);

    // Buy article
    const purchase = await client.buyArticle('article_123');
    expect(purchase.vente_id).toBe('vente_123');
  });

  test('complete follow workflow', async () => {
    const client = new FollowRoomClient('http://localhost:8090', 'token');
    
    let callCount = 0;
    global.fetch.mockImplementation(() => {
      callCount++;
      if (callCount === 1) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ success: true })
        });
      }
      if (callCount === 2) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ 
            success: true, 
            follow_id: 'follow_123',
            status: 'pending' 
          })
        });
      }
      if (callCount === 3) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ success: true })
        });
      }
    });

    // Update settings
    await client.updateFollowSettings({
      follow_type: 'require_approval'
    });

    // Follow user
    const follow = await client.followUser('user_456');
    expect(follow.status).toBe('pending');

    // Approve
    await client.approveFollow('follow_123');
  });
});

// ==================== Performance Tests ====================

describe('Performance Tests', () => {
  test('should handle rapid API calls', async () => {
    const client = new UserChannelsClient('http://localhost:8090', 'token');
    let callCount = 0;
    client.callAPI = mock(() => {
      callCount++;
      return Promise.resolve({ success: true });
    });

    const promises = [];
    for (let i = 0; i < 100; i++) {
      promises.push(client.updateLocation(48.8566, 2.3522, 10));
    }

    await Promise.all(promises);

    expect(callCount).toBe(100);
  });

  test('benchmark SSE message handling', async () => {
    const client = new UserChannelsClient('http://localhost:8090', 'token');
    
    let handledCount = 0;
    client.onSSE('test', () => {
      handledCount++;
    });

    const start = performance.now();
    
    for (let i = 0; i < 10000; i++) {
      client.handleSSEMessage({
        type: 'test',
        data: { id: i }
      });
    }

    const duration = performance.now() - start;
    
    expect(handledCount).toBeLessThan(15000);
    console.log(`Handled 10000 SSE messages in ${duration.toFixed(2)}ms`);
    expect(duration).toBeLessThan(1000); // Should be < 1 second
  });
});