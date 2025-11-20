#!/bin/bash

# ============================================
# WebRTC Social Server - Project Generator
# ============================================

set -e

PROJECT_NAME="webrtc-social-server"
BOLD='\033[1m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BOLD}${BLUE}"
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                                                            ║"
echo "║   WebRTC Social Server - Project Generator                ║"
echo "║   Version 1.0.0                                            ║"
echo "║                                                            ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if project already exists
if [ -d "$PROJECT_NAME" ]; then
    echo -e "${YELLOW}⚠️  Directory $PROJECT_NAME already exists!${NC}"
    read -p "Do you want to overwrite it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi
    rm -rf "$PROJECT_NAME"
fi

echo -e "${GREEN}📁 Creating project structure...${NC}"

# Create directory structure
mkdir -p "$PROJECT_NAME"/{pb_hooks,pb_modules,client,docs}

cd "$PROJECT_NAME"

# ============================================
# Create go.mod
# ============================================
echo -e "${GREEN}📦 Creating go.mod...${NC}"

cat > go.mod << 'EOF'
module webrtc-social-server

go 1.21

require (
	github.com/labstack/echo/v5 v5.0.0-20230722203903-ec5b858dab61
	github.com/pocketbase/pocketbase v0.33.0
	github.com/pion/webrtc/v3 v3.2.40
	github.com/vmihailenco/msgpack/v5 v5.4.1
	github.com/dop251/goja v0.0.0-20240220182346-e401ed450204
	github.com/dop251/goja_nodejs v0.0.0-20240221231712-27e3e9c9c89c
	github.com/fsnotify/fsnotify v1.7.0
)
EOF

# ============================================
# Create .gitignore
# ============================================
echo -e "${GREEN}📝 Creating .gitignore...${NC}"

cat > .gitignore << 'EOF'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
server
webrtc-social-server

# Test binary
*.test

# Output
*.out

# Dependencies
vendor/

# PocketBase data
pb_data/
pb_migrations/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Environment
.env
.env.local

# Node modules (if any)
node_modules/
EOF

# ============================================
# Create pb_modules/counter.js
# ============================================
echo -e "${GREEN}📦 Creating shared modules...${NC}"

cat > pb_modules/counter.js << 'EOF'
// Shared counter module
let i = 0;
const history = [];

function inc() {
  i++;
  history.push({ value: i, timestamp: Date.now() });
  return i;
}

function dec() {
  i--;
  history.push({ value: i, timestamp: Date.now() });
  return i;
}

function get() {
  return i;
}

function reset() {
  const old = i;
  i = 0;
  history.push({ value: i, timestamp: Date.now(), reset: true });
  return old;
}

function getHistory() {
  return history;
}

exports.inc = inc;
exports.dec = dec;
exports.get = get;
exports.reset = reset;
exports.getHistory = getHistory;
EOF

cat > pb_modules/logger.js << 'EOF'
// Shared logger module
const logs = [];

function info(message, ...args) {
  const entry = {
    level: "INFO",
    message: message,
    args: args,
    timestamp: new Date().toISOString()
  };
  logs.push(entry);
  log("[INFO]", message, ...args);
}

function warn(message, ...args) {
  const entry = {
    level: "WARN",
    message: message,
    args: args,
    timestamp: new Date().toISOString()
  };
  logs.push(entry);
  log("[WARN]", message, ...args);
}

function error(message, ...args) {
  const entry = {
    level: "ERROR",
    message: message,
    args: args,
    timestamp: new Date().toISOString()
  };
  logs.push(entry);
  log("[ERROR]", message, ...args);
}

function getLogs(level) {
  if (level) {
    return logs.filter(l => l.level === level);
  }
  return logs;
}

function clear() {
  logs.length = 0;
}

exports.info = info;
exports.warn = warn;
exports.error = error;
exports.getLogs = getLogs;
exports.clear = clear;
EOF

cat > pb_modules/cache.js << 'EOF'
// Shared cache module
const cache = {};
const expirations = {};

function set(key, value, ttl) {
  cache[key] = value;
  
  if (ttl) {
    expirations[key] = Date.now() + (ttl * 1000);
    
    setTimeout(() => {
      if (expirations[key] && Date.now() >= expirations[key]) {
        delete cache[key];
        delete expirations[key];
      }
    }, ttl * 1000);
  }
  
  return true;
}

function get(key) {
  if (expirations[key] && Date.now() >= expirations[key]) {
    delete cache[key];
    delete expirations[key];
    return null;
  }
  
  return cache[key];
}

function has(key) {
  return get(key) !== undefined && get(key) !== null;
}

function del(key) {
  delete cache[key];
  delete expirations[key];
  return true;
}

function clear() {
  Object.keys(cache).forEach(k => delete cache[k]);
  Object.keys(expirations).forEach(k => delete expirations[k]);
}

function keys() {
  return Object.keys(cache);
}

function size() {
  return Object.keys(cache).length;
}

exports.set = set;
exports.get = get;
exports.has = has;
exports.del = del;
exports.clear = clear;
exports.keys = keys;
exports.size = size;
EOF

# ============================================
# Create pb_hooks (example hook)
# ============================================
echo -e "${GREEN}🎣 Creating hooks...${NC}"

cat > pb_hooks/on-post-create.js << 'EOF'
// Hook: Auto-run on post creation events
const counter = require("counter");
const logger = require("logger");
const cache = require("cache");

function main() {
  logger.info("Post creation hook initialized");
  
  // Subscribe to post events
  pubsub.subscribe("post_events", function(eventData) {
    const event = utils.jsonDecode(eventData);
    
    if (event.type === "new_post") {
      handleNewPost(event.post_id, event.user_id);
    }
  });
  
  logger.info("Listening for new posts...");
}

function handleNewPost(postId, userId) {
  logger.info("New post detected:", postId);
  
  // Increment global counter
  const count = counter.inc();
  logger.info(`Total posts created: ${count}`);
  
  // Cache for analytics
  cache.set(`post:${postId}`, {
    id: postId,
    user: userId,
    created: Date.now()
  }, 3600);
  
  // Milestone rewards
  if (count % 10 === 0) {
    logger.info(`🎉 Milestone: ${count} posts!`);
    
    db.create("operations", {
      user: userId,
      montant: 50,
      operation: "cashin",
      desc: `Bonus: ${count}th community post!`,
      status: "paye"
    });
    
    pubsub.publish("notifications", {
      type: "milestone",
      user_id: userId,
      count: count,
      reward: 50
    });
  }
}

main();
EOF

# ============================================
# Create README.md
# ============================================
echo -e "${GREEN}📚 Creating documentation...${NC}"

cat > docs/README.md << 'EOF'
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
EOF

cat > docs/API.md << 'EOF'
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
EOF

# ============================================
# Create main.go notice
# ============================================
echo -e "${GREEN}⚠️  Creating main.go placeholder...${NC}"

cat > main.go << 'EOF'
package main

import (
	"log"
)

func main() {
	log.Println("⚠️  IMPORTANT: You need to copy the full main.go content from the artifacts!")
	log.Println("")
	log.Println("The main.go file contains ~3000+ lines of code from multiple artifacts:")
	log.Println("  1. webrtc_social_server")
	log.Println("  2. user_dedicated_channels")
	log.Println("  3. geo_location_system")
	log.Println("  4. follow_room_management")
	log.Println("  5. ts_interpreter")
	log.Println("")
	log.Println("Please merge all these artifacts into this main.go file.")
	log.Println("Then run: go mod download && go build -o server main.go")
}
EOF

# ============================================
# Create build script
# ============================================
echo -e "${GREEN}🔨 Creating build script...${NC}"

cat > build.sh << 'EOF'
#!/bin/bash

echo "🔨 Building WebRTC Social Server..."

# Check if main.go has real content
if grep -q "IMPORTANT: You need to copy" main.go; then
    echo "❌ Error: main.go is a placeholder!"
    echo ""
    echo "Please copy the full main.go content from the artifacts first."
    echo "See main.go for instructions."
    exit 1
fi

echo "📦 Downloading dependencies..."
go mod download

echo "🏗️  Compiling..."
go build -o server main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo ""
    echo "Run with: ./server serve"
else
    echo "❌ Build failed!"
    exit 1
fi
EOF

chmod +x build.sh

# ============================================
# Create run script
# ============================================
cat > run.sh << 'EOF'
#!/bin/bash

if [ ! -f "server" ]; then
    echo "❌ Server binary not found!"
    echo "Run ./build.sh first"
    exit 1
fi

echo "🚀 Starting WebRTC Social Server..."
echo ""
echo "📡 Server will be available at:"
echo "   - API: http://localhost:8090/api/"
echo "   - Admin: http://localhost:8090/_/"
echo ""

./server serve
EOF

chmod +x run.sh

# ============================================
# Create INSTRUCTIONS.md
# ============================================
cat > INSTRUCTIONS.md << 'EOF'
# 🚀 Installation Instructions

## ⚠️ IMPORTANT: Complete main.go

The generated `main.go` is a placeholder. You need to copy the full code from these artifacts:

1. **webrtc_social_server** - Core server, WebRTC, Social, Marketplace
2. **user_dedicated_channels** - SSE & User Room WebRTC
3. **geo_location_system** - Geolocation & Geofencing
4. **follow_room_management** - Follow/Followers & Room Management
5. **ts_interpreter** - TypeScript/JavaScript interpreter

### How to merge artifacts:

```go
package main

import (
    // All imports from all artifacts
    "encoding/json"
    "log"
    "sync"
    "time"
    // ... etc
)

// Copy all type definitions from all artifacts
// Copy all global variables
// Copy all functions
// Merge the main() function
```

## 📦 After copying main.go:

```bash
# 1. Download dependencies
go mod download

# 2. Build
./build.sh

# 3. Run
./run.sh
```

## 📁 Project Structure

```
webrtc-social-server/
├── main.go              # ⚠️ Copy full code here!
├── go.mod               # ✅ Ready
├── build.sh             # ✅ Ready
├── run.sh               # ✅ Ready
├── pb_hooks/            # ✅ Ready (example hook included)
│   └── on-post-create.js
├── pb_modules/          # ✅ Ready
│   ├── counter.js
│   ├── logger.js
│   └── cache.js
├── client/              # Add your client code here
└── docs/                # ✅ Documentation
    ├── README.md
    └── API.md
```

## 🎯 Next Steps

1. ✅ Project structure created
2. ⚠️  Copy main.go content from artifacts
3. ⚠️  Run `go mod download`
4. ⚠️  Run `./build.sh`
5. ⚠️  Run `./run.sh`

## 📚 Documentation

- See `docs/README.md` for overview
- See `docs/API.md` for API documentation
- See `INSTRUCTIONS.md` (this file) for setup

## 🆘 Troubleshooting

**Error: "main.go is a placeholder"**
→ Copy the full main.go code from the artifacts

**Error: package not found**
→ Run `go mod download`

**Error: port already in use**
→ Change port in main.go or stop other services on port 8090

## 🎉 Done!

Once main.go is copied and built, your server will have:
- WebRTC audio/video/data rooms
- Social network (posts, likes, comments)
- Marketplace with payments
- Real-time geolocation
- Follow/Followers system
- Advanced room management
- TypeScript/JS interpreter
- And more!
EOF

# ============================================
# Summary
# ============================================
echo ""
echo -e "${BOLD}${GREEN}✅ Project generated successfully!${NC}"
echo ""
echo -e "${BOLD}📁 Project: ${BLUE}$PROJECT_NAME${NC}"
echo ""
echo -e "${YELLOW}⚠️  IMPORTANT: Read INSTRUCTIONS.md${NC}"
echo ""
echo -e "Next steps:"
echo -e "  1. ${BOLD}cd $PROJECT_NAME${NC}"
echo -e "  2. ${BOLD}cat INSTRUCTIONS.md${NC}  (read carefully!)"
echo -e "  3. Copy the full main.go code from the artifacts"
echo -e "  4. ${BOLD}./build.sh${NC}"
echo -e "  5. ${BOLD}./run.sh${NC}"
echo ""
echo -e "${GREEN}🎉 Happy coding!${NC}"
echo ""
