# API Documentation

## Authentication

Toutes les routes protégées nécessitent un token Bearer:
```
Authorization: Bearer YOUR_TOKEN
```

## Endpoints

### WebRTC
- `POST /api/rooms` - Créer room
- `POST /api/rooms/:roomId/join` - Rejoindre room
- `POST /api/user/room/connect` - Connexion user room

### Social
- `POST /api/posts` - Créer post
- `POST /api/posts/:postId/like` - Liker post
- `POST /api/posts/:postId/comment` - Commenter

### Location
- `POST /api/location/update` - Mettre à jour position
- `POST /api/location/nearby` - Trouver utilisateurs proches

### Follow
- `POST /api/users/:userId/follow` - Follow user
- `GET /api/users/:userId/followers` - Liste followers

### Rooms Management
- `POST /api/rooms/create` - Créer room avec paramètres
- `POST /api/rooms/:roomId/join-request` - Demander à rejoindre

Voir le code pour plus de détails.
