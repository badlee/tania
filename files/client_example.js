// ==================== CLIENT WEBRTC ====================

class WebRTCClient {
  constructor(serverUrl, authToken) {
    this.serverUrl = serverUrl;
    this.authToken = authToken;
    this.peerConnection = null;
    this.dataChannel = null;
    this.participantId = null;
    this.localStream = null;
  }

  // Créer et rejoindre une room
  async joinRoom(roomId, roomType = 'audio') {
    try {
      // Get user media
      const constraints = {
        audio: true,
        video: roomType === 'video'
      };
      
      this.localStream = await navigator.mediaDevices.getUserMedia(constraints);

      // Join room and get offer
      const response = await fetch(`${this.serverUrl}/api/rooms/${roomId}/join`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.authToken}`,
          'Content-Type': 'application/json'
        }
      });

      const data = await response.json();
      this.participantId = data.participant_id;

      // Create peer connection
      this.peerConnection = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      // Setup data channel
      this.dataChannel = this.peerConnection.createDataChannel('events');
      this.setupDataChannel();

      // Add local tracks
      this.localStream.getTracks().forEach(track => {
        this.peerConnection.addTrack(track, this.localStream);
      });

      // Handle incoming tracks
      this.peerConnection.ontrack = (event) => {
        console.log('Received remote track:', event.track.kind);
        this.onRemoteTrack(event.streams[0]);
      };

      // Set remote description (offer from server)
      await this.peerConnection.setRemoteDescription(data.sdp);

      // Create answer
      const answer = await this.peerConnection.createAnswer();
      await this.peerConnection.setLocalDescription(answer);

      // Send answer to server
      await fetch(`${this.serverUrl}/api/rooms/${roomId}/participants/${this.participantId}/answer`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.authToken}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(answer)
      });

      console.log('Successfully joined room:', roomId);
      return true;

    } catch (error) {
      console.error('Error joining room:', error);
      throw error;
    }
  }

  setupDataChannel() {
    this.dataChannel.onopen = () => {
      console.log('DataChannel opened');
    };

    this.dataChannel.onmessage = async (event) => {
      // Decode msgpack
      const data = event.data instanceof ArrayBuffer 
        ? msgpack.decode(new Uint8Array(event.data))
        : JSON.parse(event.data);
      
      this.onDataEvent(data);
    };

    this.dataChannel.onerror = (error) => {
      console.error('DataChannel error:', error);
    };
  }

  // Envoyer un événement via DataChannel
  sendEvent(type, data) {
    if (this.dataChannel && this.dataChannel.readyState === 'open') {
      const event = {
        type: type,
        data: data,
        timestamp: Date.now()
      };
      
      // Encode with msgpack
      const encoded = msgpack.encode(event);
      this.dataChannel.send(encoded);
    }
  }

  // Envoyer un message chat
  sendChatMessage(message) {
    this.sendEvent('chat', { message });
  }

  // Envoyer une réaction
  sendReaction(reactionType) {
    this.sendEvent('reaction', { type: reactionType });
  }

  // Callbacks à définir
  onRemoteTrack(stream) {
    // Implémenté par l'utilisateur
    console.log('Remote stream received');
  }

  onDataEvent(event) {
    // Implémenté par l'utilisateur
    console.log('Data event:', event);
  }

  // Quitter la room
  disconnect() {
    if (this.localStream) {
      this.localStream.getTracks().forEach(track => track.stop());
    }
    if (this.peerConnection) {
      this.peerConnection.close();
    }
  }
}

// ==================== CLIENT API SOCIAL ====================

class SocialAPIClient {
  constructor(serverUrl, authToken) {
    this.serverUrl = serverUrl;
    this.authToken = authToken;
    this.eventSources = {};
  }

  // Créer un post
  async createPost(postData) {
    const formData = new FormData();
    
    Object.keys(postData).forEach(key => {
      if (postData[key] !== null && postData[key] !== undefined) {
        if (key === 'images' && Array.isArray(postData[key])) {
          postData[key].forEach(file => formData.append('images', file));
        } else {
          formData.append(key, postData[key]);
        }
      }
    });

    const response = await fetch(`${this.serverUrl}/api/collections/posts/records`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.authToken}`
      },
      body: formData
    });

    return await response.json();
  }

  // Liker un post
  async likePost(postId, reaction = 'like') {
    const response = await fetch(`${this.serverUrl}/api/posts/${postId}/like?reaction=${reaction}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        'Content-Type': 'application/json'
      }
    });

    return await response.json();
  }

  // Commenter un post
  async commentPost(postId, content, parentComment = null) {
    const response = await fetch(`${this.serverUrl}/api/posts/${postId}/comment`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        content,
        parent_comment: parentComment
      })
    });

    return await response.json();
  }

  // Récupérer les posts
  async getPosts(page = 1, perPage = 20, filter = '') {
    const params = new URLSearchParams({
      page,
      perPage,
      filter,
      expand: 'user,article',
      sort: '-created'
    });

    const response = await fetch(`${this.serverUrl}/api/collections/posts/records?${params}`, {
      headers: {
        'Authorization': `Bearer ${this.authToken}`
      }
    });

    return await response.json();
  }

  // Créer un article
  async createArticle(articleData) {
    const formData = new FormData();
    
    Object.keys(articleData).forEach(key => {
      if (articleData[key] !== null && articleData[key] !== undefined) {
        if (key === 'images' && Array.isArray(articleData[key])) {
          articleData[key].forEach(file => formData.append('images', file));
        } else {
          formData.append(key, articleData[key]);
        }
      }
    });

    const response = await fetch(`${this.serverUrl}/api/collections/articles/records`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.authToken}`
      },
      body: formData
    });

    return await response.json();
  }

  // Acheter un article
  async buyArticle(articleId) {
    const response = await fetch(`${this.serverUrl}/api/articles/${articleId}/buy`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        'Content-Type': 'application/json'
      }
    });

    return await response.json();
  }

  // S'abonner aux événements temps réel (SSE)
  subscribeToEvents(topic, callback) {
    const eventSource = new EventSource(`${this.serverUrl}/api/events/${topic}`);
    
    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data);
      callback(data);
    };

    eventSource.onerror = (error) => {
      console.error('SSE error:', error);
      eventSource.close();
    };

    this.eventSources[topic] = eventSource;
    return eventSource;
  }

  // Se désabonner
  unsubscribeFromEvents(topic) {
    if (this.eventSources[topic]) {
      this.eventSources[topic].close();
      delete this.eventSources[topic];
    }
  }

  // Récupérer mes opérations
  async getMyOperations(page = 1, perPage = 20) {
    const params = new URLSearchParams({
      page,
      perPage,
      expand: 'vente,vente.article',
      sort: '-created'
    });

    const response = await fetch(`${this.serverUrl}/api/collections/operations/records?${params}`, {
      headers: {
        'Authorization': `Bearer ${this.authToken}`
      }
    });

    return await response.json();
  }

  // Récupérer mes ventes (en tant que vendeur)
  async getMySales(page = 1, perPage = 20) {
    const params = new URLSearchParams({
      page,
      perPage,
      filter: `article.user = "${this.userId}"`,
      expand: 'article,user',
      sort: '-created'
    });

    const response = await fetch(`${this.serverUrl}/api/collections/ventesArticle/records?${params}`, {
      headers: {
        'Authorization': `Bearer ${this.authToken}`
      }
    });

    return await response.json();
  }
}

// ==================== EXEMPLE D'UTILISATION ====================

async function example() {
  const SERVER_URL = 'http://localhost:8090';
  const AUTH_TOKEN = 'votre_token_ici';

  // 1. Client Social
  const socialClient = new SocialAPIClient(SERVER_URL, AUTH_TOKEN);

  // Créer un post avec article
  const article = await socialClient.createArticle({
    title: 'iPhone 15 Pro',
    desc: 'Neuf, jamais utilisé',
    prixOriginal: 1200,
    prix: 999,
    quantite: 1,
    user: 'user_id_here',
    images: [file1, file2] // Files from input
  });

  const post = await socialClient.createPost({
    user: 'user_id_here',
    type: 'images',
    content: 'Vente iPhone 15 Pro - Excellent état!',
    article: article.id,
    action: 'buy',
    actionText: 'Acheter maintenant',
    isPublic: true,
    categories: ['tech'],
    images: [file1, file2]
  });

  console.log('Post créé:', post);

  // S'abonner aux événements temps réel
  socialClient.subscribeToEvents('post_events', (event) => {
    console.log('Nouvel événement:', event);
    
    if (event.type === 'new_post') {
      console.log('Nouveau post:', event.post_id);
    } else if (event.type === 'like') {
      console.log('Nouveau like sur:', event.post_id);
    } else if (event.type === 'comment') {
      console.log('Nouveau commentaire:', event.comment);
    }
  });

  // Liker un post
  await socialClient.likePost(post.id, 'fire');

  // Commenter
  await socialClient.commentPost(post.id, 'Super produit! 🔥');

  // Acheter
  const purchase = await socialClient.buyArticle(article.id);
  console.log('Achat effectué:', purchase);

  // 2. Client WebRTC
  const webrtcClient = new WebRTCClient(SERVER_URL, AUTH_TOKEN);

  // Callbacks
  webrtcClient.onRemoteTrack = (stream) => {
    const audio = document.createElement('audio');
    audio.srcObject = stream;
    audio.autoplay = true;
    document.body.appendChild(audio);
  };

  webrtcClient.onDataEvent = (event) => {
    console.log('Event reçu:', event);
    
    if (event.type === 'chat') {
      console.log(`${event.data.from}: ${event.data.message}`);
    } else if (event.type === 'participant_joined') {
      console.log('Participant rejoint:', event.data.participant_id);
    }
  };

  // Rejoindre une room audio
  await webrtcClient.joinRoom('room_123', 'audio');

  // Envoyer un message
  webrtcClient.sendChatMessage('Salut tout le monde!');

  // Envoyer une réaction
  webrtcClient.sendReaction('👏');

  // Récupérer les posts
  const posts = await socialClient.getPosts(1, 20, 'isPublic=true');
  console.log('Posts:', posts);

  // Récupérer mes opérations
  const operations = await socialClient.getMyOperations();
  console.log('Mes opérations:', operations);
}

// ==================== INTEGRATION REACT ====================

/* Exemple d'utilisation dans React:

import { useEffect, useState } from 'react';

function SocialFeed() {
  const [posts, setPosts] = useState([]);
  const [client] = useState(() => new SocialAPIClient(SERVER_URL, AUTH_TOKEN));

  useEffect(() => {
    // Charger les posts
    client.getPosts().then(data => setPosts(data.items));

    // S'abonner aux nouveaux posts
    client.subscribeToEvents('post_events', (event) => {
      if (event.type === 'new_post') {
        // Recharger ou ajouter le nouveau post
        client.getPosts().then(data => setPosts(data.items));
      }
    });

    return () => {
      client.unsubscribeFromEvents('post_events');
    };
  }, []);

  const handleLike = async (postId) => {
    await client.likePost(postId);
  };

  return (
    <div>
      {posts.map(post => (
        <div key={post.id}>
          <p>{post.content}</p>
          <button onClick={() => handleLike(post.id)}>
            ❤️ {post.likesCount}
          </button>
        </div>
      ))}
    </div>
  );
}

function AudioRoom({ roomId }) {
  const [client] = useState(() => new WebRTCClient(SERVER_URL, AUTH_TOKEN));
  const [messages, setMessages] = useState([]);

  useEffect(() => {
    client.onDataEvent = (event) => {
      if (event.type === 'chat') {
        setMessages(prev => [...prev, event.data]);
      }
    };

    client.joinRoom(roomId, 'audio');

    return () => {
      client.disconnect();
    };
  }, [roomId]);

  const sendMessage = (msg) => {
    client.sendChatMessage(msg);
  };

  return (
    <div>
      <div>
        {messages.map((msg, i) => (
          <div key={i}>{msg.from}: {msg.message}</div>
        ))}
      </div>
      <input onKeyPress={(e) => {
        if (e.key === 'Enter') {
          sendMessage(e.target.value);
          e.target.value = '';
        }
      }} />
    </div>
  );
}

*/
