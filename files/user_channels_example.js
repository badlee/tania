import UserChannelsClient from "./user_channels_client";

// ==================== USAGE EXAMPLES ====================

async function example() {
  const client = new UserChannelsClient('http://localhost:8090', 'YOUR_TOKEN');

  // 1. Connect SSE (for half-duplex communication)
  client.connectSSE();

  // Listen for specific events
  client.onSSE('post_created', (data) => {
    console.log('New post created:', data);
  });

  client.onSSE('notification', (data) => {
    console.log('Notification received:', data);
  });

  // 2. Connect User Room (for full-duplex communication)
  await client.connectUserRoom();

  // 3. Use API via WebRTC DataChannel
  try {
    // Update location
    const locationResult = await client.updateLocation(48.8566, 2.3522, 10, 'online');
    console.log('Location updated:', locationResult);

    // Find nearby users
    const nearby = await client.findNearby(48.8566, 2.3522, 1000);
    console.log('Nearby users:', nearby);

    // Create post
    const post = await client.createPost('Hello from WebRTC!', 'html', true);
    console.log('Post created:', post);

    // Like a post
    await client.likePost(post.post_id, 'fire');

    // Comment on post
    await client.commentPost(post.post_id, 'Great post!');

    // Get articles
    const articles = await client.getArticles();
    console.log('Articles:', articles);

    // Buy article
    if (articles.articles.length > 0) {
      await client.buyArticle(articles.articles[0].id);
    }

    // Update presence
    await client.updatePresence('away');

  } catch (error) {
    console.error('API error:', error);
  }
}

// ==================== REST API WITH respond_to=sse ====================

async function exampleRestWithSSE() {
  const client = new UserChannelsClient('http://localhost:8090', 'YOUR_TOKEN');
  
  // Connect SSE first
  client.connectSSE();

  // Make REST request with respond_to=sse
  const requestId = `req_${Date.now()}`;
  
  // Setup callback for response
  const responsePromise = new Promise((resolve) => {
    client.requestCallbacks.set(requestId, (error, data) => {
      resolve(data);
    });
  });

  // Make REST request
  fetch('http://localhost:8090/api/rooms?respond_to=sse', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer YOUR_TOKEN',
      'Content-Type': 'application/json',
      'X-Request-ID': requestId
    },
    body: JSON.stringify({
      room_type: 'audio',
      name: 'Test Room'
    })
  });

  // Response will arrive via SSE
  const response = await responsePromise;
  console.log('Response via SSE:', response);
}

// ==================== REACT HOOK ====================

function useUserChannels(authToken) {
  const [client, setClient] = useState(null);
  const [sseConnected, setSseConnected] = useState(false);
  const [roomConnected, setRoomConnected] = useState(false);

  useEffect(() => {
    const c = new UserChannelsClient('http://localhost:8090', authToken);
    
    // Connect SSE
    c.connectSSE();
    setSseConnected(true);

    // Connect User Room
    c.connectUserRoom().then(() => {
      setRoomConnected(true);
    });

    setClient(c);

    return () => {
      c.disconnectSSE();
      c.disconnectUserRoom();
    };
  }, [authToken]);

  return { client, sseConnected, roomConnected };
}

// Usage in React:
function MyComponent() {
  const { client, sseConnected, roomConnected } = useUserChannels(authToken);

  useEffect(() => {
    if (!client) return;

    // Listen for notifications
    client.onSSE('notification', (data) => {
      // Show notification in UI
    });
  }, [client]);

  const handleCreatePost = async () => {
    if (client && roomConnected) {
      const result = await client.createPost('My post', 'html', true);
      console.log('Post created:', result);
    }
  };

  return (
    <div>
      <div>SSE: {sseConnected ? '✅' : '❌'}</div>
      <div>Room: {roomConnected ? '✅' : '❌'}</div>
      <button onClick={handleCreatePost}>Create Post</button>
    </div>
  );
}

// ==================== GEOLOCATION TRACKING ====================

function startLocationTracking(client) {
  if (!navigator.geolocation) {
    console.error('Geolocation not supported');
    return;
  }

  navigator.geolocation.watchPosition(
    async (position) => {
      await client.updateLocation(
        position.coords.latitude,
        position.coords.longitude,
        position.coords.accuracy,
        'online'
      );
    },
    (error) => {
      console.error('Geolocation error:', error);
    },
    {
      enableHighAccuracy: true,
      maximumAge: 10000,
      timeout: 5000
    }
  );
}

// ==================== PRESENCE MANAGEMENT ====================

function setupPresenceManagement(client) {
  // Online when tab is visible
  document.addEventListener('visibilitychange', async () => {
    if (document.hidden) {
      await client.updatePresence('away');
    } else {
      await client.updatePresence('online');
    }
  });

  // Offline when page unloads
  window.addEventListener('beforeunload', () => {
    client.updatePresence('offline');
  });

  // Idle detection
  let idleTimeout;
  const resetIdle = () => {
    clearTimeout(idleTimeout);
    client.updatePresence('online');
    
    idleTimeout = setTimeout(() => {
      client.updatePresence('away');
    }, 5 * 60 * 1000); // 5 minutes
  };

  ['mousedown', 'keydown', 'scroll', 'touchstart'].forEach(event => {
    document.addEventListener(event, resetIdle, true);
  });

  resetIdle();
}
