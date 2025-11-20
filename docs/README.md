# WebRTC Social Server

Serveur complet en pure Go avec:
- ✅ WebRTC (audio/video/data)
- ✅ Système social (posts, likes, comments)
- ✅ Marketplace intégré
- ✅ Géolocalisation temps réel
- ✅ Follow/Followers avec paiements
- ✅ Gestion de rooms avancée
- ✅ Interpréteur TypeScript (Goja)
- ✅ PocketBase 0.33.0

## Installation

```bash
# Installer les dépendances
go mod download

# Compiler
go build -o server main.go

# Lancer
./server serve
```

## Utilisation

Le serveur démarre sur http://localhost:8090

- Admin UI: http://localhost:8090/_/
- API: http://localhost:8090/api/
- SSE: http://localhost:8090/api/user/sse

## Documentation

Voir docs/API.md pour la documentation complète de l'API.
