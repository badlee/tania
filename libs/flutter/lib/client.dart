// user_channels_client.dart
// Port Dart/Flutter du Client JavaScript
//
// Dépendances requises dans pubspec.yaml:
// dependencies:
//    flutter_webrtc: ^1.2.0
//    http: ^1.6.0
//    msgpack_dart: ^1.0.1
//    localstorage: ^6.0.0

import 'dart:async';
import 'dart:convert';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:localstorage/localstorage.dart';
import 'package:msgpack_dart/msgpack_dart.dart' as msgpack;
import 'package:http/http.dart' as http;

// ==================== EVENT EMITTER ====================

typedef EventListener = void Function(dynamic data);

class EventEmitter {
  final Map<String, List<EventListener>> _events = {};

  void on(String event, EventListener listener) {
    _events.putIfAbsent(event, () => []).add(listener);
  }

  void once(String event, EventListener listener) {
    void onceWrapper(dynamic data) {
      listener(data);
      off(event, onceWrapper);
    }

    on(event, onceWrapper);
  }

  void off(String event, [EventListener? listener]) {
    if (!_events.containsKey(event)) return;

    if (listener == null) {
      _events.remove(event);
      return;
    }

    _events[event]?.remove(listener);
    if (_events[event]?.isEmpty ?? false) {
      _events.remove(event);
    }
  }

  bool emit(String event, [dynamic data]) {
    if (!_events.containsKey(event)) return false;

    final listeners = List<EventListener>.from(_events[event]!);
    for (var listener in listeners) {
      try {
        listener(data);
      } catch (e) {
        print('Error in event listener for "$event": $e');
      }
    }

    return true;
  }

  void removeAllListeners([String? event]) {
    if (event != null) {
      _events.remove(event);
    } else {
      _events.clear();
    }
  }

  int listenerCount(String event) {
    return _events[event]?.length ?? 0;
  }

  List<String> eventNames() {
    return _events.keys.toList();
  }
}

// ==================== STREAM-BASED EVENT EMITTER ====================

class StreamEventEmitter extends EventEmitter {
  final Map<String, StreamController<dynamic>> _streamControllers = {};

  /// Obtient un Stream pour un ou plusieurs types d'événements
  /// Si [events] est null, retourne un Stream de tous les événements
  Stream<Map<String, dynamic>> stream([List<String>? events]) {
    final controller = StreamController<Map<String, dynamic>>.broadcast();

    void listener(dynamic data) {
      controller.add({'event': data['event'], 'data': data['data']});
    }

    if (events == null || events.isEmpty) {
      // Stream de tous les événements
      on('*', listener);
      controller.onCancel = () => off('*', listener);
    } else {
      // Stream d'événements spécifiques
      for (var event in events) {
        on(event, (data) {
          controller.add({'event': event, 'data': data});
        });
      }
      controller.onCancel = () {
        for (var event in events) {
          off(event);
        }
      };
    }

    return controller.stream;
  }

  /// Stream typé pour un événement spécifique
  Stream<T> streamOf<T>(String event) {
    if (!_streamControllers.containsKey(event)) {
      _streamControllers[event] = StreamController<T>.broadcast();

      on(event, (data) {
        if (!_streamControllers[event]!.isClosed) {
          _streamControllers[event]!.add(data as T);
        }
      });
    }

    return (_streamControllers[event] as StreamController<T>).stream;
  }

  @override
  bool emit(String event, [dynamic data]) {
    // Emit sur l'événement wildcard pour stream()
    super.emit('*', {'event': event, 'data': data});
    return super.emit(event, data);
  }

  void dispose() {
    for (var controller in _streamControllers.values) {
      controller.close();
    }
    _streamControllers.clear();
    removeAllListeners();
  }
}

// ==================== USER CHANNELS CLIENT ====================

class Client extends StreamEventEmitter {
  final String serverUrl;
  String authToken;
  final int reconnectionTTL;

  // Auth
  static const String _tokenKey = 'user_channels_auth_token';
  static const String _recordKey = 'user_channels_auth_record';
  Map<String, dynamic>? _authRecord;

  // SSE Channel
  http.Client? _sseClient;
  StreamSubscription? _sseSubscription;
  bool _sseConnected = false;

  // WebRTC Room
  RTCPeerConnection? _peerConnection;
  RTCDataChannel? _dataChannel;
  final Map<String, Completer<dynamic>> _requestCallbacks = {};
  int _requestIdCounter = 0;
  int _reconnectionTimeToWait = 0;

  Client(this.serverUrl, {this.authToken = '', this.reconnectionTTL = 4000}) {
    _reconnectionTimeToWait = reconnectionTTL;
  }

  // ==================== AUTH METHODS ====================

  /// Login avec email et mot de passe
  Future<Map<String, dynamic>> login(String email, String password) async {
    try {
      final response = await http.post(
        Uri.parse('$serverUrl/api/collections/users/auth-with-password'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({'identity': email, 'password': password}),
      );

      if (response.statusCode != 200) {
        final error = json.decode(response.body);
        throw Exception(error['message'] ?? 'Login failed');
      }

      final data = json.decode(response.body);
      authToken = data['token'];
      _authRecord = data['record'];

      // Sauvegarder dans SharedPreferences
      await _saveAuthData(authToken, _authRecord!);

      emit('auth:login', _authRecord);
      print('✅ Login successful: ${_authRecord!['email']}');

      return data;
    } catch (e) {
      emit('auth:error', e);
      print('❌ Login failed: $e');
      rethrow;
    }
  }

  /// Inscription avec email, mot de passe et autres détails
  Future<Map<String, dynamic>> register({
    required String name,
    String? genre,
    required String email,
    String? phone,
    required String username,
    required String password,
  }) async {
    try {
      final response = await http.post(
        Uri.parse('$serverUrl/api/collections/users/records'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({
          'username': username,
          'email': email,
          'emailVisibility': true,
          'password': password,
          'passwordConfirm': password,
          'name': name,
          'genre': genre,
          'phone': phone,
        }),
      );

      if (response.statusCode >= 400) {
        final errorBody = json.decode(response.body);
        final errors = errorBody['data'] as Map<String, dynamic>?;
        if (errors != null && errors.isNotEmpty) {
          final firstError = errors.values.first['message'];
          throw Exception(firstError ?? 'Registration failed');
        }
        throw Exception(errorBody['message'] ?? 'Registration failed');
      }

      final data = json.decode(response.body);
      print('✅ Registration successful for: ${data['email']}');
      return data;
    } catch (e) {
      print('❌ Registration failed: $e');
      rethrow;
    }
  }

  /// Auto-login à partir du token sauvegardé
  Future<bool> autoLogin() async {
    try {
      await initLocalStorage();
      final savedToken = localStorage.getItem(_tokenKey);
      final savedRecordJson = localStorage.getItem(_recordKey);

      if (savedToken == null || savedRecordJson == null) {
        print('ℹ️ No saved auth data found');
        return false;
      }

      authToken = savedToken;
      _authRecord = json.decode(savedRecordJson);

      // Refresh le token pour vérifier s'il est toujours valide
      try {
        await refreshAuth();
        emit('auth:auto-login', _authRecord);
        print('✅ Auto-login successful: ${_authRecord!['email']}');
        return true;
      } catch (e) {
        print('⚠️ Saved token expired, clearing auth data');
        await logout();
        return false;
      }
    } catch (e) {
      emit('auth:error', e);
      print('❌ Auto-login failed: $e');
      return false;
    }
  }

  /// Refresh le token d'authentification
  Future<Map<String, dynamic>> refreshAuth() async {
    try {
      final response = await http.post(
        Uri.parse('$serverUrl/api/collections/users/auth-refresh'),
        headers: {
          'Authorization': 'Bearer $authToken',
          'Content-Type': 'application/json',
        },
      );

      if (response.statusCode != 200) {
        final error = json.decode(response.body);
        throw Exception(error['message'] ?? 'Auth refresh failed');
      }

      final data = json.decode(response.body);
      authToken = data['token'];
      _authRecord = data['record'];

      // Mettre à jour dans SharedPreferences
      await _saveAuthData(authToken, _authRecord!);

      emit('auth:refresh', _authRecord);
      print('✅ Auth refreshed: ${_authRecord!['email']}');

      return data;
    } catch (e) {
      emit('auth:error', e);
      print('❌ Auth refresh failed: $e');
      rethrow;
    }
  }

  /// Logout et suppression des données sauvegardées
  Future<void> logout() async {
    try {
      // Déconnecter les canaux
      disconnectSSE();
      disconnectUserRoom();

      // Supprimer les données sauvegardées
      await initLocalStorage();
      localStorage.removeItem(_tokenKey);
      localStorage.removeItem(_recordKey);

      // Réinitialiser les données locales
      authToken = '';
      _authRecord = null;

      emit('auth:logout');
      print('✅ Logout successful');
    } catch (e) {
      emit('auth:error', e);
      print('❌ Logout failed: $e');
      rethrow;
    }
  }

  /// Sauvegarder les données d'authentification
  Future<void> _saveAuthData(String token, Map<String, dynamic> record) async {
    await initLocalStorage();
    localStorage.setItem(_tokenKey, token);
    localStorage.setItem(_recordKey, json.encode(record));
  }

  /// Obtenir les informations de l'utilisateur connecté
  Map<String, dynamic>? get authRecord => _authRecord;

  /// Vérifier si l'utilisateur est connecté
  bool get isAuthenticated => authToken.isNotEmpty && _authRecord != null;

  // ==================== SSE CHANNEL ====================

  Future<void> connectSSE() async {
    try {
      _sseClient = http.Client();
      final request = http.Request(
        'GET',
        Uri.parse('$serverUrl/api/user/sse?token=$authToken'),
      );
      request.headers['Accept'] = 'text/event-stream';
      request.headers['Cache-Control'] = 'no-cache';

      final response = await _sseClient!.send(request);

      if (response.statusCode != 200) {
        throw Exception('SSE connection failed: ${response.statusCode}');
      }

      print('✅ SSE Channel connected');
      _sseConnected = true;
      emit('sse:connected');
      _reconnectionTimeToWait = reconnectionTTL;

      _sseSubscription = response.stream
          .transform(utf8.decoder)
          .transform(const LineSplitter())
          .listen(
            _handleSSELine,
            onError: _handleSSEError,
            onDone: _handleSSEDone,
          );
    } catch (e) {
      print('❌ SSE connection error: $e');
      emit('sse:error', e);
      _handleSSEError(e);
    }
  }

  String _sseBuffer = '';

  void _handleSSELine(String line) {
    if (line.isEmpty) {
      if (_sseBuffer.isNotEmpty) {
        final data = _sseBuffer.replaceFirst('data: ', '');
        try {
          final message = json.decode(data);
          _handleSSEMessage(message);
        } catch (e) {
          print('Error parsing SSE message: $e');
        }
        _sseBuffer = '';
      }
      return;
    }

    if (line.startsWith('data: ')) {
      _sseBuffer = line;
    }
  }

  void _handleSSEMessage(Map<String, dynamic> message) {
    emit('sse:message', message);
    emit('message', message);

    final requestId = message['request_id'];
    if (requestId != null && _requestCallbacks.containsKey(requestId)) {
      _requestCallbacks[requestId]!.complete(message['data']);
      _requestCallbacks.remove(requestId);
      return;
    }

    final type = message['type'] as String?;
    if (type != null) {
      emit('sse:$type', message['data']);
      emit(type, message['data']);

      switch (type) {
        case 'connected':
          print('✅ SSE ready: ${message['data']['message']}');
          break;
        case 'notification':
          _handleNotification(message['data']);
          break;
        case 'post_liked':
          _handlePostLiked(message['data']);
          break;
        case 'post_commented':
          _handlePostCommented(message['data']);
          break;
        case 'location_update':
          _handleLocationUpdate(message['data']);
          break;
        case 'geo_event':
          _handleGeoEvent(message['data']);
          break;
      }
    }
  }

  void _handleSSEError(dynamic error) {
    print('❌ SSE error: $error');
    _sseConnected = false;
    emit('sse:error', error);

    if (reconnectionTTL > 0) {
      Future.delayed(Duration(milliseconds: _reconnectionTimeToWait), () {
        disconnectSSE();
        emit('sse:reconnecting', _reconnectionTimeToWait);
        connectSSE();
      });

      _reconnectionTimeToWait =
          (_reconnectionTimeToWait + _reconnectionTimeToWait * 0.05).toInt();
    }
  }

  void _handleSSEDone() {
    print('🔌 SSE connection closed');
    _sseConnected = false;
    emit('sse:disconnected');

    if (reconnectionTTL > 0) {
      _handleSSEError('Connection closed');
    }
  }

  void disconnectSSE() {
    _sseSubscription?.cancel();
    _sseClient?.close();
    _sseClient = null;
    _sseConnected = false;
    emit('sse:disconnected');
  }

  // ==================== USER WEBRTC ROOM ====================

  Future<void> connectUserRoom() async {
    print('🔌 Connecting to user room...');
    emit('room:connecting');

    try {
      // Create peer connection
      final configuration = {
        'iceServers': [
          {'urls': 'stun:stun.l.google.com:19302'},
        ],
      };

      _peerConnection = await createPeerConnection(configuration);

      // Handle ICE candidates
      _peerConnection!.onIceCandidate = (RTCIceCandidate candidate) {
        print('🧊 ICE candidate: ${candidate.candidate}');
        emit('room:ice-candidate', candidate);
      };

      // Handle connection state changes
      _peerConnection!.onConnectionState = (RTCPeerConnectionState state) {
        print('🔗 Connection state: $state');
        emit('room:connection-state', state);

        switch (state) {
          case RTCPeerConnectionState.RTCPeerConnectionStateConnected:
            emit('room:connected');
            break;
          case RTCPeerConnectionState.RTCPeerConnectionStateDisconnected:
            emit('room:disconnected');
            break;
          case RTCPeerConnectionState.RTCPeerConnectionStateFailed:
            emit('room:failed');
            break;
          default:
            break;
        }
      };

      // Handle data channel from server
      _peerConnection!.onDataChannel = (RTCDataChannel channel) {
        print('📡 DataChannel received from server');
        _dataChannel = channel;
        _setupDataChannel();
        emit('room:datachannel-received', channel);
      };

      // Get offer from server
      final response = await http.post(
        Uri.parse('$serverUrl/api/user/room/connect'),
        headers: {
          'Authorization': 'Bearer $authToken',
          'Content-Type': 'application/json',
        },
      );

      final responseData = json.decode(response.body);
      final sdp = responseData['sdp'];

      // Set remote description
      await _peerConnection!.setRemoteDescription(
        RTCSessionDescription(sdp['sdp'], sdp['type']),
      );

      // Create answer
      final answer = await _peerConnection!.createAnswer();
      await _peerConnection!.setLocalDescription(answer);

      // Send answer to server
      await http.post(
        Uri.parse('$serverUrl/api/user/room/answer'),
        headers: {
          'Authorization': 'Bearer $authToken',
          'Content-Type': 'application/json',
        },
        body: json.encode({'sdp': answer.sdp, 'type': answer.type}),
      );

      print('✅ User room connected');
    } catch (error) {
      print('❌ Failed to connect user room: $error');
      emit('room:error', error);
      rethrow;
    }
  }

  void _setupDataChannel() {
    _dataChannel!.onDataChannelState = (RTCDataChannelState state) {
      print('📡 DataChannel state: $state');

      if (state == RTCDataChannelState.RTCDataChannelOpen) {
        print('✅ User room DataChannel opened');
        emit('datachannel:open');
      } else if (state == RTCDataChannelState.RTCDataChannelClosed) {
        print('🔌 DataChannel closed');
        emit('datachannel:close');
      }
    };

    _dataChannel!.onMessage = (RTCDataChannelMessage message) {
      try {
        final data = message.isBinary
            ? msgpack.deserialize(message.binary)
            : json.decode(message.text);

        emit('datachannel:message', data);
        _handleRoomMessage(data);
      } catch (e) {
        print('Error handling datachannel message: $e');
      }
    };
  }

  void _handleRoomMessage(Map<String, dynamic> message) {
    final type = message['type'] as String?;
    print('📨 Room Message: $type');

    if (type != null) {
      emit('room:$type', message['data']);
    }

    // Handle API response
    final requestId = message['request_id'];
    if (requestId != null && _requestCallbacks.containsKey(requestId)) {
      _requestCallbacks[requestId]!.complete(message['data']);
      _requestCallbacks.remove(requestId);
      return;
    }

    // Handle notifications/events
    switch (type) {
      case 'welcome':
        print('👋 Welcome: ${message['data']['message']}');
        break;
      case 'notification':
        _handleNotification(message['data']);
        break;
      case 'presence_change':
        _handlePresenceChange(message['data']);
        break;
      case 'location_update':
        _handleLocationUpdate(message['data']);
        break;
    }
  }

  void disconnectUserRoom() {
    _dataChannel?.close();
    _peerConnection?.close();
    emit('room:disconnected');
  }

  // ==================== API CALLS VIA WEBRTC ====================

  Future<dynamic> callAPI(
    String method,
    String endpoint, {
    Map<String, dynamic>? body,
    Map<String, dynamic>? query,
  }) async {
    if (_dataChannel == null ||
        _dataChannel!.state != RTCDataChannelState.RTCDataChannelOpen) {
      final error = Exception('User room not connected');
      emit('api:error', error);
      throw error;
    }

    final requestId =
        'req_${++_requestIdCounter}_${DateTime.now().millisecondsSinceEpoch}';

    final request = {
      'request_id': requestId,
      'method': method,
      'endpoint': endpoint,
      'body': body,
      'query': query,
    };

    emit('api:request', {
      'method': method,
      'endpoint': endpoint,
      'requestId': requestId,
    });

    // Send request
    final encoded = msgpack.serialize(request);
    await _dataChannel!.send(RTCDataChannelMessage.fromBinary(encoded));

    // Wait for response
    final completer = Completer<dynamic>();
    _requestCallbacks[requestId] = completer;

    // Timeout
    Future.delayed(const Duration(seconds: 30), () {
      if (!completer.isCompleted) {
        _requestCallbacks.remove(requestId);
        emit('api:timeout', {
          'requestId': requestId,
          'method': method,
          'endpoint': endpoint,
        });
        completer.completeError(Exception('Request timeout'));
      }
    });

    try {
      final result = await completer.future;
      emit('api:response', {
        'data': result,
        'requestId': requestId,
        'method': method,
        'endpoint': endpoint,
      });
      return result;
    } catch (e) {
      emit('api:error', {
        'error': e,
        'requestId': requestId,
        'method': method,
        'endpoint': endpoint,
      });
      rethrow;
    }
  }

  // ==================== CONVENIENCE METHODS ====================

  // Location
  Future<dynamic> updateLocation(
    double lat,
    double lng, {
    double accuracy = 10,
    String presence = 'online',
  }) async {
    final result = await callAPI(
      'POST',
      '/location/update',
      body: {
        'location': {
          'point': {'lat': lat, 'lng': lng},
          'accuracy': accuracy,
          'altitude': 0,
          'speed': 0,
          'heading': 0,
        },
        'presence': presence,
      },
    );
    emit('location:updated', {'lat': lat, 'lng': lng, 'presence': presence});
    return result;
  }

  Future<dynamic> findNearby(double lat, double lng, double radius) async {
    final result = await callAPI(
      'POST',
      '/location/nearby',
      body: {
        'point': {'lat': lat, 'lng': lng},
        'radius': radius,
      },
    );
    emit('location:nearby-found', result);
    return result;
  }

  Future<dynamic> findInPolygon(List<Map<String, double>> polygon) async {
    final result = await callAPI(
      'POST',
      '/location/polygon',
      body: {'polygon': polygon},
    );
    emit('location:polygon-found', result);
    return result;
  }

  // Posts
  Future<dynamic> getPosts({String filter = 'isPublic = true'}) async {
    final result = await callAPI('GET', '/posts', query: {'filter': filter});
    emit('posts:fetched', result);
    return result;
  }

  Future<dynamic> createPost(
    String content, {
    String type = 'html',
    bool isPublic = true,
  }) async {
    final result = await callAPI(
      'POST',
      '/posts',
      body: {'content': content, 'type': type, 'isPublic': isPublic},
    );
    emit('posts:created', result);
    return result;
  }

  Future<dynamic> likePost(String postId, {String reaction = 'like'}) async {
    final result = await callAPI(
      'POST',
      '/posts/like',
      body: {'post_id': postId, 'reaction': reaction},
    );
    emit('posts:liked', {'postId': postId, 'reaction': reaction});
    return result;
  }

  Future<dynamic> commentPost(String postId, String content) async {
    final result = await callAPI(
      'POST',
      '/posts/comment',
      body: {'post_id': postId, 'content': content},
    );
    emit('posts:commented', {'postId': postId, 'content': content});
    return result;
  }

  // Marketplace
  Future<dynamic> getArticles({String filter = 'quantite > 0'}) async {
    final result = await callAPI('GET', '/articles', query: {'filter': filter});
    emit('articles:fetched', result);
    return result;
  }

  Future<dynamic> buyArticle(String articleId) async {
    final result = await callAPI(
      'POST',
      '/articles/buy',
      body: {'article_id': articleId},
    );
    emit('articles:bought', {'articleId': articleId, 'result': result});
    return result;
  }

  // Presence
  Future<dynamic> updatePresence(String presence) async {
    final result = await callAPI(
      'POST',
      '/presence/update',
      body: {'presence': presence},
    );
    emit('presence:updated', presence);
    return result;
  }

  // Rooms
  Future<dynamic> joinRoom(String roomId) async {
    final result = await callAPI(
      'POST',
      '/rooms/join',
      body: {'room_id': roomId},
    );
    emit('rooms:joined', roomId);
    return result;
  }

  Future<dynamic> leaveRoom(String roomId) async {
    final result = await callAPI(
      'POST',
      '/rooms/leave',
      body: {'room_id': roomId},
    );
    emit('rooms:left', roomId);
    return result;
  }

  // ==================== EVENT HANDLERS ====================

  void _handleNotification(dynamic data) {
    print('🔔 Notification: $data');
    emit('notification', data);
  }

  void _handlePostLiked(dynamic data) {
    print('❤️ Your post was liked: $data');
    emit('post:liked', data);
  }

  void _handlePostCommented(dynamic data) {
    print('💬 New comment on your post: $data');
    emit('post:commented', data);
  }

  void _handleLocationUpdate(dynamic data) {
    print('📍 Location update: $data');
    emit('location:update', data);
  }

  void _handleGeoEvent(dynamic data) {
    print('🎯 Geo event: $data');
    emit('geo:event', data);
  }

  void _handlePresenceChange(dynamic data) {
    print('👤 Presence changed: $data');
    emit('presence:change', data);
  }

  // ==================== CLEANUP ====================

  @override
  void dispose() {
    disconnectSSE();
    disconnectUserRoom();
    _requestCallbacks.clear();
    super.dispose();
  }
}

// ==================== USAGE EXAMPLES ====================

/*
void main() async {
  // ==================== Initialisation ====================
  
  final client = Client(
    'http://localhost:8090',
    authToken: '', // Vide au départ
  );

  // ==================== Auto-login au démarrage ====================
  
  final autoLoggedIn = await client.autoLogin();
  if (autoLoggedIn) {
    print('User auto-logged in: ${client.authRecord!['email']}');
    // Connecter les canaux
    await client.connectSSE();
    await client.connectUserRoom();
  } else {
    print('No saved session, showing login screen');
  }

  // ==================== Login manuel ====================
  
  try {
    final result = await client.login('user@example.com', 'password123');
    print('Login successful!');
    print('Token: ${result['token']}');
    print('User: ${result['record']['email']}');
    
    // Connecter les canaux après le login
    await client.connectSSE();
    await client.connectUserRoom();
  } catch (e) {
    print('Login failed: $e');
  }

  // ==================== Écouter les événements d'auth ====================
  
  client.on('auth:login', (data) {
    print('✅ User logged in: ${data['email']}');
  });

  client.on('auth:auto-login', (data) {
    print('✅ User auto-logged in: ${data['email']}');
  });

  client.on('auth:refresh', (data) {
    print('🔄 Token refreshed for: ${data['email']}');
  });

  client.on('auth:logout', (data) {
    print('👋 User logged out');
  });

  client.on('auth:error', (error) {
    print('❌ Auth error: $error');
  });

  // ==================== Refresh token ====================
  
  try {
    await client.refreshAuth();
    print('Token refreshed successfully');
  } catch (e) {
    print('Token refresh failed: $e');
  }

  // ==================== Logout ====================
  
  await client.logout();
  print('User logged out');

  // ==================== Utilisation dans Flutter ====================

  // Dans un StatefulWidget
  late Client client;
  StreamSubscription? _notificationSubscription;

  @override
  void initState() {
    super.initState();
    
    client = Client(
      serverUrl: 'http://localhost:8090',
      authToken: '',
    );
    
    // Auto-login au démarrage
    _initAuth();
    
    _notificationSubscription = client
        .streamOf<Map<String, dynamic>>('notification')
        .listen((data) {
      setState(() {
        // Mettre à jour l'UI
      });
    });
  }

  Future<void> _initAuth() async {
    final loggedIn = await client.autoLogin();
    if (loggedIn) {
      await client.connectSSE();
      await client.connectUserRoom();
    }
  }

  Future<void> _login() async {
    try {
      await client.login('user@example.com', 'password');
      await client.connectSSE();
      await client.connectUserRoom();
      setState(() {});
    } catch (e) {
      // Afficher erreur
    }
  }

  Future<void> _logout() async {
    await client.logout();
    setState(() {});
  }

  // ==================== Écouter avec des callbacks ====================
  
  client.on('sse:connected', (data) {
    print('SSE connected!');
  });

  client.on('notification', (data) {
    print('📬 Notification: $data');
  });

  client.on('post:liked', (data) {
    print('❤️ Someone liked your post: $data');
  });

  // ==================== Écouter avec des Streams ====================

  // Stream de tous les événements
  client.stream().listen((event) {
    print('Event: ${event['event']} - Data: ${event['data']}');
  });

  // Stream d'événements spécifiques
  client.stream(['notification', 'post:liked', 'location:update']).listen((event) {
    print('Filtered event: ${event['event']} - ${event['data']}');
  });

  // Stream typé pour un événement spécifique
  client.streamOf<Map<String, dynamic>>('notification').listen((data) {
    print('Notification via stream: $data');
  });

  // ==================== Utilisation dans Flutter ====================

  // Dans un StatefulWidget
  StreamSubscription? _notificationSubscription;

  @override
  void initState() {
    super.initState();
    
    _notificationSubscription = client
        .streamOf<Map<String, dynamic>>('notification')
        .listen((data) {
      setState(() {
        // Mettre à jour l'UI
      });
    });
  }

  @override
  void dispose() {
    _notificationSubscription?.cancel();
    client.dispose();
    super.dispose();
  }

  // ==================== Avec StreamBuilder ====================

  Widget build(BuildContext context) {
    // Vérifier l'authentification
    if (!client.isAuthenticated) {
      return LoginScreen(client: client);
    }
    
    return StreamBuilder<Map<String, dynamic>>(
      stream: client.streamOf('notification'),
      builder: (context, snapshot) {
        if (snapshot.hasData) {
          return Text('Notification: ${snapshot.data}');
        }
        return Text('No notifications');
      },
    );
  }

  // ==================== Widget de Login ====================
  
  class LoginScreen extends StatefulWidget {
    final Client client;
    
    const LoginScreen({required this.client});
    
    @override
    _LoginScreenState createState() => _LoginScreenState();
  }
  
  class _LoginScreenState extends State<LoginScreen> {
    final _emailController = TextEditingController();
    final _passwordController = TextEditingController();
    bool _isLoading = false;
    String? _error;
    
    @override
    Widget build(BuildContext context) {
      return Scaffold(
        appBar: AppBar(title: Text('Login')),
        body: Padding(
          padding: EdgeInsets.all(16),
          child: Column(
            children: [
              TextField(
                controller: _emailController,
                decoration: InputDecoration(labelText: 'Email'),
                keyboardType: TextInputType.emailAddress,
              ),
              TextField(
                controller: _passwordController,
                decoration: InputDecoration(labelText: 'Password'),
                obscureText: true,
              ),
              if (_error != null)
                Text(_error!, style: TextStyle(color: Colors.red)),
              SizedBox(height: 20),
              _isLoading
                  ? CircularProgressIndicator()
                  : ElevatedButton(
                      onPressed: _login,
                      child: Text('Login'),
                    ),
            ],
          ),
        ),
      );
    }
    
    Future<void> _login() async {
      setState(() {
        _isLoading = true;
        _error = null;
      });
      
      try {
        await widget.client.login(
          _emailController.text,
          _passwordController.text,
        );
        
        // Connecter les canaux
        await widget.client.connectSSE();
        await widget.client.connectUserRoom();
        
        // Navigation vers l'écran principal
        if (mounted) {
          Navigator.of(context).pushReplacement(
            MaterialPageRoute(builder: (_) => HomeScreen()),
          );
        }
      } catch (e) {
        setState(() {
          _error = e.toString();
        });
      } finally {
        setState(() {
          _isLoading = false;
        });
      }
    }
    
    @override
    void dispose() {
      _emailController.dispose();
      _passwordController.dispose();
      super.dispose();
    }
  }

  // ==================== Connecter ====================

  await client.connectSSE();
  await client.connectUserRoom();

  // ==================== Appeler l'API ====================

  try {
    final posts = await client.getPosts();
    print('Posts: $posts');

    await client.updateLocation(48.8566, 2.3522, presence: 'online');
    
    await client.createPost('Hello from Flutter!', isPublic: true);
  } catch (e) {
    print('Error: $e');
  }

  // ==================== Cleanup ====================

  client.dispose();
}
*/
