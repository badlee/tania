if(typeof window == 'undefined'){
  global.window = global;
}

// ==================== START MSGPK : https://github.com/ygoe/msgpack.js ====================
!function(){"use strict";function e(r,s){if(s&&s.multiple&&!Array.isArray(r))throw new Error("Invalid argument type: Expected an Array to serialize multiple values.");const d=4294967296;let y,h,n=new Uint8Array(128),i=0;if(s&&s.multiple)for(let e=0;e<r.length;e++)g(r[e]);else g(r);return n.subarray(0,i);function g(r,e){switch(typeof r){case"undefined":w();break;case"boolean":v(r?195:194);break;case"number":t=r,isFinite(t)&&Number.isSafeInteger(t)?0<=t&&t<=127||t<0&&-32<=t?v(t):0<t&&t<=255?m([204,t]):-128<=t&&t<=127?m([208,t]):0<t&&t<=65535?m([205,t>>>8,t]):-32768<=t&&t<=32767?m([209,t>>>8,t]):0<t&&t<=4294967295?m([206,t>>>24,t>>>16,t>>>8,t]):-2147483648<=t&&t<=2147483647?m([210,t>>>24,t>>>16,t>>>8,t]):0<t&&t<=0x10000000000000000?(v(207),b(t)):-0x8000000000000000<=t&&t<=0x8000000000000000?(v(211),b(t)):m(t<0?[211,128,0,0,0,0,0,0,0]:[207,255,255,255,255,255,255,255,255]):(h||(y=new ArrayBuffer(8),h=new DataView(y)),h.setFloat64(0,t),v(203),m(new Uint8Array(y)));break;case"string":var t=r,n=(t=function(t){let r=!0,n=t.length;for(let e=0;e<n;e++)if(127<t.charCodeAt(e)){r=!1;break}let i=0,a=new Uint8Array(t.length*(r?1:4));for(let r=0;r!==n;r++){let e=t.charCodeAt(r);if(e<128)a[i++]=e;else{if(e<2048)a[i++]=e>>6|192;else{if(55295<e&&e<56320){if(++r>=n)throw new Error("UTF-8 encode: incomplete surrogate pair");var o=t.charCodeAt(r);if(o<56320||57343<o)throw new Error("UTF-8 encode: second surrogate character 0x"+o.toString(16)+" at index "+r+" out of range");e=65536+((1023&e)<<10)+(1023&o),a[i++]=e>>18|240,a[i++]=e>>12&63|128}else a[i++]=e>>12|224;a[i++]=e>>6&63|128}a[i++]=63&e|128}}return r?a:a.subarray(0,i)}(r)).length;n<=31?v(160+n):m(n<=255?[217,n]:n<=65535?[218,n>>>8,n]:[219,n>>>24,n>>>16,n>>>8,n]),m(t);break;case"object":if(null===r)w();else if(r instanceof Date){n=r;var i=n.getTime()/1e3;{var a;0===n.getMilliseconds()&&0<=i&&i<4294967296?m([214,255,i>>>24,i>>>16,i>>>8,i]):0<=i&&i<17179869184?m([215,255,(a=1e6*n.getMilliseconds())>>>22,a>>>14,a>>>6,a<<2>>>0|i/d,i>>>24,i>>>16,i>>>8,i]):(m([199,12,255,(a=1e6*n.getMilliseconds())>>>24,a>>>16,a>>>8,a]),b(i))}}else if(Array.isArray(r))p(r);else if(r instanceof Uint8Array||r instanceof Uint8ClampedArray){i=r;var o=i.length;m(o<=255?[196,o]:o<=65535?[197,o>>>8,o]:[198,o>>>24,o>>>16,o>>>8,o]);m(i)}else if(r instanceof Int8Array||r instanceof Int16Array||r instanceof Uint16Array||r instanceof Int32Array||r instanceof Uint32Array||r instanceof Float32Array||r instanceof Float64Array)p(r);else{var f,l,u=r;let e=0;for(f in u)void 0!==u[f]&&e++;for(l in e<=15?v(128+e):e<=65535?m([222,e>>>8,e]):m([223,e>>>24,e>>>16,e>>>8,e]),u){var c=u[l];void 0!==c&&(g(l),g(c))}}break;default:if(e||!s||!s.invalidTypeReplacement)throw new Error("Invalid argument type: The type '"+typeof r+"' cannot be serialized.");"function"==typeof s.invalidTypeReplacement?g(s.invalidTypeReplacement(r),!0):g(s.invalidTypeReplacement,!0)}var t}function w(){v(192)}function p(r){var t=r.length;t<=15?v(144+t):m(t<=65535?[220,t>>>8,t]:[221,t>>>24,t>>>16,t>>>8,t]);for(let e=0;e<t;e++)g(r[e])}function v(e){if(n.length<i+1){let e=2*n.length;for(;e<i+1;)e*=2;var r=new Uint8Array(e);r.set(n),n=r}n[i]=e,i++}function m(r){if(n.length<i+r.length){let e=2*n.length;for(;e<i+r.length;)e*=2;var t=new Uint8Array(e);t.set(n),n=t}n.set(r,i),i+=r.length}function b(e){let r,t;t=0<=e?(r=e/d,e%d):(e++,r=Math.abs(e)/d,t=Math.abs(e)%d,r=~r,~t),m([r>>>24,r>>>16,r>>>8,r,t>>>24,t>>>16,t>>>8,t])}}function r(o,e){const i=4294967296;let f=0;if("object"!=typeof(o=o instanceof ArrayBuffer?new Uint8Array(o):o)||void 0===o.length)throw new Error("Invalid argument type: Expected a byte array (Array or Uint8Array) to deserialize.");if(!o.length)throw new Error("Invalid argument: The byte array to deserialize is empty.");o instanceof Uint8Array||(o=new Uint8Array(o));let r;if(e&&e.multiple)for(r=[];f<o.length;)r.push(n());else r=n();return r;function n(){var e=o[f++];if(0<=e&&e<=127)return e;if(128<=e&&e<=143)return c(e-128);if(144<=e&&e<=159)return s(e-144);if(160<=e&&e<=191)return d(e-160);if(192===e)return null;if(193===e)throw new Error("Invalid byte code 0xc1 found.");if(194===e)return!1;if(195===e)return!0;if(196===e)return u(-1,1);if(197===e)return u(-1,2);if(198===e)return u(-1,4);if(199===e)return y(-1,1);if(200===e)return y(-1,2);if(201===e)return y(-1,4);if(202===e)return t(4);if(203===e)return t(8);if(204===e)return l(1);if(205===e)return l(2);if(206===e)return l(4);if(207===e)return l(8);if(208===e)return a(1);if(209===e)return a(2);if(210===e)return a(4);if(211===e)return a(8);if(212===e)return y(1);if(213===e)return y(2);if(214===e)return y(4);if(215===e)return y(8);if(216===e)return y(16);if(217===e)return d(-1,1);if(218===e)return d(-1,2);if(219===e)return d(-1,4);if(220===e)return s(-1,2);if(221===e)return s(-1,4);if(222===e)return c(-1,2);if(223===e)return c(-1,4);if(224<=e&&e<=255)return e-256;throw console.debug("msgpack array:",o),new Error("Invalid byte value '"+e+"' at index "+(f-1)+" in the MessagePack binary data (length "+o.length+"): Expecting a range of 0 to 255. This is not a byte array.")}function a(e){let r=0,t=!0;for(;0<e--;){var n;t?(n=o[f++],r+=127&n,128&n&&(r-=128),t=!1):r=(r*=256)+o[f++]}return r}function l(e){let r=0;for(;0<e--;)r=(r*=256)+o[f++];return r}function t(e){var r=new DataView(o.buffer,f+o.byteOffset,e);return f+=e,4===e?r.getFloat32(0,!1):8===e?r.getFloat64(0,!1):void 0}function u(e,r){e<0&&(e=l(r));r=o.subarray(f,f+e);return f+=e,r}function c(e,r){e<0&&(e=l(r));for(var t={};0<e--;)t[n()]=n();return t}function s(e,r){e<0&&(e=l(r));for(var t=[];0<e--;)t.push(n());return t}function d(e,n){e<0&&(e=l(n));n=f;f+=e;{var i=o,a=e;let r=n,t="";for(a+=n;r<a;){let e=i[r++];if(127<e)if(191<e&&e<224){if(r>=a)throw new Error("UTF-8 decode: incomplete 2-byte sequence");e=(31&e)<<6|63&i[r++]}else if(223<e&&e<240){if(r+1>=a)throw new Error("UTF-8 decode: incomplete 3-byte sequence");e=(15&e)<<12|(63&i[r++])<<6|63&i[r++]}else{if(!(239<e&&e<248))throw new Error("UTF-8 decode: unknown multibyte start 0x"+e.toString(16)+" at index "+(r-1));if(r+2>=a)throw new Error("UTF-8 decode: incomplete 4-byte sequence");e=(7&e)<<18|(63&i[r++])<<12|(63&i[r++])<<6|63&i[r++]}if(e<=65535)t+=String.fromCharCode(e);else{if(!(e<=1114111))throw new Error("UTF-8 decode: code point 0x"+e.toString(16)+" exceeds UTF-16 reach");e-=65536,t=(t+=String.fromCharCode(e>>10|55296))+String.fromCharCode(1023&e|56320)}}return t}}function y(e,r){e<0&&(e=l(r));var t,r=l(1),e=u(e);if(255!==r)return{type:r,data:e};if(4===(r=e).length)return t=(r[0]<<24>>>0)+(r[1]<<16>>>0)+(r[2]<<8>>>0)+r[3],new Date(1e3*t);if(8===r.length)return t=(r[0]<<22>>>0)+(r[1]<<14>>>0)+(r[2]<<6>>>0)+(r[3]>>>2),n=(3&r[3])*i+(r[4]<<24>>>0)+(r[5]<<16>>>0)+(r[6]<<8>>>0)+r[7],new Date(1e3*n+t/1e6);if(12!==r.length)throw new Error("Invalid data length for a date value.");var n=(r[0]<<24>>>0)+(r[1]<<16>>>0)+(r[2]<<8>>>0)+r[3],r=(f-=8,a(8));return new Date(1e3*r+n/1e6)}}var t={serialize:e,deserialize:r,encode:e,decode:r};"object"==typeof module&&module&&"object"==typeof module.exports?module.exports=t:window[window.msgpackJsName||"msgpack"]=t}();
// ==================== END   MSGPK : https://github.com/ygoe/msgpack.js ====================

// ==================== EVENT EMITTER ====================

class EventEmitter {
  constructor() {
    this.events = new Map();
  }

  on(event, listener) {
    if (!this.events.has(event)) {
      this.events.set(event, []);
    }
    this.events.get(event).push(listener);
    return this;
  }

  once(event, listener) {
    const onceWrapper = (...args) => {
      listener(...args);
      this.off(event, onceWrapper);
    };
    return this.on(event, onceWrapper);
  }

  off(event, listenerToRemove) {
    if (!this.events.has(event)) return this;
    if(!listenerToRemove){
      return this.removeAllListeners(event);
    }
    const listeners = this.events.get(event);
    const index = listeners.indexOf(listenerToRemove);
    
    if (index !== -1) {
      listeners.splice(index, 1);
    }

    if (listeners.length === 0) {
      this.events.delete(event);
    }

    return this;
  }

  emit(event, ...args) {
    if (!this.events.has(event)) return false;

    const listeners = this.events.get(event);
    listeners.forEach(listener => {
      try {
        listener(...args);
      } catch (error) {
        console.error(`Error in event listener for "${event}":`, error);
      }
    });

    return true;
  }

  removeAllListeners(event) {
    if (event) {
      this.events.delete(event);
    } else {
      this.events.clear();
    }
    return this;
  }

  listenerCount(event) {
    return this.events.has(event) ? this.events.get(event).length : 0;
  }

  eventNames() {
    return Array.from(this.events.keys());
  }
}

// ==================== USER CHANNELS CLIENT ====================

class UserChannelsClient extends EventEmitter {
  constructor(serverUrl, authToken, reconnectionTTL) {
    super();
    
    this.serverUrl = serverUrl;
    this.authToken = authToken;
    
    // Notification Config
    /** @type NotificationOptions */
    this.notificationOptions = {
      icon: '/icon.png',
      requireInteraction: true,
    };
    this.notificationTitle = "Notification"

    // SSE Channel
    this.sseConnection = null;
    
    // WebRTC Room
    this.userRoom = null;
    this.dataChannel = null;
    this.peerConnection = null;
    this.requestCallbacks = new Map();
    this.requestIdCounter = 0;
    this.reconnectionTTL = reconnectionTTL || 4000;
    
    if (this.reconnectionTTL < 0) {
      this.reconnectionTTL = 0;
    }
    
    this.reconnectionTimeToWait = this.reconnectionTTL;
  }

  // ==================== SSE CHANNEL ====================

  connectSSE() {
    const url = `${this.serverUrl}/api/user/sse?token=${this.authToken}`;
    this.sseConnection = new EventSource(url);

    this.sseConnection.onopen = () => {
      console.log('✅ SSE Channel connected');
      this.emit('sse:connected');
      this.reconnectionTimeToWait = this.reconnectionTTL; // Reset on success
    };

    this.sseConnection.onmessage = (event) => {
      const data = JSON.parse(event.data);
      this.#handleSSEMessage(data);
    };

    this.sseConnection.onerror = (error) => {
      console.error('❌ SSE error:', error);
      this.emit('sse:error', error);
      
      if (this.reconnectionTTL > 0) {
        setTimeout(() => {
          this.sseConnection.close(); // Force close
          this.emit('sse:reconnecting', this.reconnectionTimeToWait);
          this.connectSSE();
        }, this.reconnectionTimeToWait);
        
        this.reconnectionTimeToWait += this.reconnectionTimeToWait * 0.05;
      }
    };
  }

  #handleSSEMessage(message) {
    // Emit raw message event
    this.emit('sse:message', message);
    this.emit('message', message);

    // Handle response to specific request
    if (message.request_id && this.requestCallbacks.has(message.request_id)) {
      const callback = this.requestCallbacks.get(message.request_id);
      callback(null, message.data);
      this.requestCallbacks.delete(message.request_id);
      return;
    }

    // Emit typed events
    this.emit(`sse:${message.type}`, message.data);
    this.emit(`${message.type}`, message.data);

    // Default handlers
    switch (message.type) {
      case 'connected':
        console.log('✅ SSE ready:', message.data.message);
        break;
      case 'notification':
        this.#handleNotification(message.data);
        break;
      case 'post_liked':
        this.#handlePostLiked(message.data);
        break;
      case 'post_commented':
        this.#handlePostCommented(message.data);
        break;
      case 'location_update':
        this.#handleLocationUpdate(message.data);
        break;
      case 'geo_event':
        this.#handleGeoEvent(message.data);
        break;
    }
  }

  disconnectSSE() {
    if (this.sseConnection) {
      this.sseConnection.close();
      this.sseConnection = null;
      this.emit('sse:disconnected');
    }
  }

  // ==================== USER WEBRTC ROOM ====================

  async connectUserRoom() {
    console.log('🔌 Connecting to user room...');
    this.emit('room:connecting');

    try {
      // Create peer connection
      this.peerConnection = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      // Handle ICE candidates
      this.peerConnection.onicecandidate = (event) => {
        if (event.candidate) {
          console.log('🧊 ICE candidate:', event.candidate);
          this.emit('room:ice-candidate', event.candidate);
        }
      };

      // Handle connection state changes
      this.peerConnection.onconnectionstatechange = () => {
        const state = this.peerConnection.connectionState;
        console.log('🔗 Connection state:', state);
        this.emit('room:connection-state', state);
        
        if (state === 'connected') {
          this.emit('room:connected');
        } else if (state === 'disconnected') {
          this.emit('room:disconnected');
        } else if (state === 'failed') {
          this.emit('room:failed');
        }
      };

      // Handle data channel from server
      this.peerConnection.ondatachannel = (event) => {
        console.log('📡 DataChannel received from server');
        this.dataChannel = event.channel;
        this.setupDataChannel();
        this.emit('room:datachannel-received', event.channel);
      };

      // Get offer from server
      const response = await fetch(`${this.serverUrl}/api/user/room/connect`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.authToken}`,
          'Content-Type': 'application/json'
        }
      });

      const { sdp } = await response.json();

      // Set remote description
      await this.peerConnection.setRemoteDescription(sdp);

      // Create answer
      const answer = await this.peerConnection.createAnswer();
      await this.peerConnection.setLocalDescription(answer);

      // Send answer to server
      await fetch(`${this.serverUrl}/api/user/room/answer`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.authToken}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(answer)
      });

      console.log('✅ User room connected');
    } catch (error) {
      console.error('❌ Failed to connect user room:', error);
      this.emit('room:error', error);
      throw error;
    }
  }

  setupDataChannel() {
    this.dataChannel.onopen = () => {
      console.log('✅ User room DataChannel opened');
      this.emit('datachannel:open');
    };

    this.dataChannel.onmessage = async (event) => {
      const data = event.data instanceof ArrayBuffer 
        ? msgpack.decode(new Uint8Array(event.data))
        : JSON.parse(event.data);
      
      this.emit('datachannel:message', data);
      this.handleRoomMessage(data);
    };

    this.dataChannel.onerror = (error) => {
      console.error('❌ DataChannel error:', error);
      this.emit('datachannel:error', error);
    };

    this.dataChannel.onclose = () => {
      console.log('🔌 DataChannel closed');
      this.emit('datachannel:close');
    };
  }

  handleRoomMessage(message) {
    console.log('📨 Room Message:', message.type);

    // Emit specific message type event
    this.emit(`room:${message.type}`, message.data);

    // Handle API response
    if (message.request_id && this.requestCallbacks.has(message.request_id)) {
      const callback = this.requestCallbacks.get(message.request_id);
      callback(null, message.data);
      this.requestCallbacks.delete(message.request_id);
      return;
    }

    // Handle notifications/events
    switch (message.type) {
      case 'welcome':
        console.log('👋 Welcome:', message.data.message);
        break;
      case 'notification':
        this.#handleNotification(message.data);
        break;
      case 'presence_change':
        this.#handlePresenceChange(message.data);
        break;
      case 'location_update':
        this.#handleLocationUpdate(message.data);
        break;
    }
  }

  disconnectUserRoom() {
    if (this.dataChannel) {
      this.dataChannel.close();
    }
    if (this.peerConnection) {
      this.peerConnection.close();
    }
    this.emit('room:disconnected');
  }

  // ==================== API CALLS VIA WEBRTC ====================

  async callAPI(method, endpoint, body = null, query = null) {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      const error = new Error('User room not connected');
      this.emit('api:error', error);
      throw error;
    }

    const requestId = `req_${++this.requestIdCounter}_${Date.now()}`;

    const request = {
      request_id: requestId,
      method: method,
      endpoint: endpoint,
      body: body,
      query: query
    };

    // Emit API request event
    this.emit('api:request', { method, endpoint, requestId });

    // Send request
    const encoded = msgpack.encode(request);
    this.dataChannel.send(encoded);

    // Wait for response
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.requestCallbacks.delete(requestId);
        const error = new Error('Request timeout');
        this.emit('api:timeout', { requestId, method, endpoint });
        reject(error);
      }, 30000); // 30s timeout

      this.requestCallbacks.set(requestId, (error, data) => {
        clearTimeout(timeout);
        if (error) {
          this.emit('api:error', { error, requestId, method, endpoint });
          reject(error);
        } else {
          this.emit('api:response', { data, requestId, method, endpoint });
          resolve(data);
        }
      });
    });
  }

  // ==================== CONVENIENCE METHODS ====================

  // Location
  async updateLocation(lat, lng, accuracy = 10, presence = 'online') {
    const result = await this.callAPI('POST', '/location/update', {
      location: {
        point: { lat, lng },
        accuracy: accuracy,
        altitude: 0,
        speed: 0,
        heading: 0
      },
      presence: presence
    });
    this.emit('location:updated', { lat, lng, presence });
    return result;
  }

  async findNearby(lat, lng, radius) {
    const result = await this.callAPI('POST', '/location/nearby', {
      point: { lat, lng },
      radius: radius
    });
    this.emit('location:nearby-found', result);
    return result;
  }

  async findInPolygon(polygon) {
    const result = await this.callAPI('POST', '/location/polygon', {
      polygon: polygon
    });
    this.emit('location:polygon-found', result);
    return result;
  }

  // Posts
  async getPosts(filter = 'isPublic = true') {
    const result = await this.callAPI('GET', '/posts', null, { filter });
    this.emit('posts:fetched', result);
    return result;
  }

  async createPost(content, type = 'html', isPublic = true) {
    const result = await this.callAPI('POST', '/posts', {
      content: content,
      type: type,
      isPublic: isPublic
    });
    this.emit('posts:created', result);
    return result;
  }

  async likePost(postId, reaction = 'like') {
    const result = await this.callAPI('POST', '/posts/like', {
      post_id: postId,
      reaction: reaction
    });
    this.emit('posts:liked', { postId, reaction });
    return result;
  }

  async commentPost(postId, content) {
    const result = await this.callAPI('POST', '/posts/comment', {
      post_id: postId,
      content: content
    });
    this.emit('posts:commented', { postId, content });
    return result;
  }

  // Marketplace
  async getArticles(filter = 'quantite > 0') {
    const result = await this.callAPI('GET', '/articles', null, { filter });
    this.emit('articles:fetched', result);
    return result;
  }

  async buyArticle(articleId) {
    const result = await this.callAPI('POST', '/articles/buy', {
      article_id: articleId
    });
    this.emit('articles:bought', { articleId, result });
    return result;
  }

  // Presence
  async updatePresence(presence) {
    const result = await this.callAPI('POST', '/presence/update', {
      presence: presence
    });
    this.emit('presence:updated', presence);
    return result;
  }

  // Rooms
  async joinRoom(roomId) {
    const result = await this.callAPI('POST', '/rooms/join', {
      room_id: roomId
    });
    this.emit('rooms:joined', roomId);
    return result;
  }

  async leaveRoom(roomId) {
    const result = await this.callAPI('POST', '/rooms/leave', {
      room_id: roomId
    });
    this.emit('rooms:left', roomId);
    return result;
  }

  // ==================== EVENT HANDLERS ====================

  #handleNotification(data) {
    console.log('🔔 Notification:', data);
    this.emit('notification', data);
    
    // Show browser notification
    if ('Notification' in window && Notification.permission === 'granted') {
      if (!document.hasFocus()) {
        new Notification(data.title || 'Notification', {
          ...this.notificationOptions,
          body: data.message,
          data: data,
        });
      }
    }
  }

  #handlePostLiked(data) {
    console.log('❤️ Your post was liked:', data);
    this.emit('post:liked', data);
  }

  #handlePostCommented(data) {
    console.log('💬 New comment on your post:', data);
    this.emit('post:commented', data);
  }

  #handleLocationUpdate(data) {
    console.log('📍 Location update:', data);
    this.emit('location:update', data);
  }

  #handleGeoEvent(data) {
    console.log('🎯 Geo event:', data);
    this.emit('geo:event', data);
  }

  #handlePresenceChange(data) {
    console.log('👤 Presence changed:', data);
    this.emit('presence:change', data);
  }
}

// ==================== USAGE EXAMPLES ====================

/*
// Créer le client
const client = new UserChannelsClient('http://localhost:8090', 'token');

// ==================== Écouter les événements SSE ====================

client.on('sse:connected', () => {
  console.log('SSE connected!');
});

client.on('sse:error', (error) => {
  console.error('SSE error:', error);
});

client.on('sse:reconnecting', (delay) => {
  console.log(`SSE reconnecting in ${delay}ms`);
});

client.on('sse:message', (message) => {
  console.log('Received SSE message:', message);
});

// Écouter des types spécifiques
client.on('sse:notification', (data) => {
  console.log('Notification via SSE:', data);
});

client.on('sse:post_liked', (data) => {
  console.log('Post liked:', data);
});

// ==================== Écouter les événements Room ====================

client.on('room:connecting', () => {
  console.log('Connecting to room...');
});

client.on('room:connected', () => {
  console.log('Room connected!');
});

client.on('room:disconnected', () => {
  console.log('Room disconnected');
});

client.on('room:error', (error) => {
  console.error('Room error:', error);
});

client.on('room:connection-state', (state) => {
  console.log('Connection state:', state);
});

// ==================== Écouter les événements DataChannel ====================

client.on('datachannel:open', () => {
  console.log('DataChannel is open!');
});

client.on('datachannel:message', (data) => {
  console.log('Received message:', data);
});

client.on('datachannel:error', (error) => {
  console.error('DataChannel error:', error);
});

// ==================== Écouter les événements API ====================

client.on('api:request', ({ method, endpoint, requestId }) => {
  console.log(`API Request: ${method} ${endpoint} [${requestId}]`);
});

client.on('api:response', ({ data, requestId }) => {
  console.log(`API Response [${requestId}]:`, data);
});

client.on('api:error', ({ error, requestId }) => {
  console.error(`API Error [${requestId}]:`, error);
});

client.on('api:timeout', ({ requestId, method, endpoint }) => {
  console.error(`API Timeout [${requestId}]: ${method} ${endpoint}`);
});

// ==================== Écouter les événements métier ====================

// Notifications
client.on('notification', (data) => {
  console.log('📬 Notification:', data);
  // Afficher une notification UI personnalisée
});

// Posts
client.on('post:liked', (data) => {
  console.log('❤️ Someone liked your post:', data);
});

client.on('post:commented', (data) => {
  console.log('💬 New comment:', data);
});

client.on('posts:created', (result) => {
  console.log('✅ Post created:', result);
});

// Location
client.on('location:updated', ({ lat, lng, presence }) => {
  console.log(`📍 Location updated: ${lat}, ${lng} [${presence}]`);
});

client.on('location:nearby-found', (result) => {
  console.log(`👥 Found ${result.count} nearby users`);
});

// Presence
client.on('presence:change', (data) => {
  console.log('👤 Presence changed:', data);
});

client.on('presence:updated', (presence) => {
  console.log('✅ Presence updated to:', presence);
});

// Geo events
client.on('geo:event', (data) => {
  console.log('🎯 Geo event:', data);
  if (data.type === 'user_entered') {
    console.log('Entered geofence:', data.fence_id);
  }
});

// Articles
client.on('articles:bought', ({ articleId, result }) => {
  console.log(`🛒 Bought article ${articleId}:`, result);
});

// Rooms
client.on('rooms:joined', (roomId) => {
  console.log(`🚪 Joined room: ${roomId}`);
});

client.on('rooms:left', (roomId) => {
  console.log(`🚪 Left room: ${roomId}`);
});

// ==================== Utilisation avec Once ====================

// Écouter un seul événement
client.once('sse:connected', () => {
  console.log('SSE connected for the first time!');
});

// ==================== Supprimer des listeners ====================

const handler = (data) => console.log('Handler:', data);

// Ajouter
client.on('notification', handler);

// Supprimer
client.off('notification', handler);

// Supprimer tous les listeners d'un événement
client.removeAllListeners('notification');
client.off('notification');

// Supprimer tous les listeners
client.removeAllListeners();

// ==================== Vérifier les listeners ====================

console.log('Listener count:', client.listenerCount('notification'));
console.log('Event names:', client.eventNames());

// ==================== Exemple d'intégration React ====================

function MyComponent() {
  const [notifications, setNotifications] = useState([]);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const client = new UserChannelsClient('http://localhost:8090', token);

    // SSE events
    client.on('sse:connected', () => setIsConnected(true));
    client.on('sse:disconnected', () => setIsConnected(false));

    // Business events
    client.on('notification', (data) => {
      setNotifications(prev => [...prev, data]);
    });

    client.on('post:liked', (data) => {
      toast.success(`${data.user_id} liked your post!`);
    });

    client.on('geo:event', (data) => {
      if (data.type === 'user_entered') {
        toast.info(`You entered ${data.fence_name}`);
      }
    });

    // Connect
    client.connectSSE();
    client.connectUserRoom();

    // Cleanup
    return () => {
      client.removeAllListeners();
      client.disconnectSSE();
      client.disconnectUserRoom();
    };
  }, [token]);

  return (
    <div>
      <div>Status: {isConnected ? '✅ Connected' : '❌ Disconnected'}</div>
      <div>Notifications: {notifications.length}</div>
    </div>
  );
}
*/

// Export
export default UserChannelsClient
export { UserChannelsClient, EventEmitter, msgpack }