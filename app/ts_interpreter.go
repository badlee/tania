package app

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/fsnotify/fsnotify"
	"github.com/pocketbase/pocketbase/core"

	msgpack "github.com/vmihailenco/msgpack/v5"
)

// ==================== SHARED MODULE SYSTEM ====================

// ModuleRegistry - Espace partagé pour tous les modules
type ModuleRegistry struct {
	modules map[string]*goja.Object
	mu      sync.RWMutex
}

var SharedModules = &ModuleRegistry{
	modules: make(map[string]*goja.Object),
}

func (mr *ModuleRegistry) Get(name string) (*goja.Object, bool) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	mod, exists := mr.modules[name]
	return mod, exists
}

func (mr *ModuleRegistry) Set(name string, module *goja.Object) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.modules[name] = module
}

func (mr *ModuleRegistry) Delete(name string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	delete(mr.modules, name)
}

// ==================== SCRIPT CONTEXT ====================

type ScriptContext struct {
	ID        string
	Name      string
	FilePath  string
	vm        *goja.Runtime
	app       core.App
	exports   *goja.Object
	isRunning bool
	stopChan  chan bool
	mu        sync.RWMutex
}

// ==================== SCRIPT MANAGER ====================

type ScriptManager struct {
	sync.RWMutex
	app        core.App
	scriptsDir string
	modulesDir string
	contexts   map[string]*ScriptContext
	watcher    *fsnotify.Watcher
}

func NewScriptManager(app core.App, scriptsDir, modulesDir string) (*ScriptManager, error) {
	// Créer les dossiers si nécessaire
	os.MkdirAll(scriptsDir, 0755)
	os.MkdirAll(modulesDir, 0755)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	sm := &ScriptManager{
		app:        app,
		scriptsDir: scriptsDir,
		modulesDir: modulesDir,
		contexts:   make(map[string]*ScriptContext),
		watcher:    watcher,
	}

	// Watch les dossiers
	watcher.Add(scriptsDir)
	watcher.Add(modulesDir)

	return sm, nil
}

func (sm *ScriptManager) Get(name string) (context *ScriptContext, found bool) {
	context, found = sm.contexts[name]
	return
}

// Charger tous les scripts au démarrage
func (sm *ScriptManager) LoadAll() error {
	log.Println("📦 Loading scripts from:", sm.scriptsDir)

	// Charger d'abord les modules partagés
	if err := sm.loadSharedModules(); err != nil {
		log.Println("⚠️  Error loading modules:", err)
	}

	// Charger les scripts hooks
	err := filepath.WalkDir(sm.scriptsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && (strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".ts")) {
			log.Printf("📜 Loading script: %s", filepath.Base(path))
			if err := sm.LoadScript(path); err != nil {
				log.Printf("❌ Failed to load %s: %v", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Démarrer le file watcher
	go sm.watchFiles()

	log.Printf("✅ Loaded %d scripts", len(sm.contexts))
	return nil
}

// Charger les modules partagés
func (sm *ScriptManager) loadSharedModules() error {
	log.Println("📚 Loading shared modules from:", sm.modulesDir)

	return filepath.WalkDir(sm.modulesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && (strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".ts")) {
			moduleName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			log.Printf("  📦 Loading module: %s", moduleName)

			code, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Créer une VM temporaire pour initialiser le module
			vm := goja.New()
			vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

			// Exécuter le code du module
			_, err = vm.RunString(string(code))
			if err != nil {
				log.Printf("  ❌ Failed to initialize module %s: %v", moduleName, err)
				return nil
			}

			// Récupérer les exports
			exportsVal := vm.Get("exports")
			if exportsVal == nil || goja.IsUndefined(exportsVal) {
				log.Printf("  ⚠️  Module %s has no exports", moduleName)
				return nil
			}

			// Stocker dans le registry partagé
			if exportsObj, ok := exportsVal.(*goja.Object); ok {
				SharedModules.Set(moduleName, exportsObj)
				log.Printf("  ✅ Module %s loaded and shared", moduleName)
			}
		}

		return nil
	})
}

// Charger un script individuel
func (sm *ScriptManager) LoadScript(filePath string) error {
	code, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	scriptName := filepath.Base(filePath)
	scriptID := strings.TrimSuffix(scriptName, filepath.Ext(scriptName))

	// Créer un nouveau contexte
	ctx := &ScriptContext{
		ID:       scriptID,
		Name:     scriptName,
		FilePath: filePath,
		vm:       goja.New(),
		app:      sm.app,
		stopChan: make(chan bool),
	}

	// Configurer la VM
	ctx.vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// Setup Node.js modules
	registry := require.NewRegistry(require.WithGlobalFolders("."))
	registry.Enable(ctx.vm)
	console.Enable(ctx.vm)

	// Exposer toutes les APIs
	sm.exposeAPIs(ctx)

	// Exposer la fonction require pour les modules partagés
	ctx.vm.Set("require", func(moduleName string) goja.Value {
		return sm.requireModule(ctx, moduleName)
	})

	// Créer l'objet exports
	ctx.exports = ctx.vm.NewObject()
	ctx.vm.Set("exports", ctx.exports)
	ctx.vm.Set("module", map[string]interface{}{
		"exports": ctx.exports,
	})

	// Exécuter le script
	_, err = ctx.vm.RunString(string(code))
	if err != nil {
		return fmt.Errorf("script execution error: %v", err)
	}

	// Stocker le contexte
	sm.Lock()
	sm.contexts[scriptID] = ctx
	sm.Unlock()

	// Démarrer le script s'il a une fonction main
	go sm.runScriptBackground(ctx)

	log.Printf("✅ Script loaded: %s", scriptName)
	return nil
}

// Require un module partagé
func (sm *ScriptManager) requireModule(ctx *ScriptContext, moduleName string) goja.Value {
	// Vérifier si le module existe dans le registry partagé
	module, exists := SharedModules.Get(moduleName)
	if !exists {
		// Essayer de charger depuis le dossier modules
		modulePath := filepath.Join(sm.modulesDir, moduleName+".js")
		if _, err := os.Stat(modulePath); err == nil {
			sm.loadSharedModules() // Recharger les modules
			module, exists = SharedModules.Get(moduleName)
		}

		if !exists {
			ctx.vm.RunString(fmt.Sprintf(`throw new Error("Module '%s' not found")`, moduleName))
			return goja.Undefined()
		}
	}

	// Retourner le module partagé (pas une copie!)
	return ctx.vm.ToValue(module)
}

// Exécuter le script en arrière-plan
func (sm *ScriptManager) runScriptBackground(ctx *ScriptContext) {
	ctx.mu.Lock()
	ctx.isRunning = true
	ctx.mu.Unlock()

	defer func() {
		ctx.mu.Lock()
		ctx.isRunning = false
		ctx.mu.Unlock()

		if r := recover(); r != nil {
			log.Printf("❌ Script %s panicked: %v", ctx.Name, r)
		}
	}()

	// Vérifier si le script a une fonction main
	mainFunc := ctx.vm.Get("main")
	if mainFunc == nil || goja.IsUndefined(mainFunc) {
		log.Printf("ℹ️  Script %s has no main() function, loaded as library", ctx.Name)
		return
	}

	// Vérifier si c'est une fonction
	if callable, ok := goja.AssertFunction(mainFunc); ok {
		log.Printf("▶️  Running script: %s", ctx.Name)

		// Exécuter main()
		result, err := callable(goja.Undefined())
		if err != nil {
			log.Printf("❌ Script %s error: %v", ctx.Name, err)
			return
		}

		// Logger le résultat si présent
		if result != nil && !goja.IsUndefined(result) {
			log.Printf("✅ Script %s completed with result: %v", ctx.Name, result.Export())
		} else {
			log.Printf("✅ Script %s completed", ctx.Name)
		}
	}
}

// Arrêter un script
func (sm *ScriptManager) StopScript(scriptID string) error {
	sm.RLock()
	ctx, exists := sm.Get(scriptID)
	sm.RUnlock()

	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	ctx.mu.Lock()
	if ctx.isRunning {
		close(ctx.stopChan)
		ctx.isRunning = false
	}
	ctx.mu.Unlock()

	log.Printf("⏹️  Stopped script: %s", ctx.Name)
	return nil
}

// Recharger un script
func (sm *ScriptManager) ReloadScript(scriptID string) error {
	sm.StopScript(scriptID)

	sm.Lock()
	ctx, _ := sm.Get(scriptID)
	filePath := ctx.FilePath
	delete(sm.contexts, scriptID)
	sm.Unlock()

	log.Printf("🔄 Reloading script: %s", scriptID)
	return sm.LoadScript(filePath)
}

// Watch les changements de fichiers
func (sm *ScriptManager) watchFiles() {
	for {
		select {
		case event, ok := <-sm.watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				if strings.HasSuffix(event.Name, ".js") || strings.HasSuffix(event.Name, ".ts") {
					// Module partagé modifié
					if strings.HasPrefix(event.Name, sm.modulesDir) {
						log.Printf("🔄 Module changed: %s", filepath.Base(event.Name))
						sm.loadSharedModules()
						// Recharger tous les scripts qui utilisent ce module
						sm.reloadAllScripts()
					} else {
						// Script hook modifié
						scriptName := strings.TrimSuffix(filepath.Base(event.Name), filepath.Ext(event.Name))
						log.Printf("🔄 Script changed: %s", scriptName)
						sm.ReloadScript(scriptName)
					}
				}
			} else if event.Op&fsnotify.Create == fsnotify.Create {
				if strings.HasSuffix(event.Name, ".js") || strings.HasSuffix(event.Name, ".ts") {
					log.Printf("➕ New file detected: %s", filepath.Base(event.Name))
					time.Sleep(100 * time.Millisecond) // Attendre que le fichier soit complètement écrit

					if strings.HasPrefix(event.Name, sm.modulesDir) {
						sm.loadSharedModules()
					} else {
						sm.LoadScript(event.Name)
					}
				}
			}

		case err, ok := <-sm.watcher.Errors:
			if !ok {
				return
			}
			log.Println("❌ Watcher error:", err)
		}
	}
}

// Recharger tous les scripts
func (sm *ScriptManager) reloadAllScripts() {
	sm.RLock()
	scriptIDs := make([]string, 0, len(sm.contexts))
	for id := range sm.contexts {
		scriptIDs = append(scriptIDs, id)
	}
	sm.RUnlock()

	for _, id := range scriptIDs {
		sm.ReloadScript(id)
	}
}

// ==================== API EXPOSURE ====================

func (sm *ScriptManager) exposeAPIs(ctx *ScriptContext) {
	vm := ctx.vm

	// ==================== CONSOLE & UTILITIES ====================
	vm.Set("log", func(args ...interface{}) {
		log.Printf("[%s] %v", ctx.Name, fmt.Sprint(args...))
	})

	vm.Set("sleep", func(ms int) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	})

	vm.Set("timestamp", func() int64 {
		return time.Now().Unix()
	})

	// ==================== DATABASE API ====================
	dbAPI := map[string]interface{}{
		"findById": func(collection, id string) map[string]interface{} {
			record, err := sm.app.FindRecordById(collection, id)
			if err != nil {
				return map[string]interface{}{"error": err.Error()}
			}
			return recordToMap(record)
		},

		"findOne": func(collection, filter string) map[string]interface{} {
			record, err := sm.app.FindFirstRecordByFilter(collection, filter)
			if err != nil {
				return map[string]interface{}{"error": err.Error()}
			}
			return recordToMap(record)
		},

		"findAll": func(collection string, args ...interface{}) []map[string]interface{} {
			filter := ""
			sort := "-created"
			limit := 100

			if len(args) > 0 {
				if f, ok := args[0].(string); ok {
					filter = f
				}
			}
			if len(args) > 1 {
				if s, ok := args[1].(string); ok {
					sort = s
				}
			}
			if len(args) > 2 {
				if l, ok := args[2].(int64); ok {
					limit = int(l)
				}
			}

			records, err := sm.app.FindRecordsByFilter(collection, filter, sort, limit, 0)
			if err != nil {
				return []map[string]interface{}{{"error": err.Error()}}
			}

			result := make([]map[string]interface{}, len(records))
			for i, r := range records {
				result[i] = recordToMap(r)
			}
			return result
		},

		"create": func(collection string, data map[string]interface{}) map[string]interface{} {
			coll, err := sm.app.FindCollectionByNameOrId(collection)
			if err != nil {
				return map[string]interface{}{"error": err.Error()}
			}

			record := core.NewRecord(coll)
			for key, value := range data {
				record.Set(key, value)
			}

			if err := sm.app.Save(record); err != nil {
				return map[string]interface{}{"error": err.Error()}
			}

			return recordToMap(record)
		},

		"update": func(collection, id string, data map[string]interface{}) map[string]interface{} {
			record, err := sm.app.FindRecordById(collection, id)
			if err != nil {
				return map[string]interface{}{"error": err.Error()}
			}

			for key, value := range data {
				record.Set(key, value)
			}

			if err := sm.app.Save(record); err != nil {
				return map[string]interface{}{"error": err.Error()}
			}

			return recordToMap(record)
		},

		"delete": func(collection, id string) map[string]interface{} {
			record, err := sm.app.FindRecordById(collection, id)
			if err != nil {
				return map[string]interface{}{"error": err.Error()}
			}

			if err := sm.app.Delete(record); err != nil {
				return map[string]interface{}{"error": err.Error()}
			}

			return map[string]interface{}{"success": true, "id": id}
		},

		"count": func(collection, filter string) int64 {
			records, err := sm.app.FindRecordsByFilter(collection, filter, "", 999999, 0)
			if err != nil {
				return 0
			}
			return int64(len(records))
		},
	}
	vm.Set("db", dbAPI)

	// ==================== WEBRTC API ====================
	webrtcAPI := map[string]interface{}{
		"getRoom": func(roomId string) map[string]interface{} {
			roomsMutex.RLock()
			room, exists := rooms[roomId]
			roomsMutex.RUnlock()

			if !exists {
				return map[string]interface{}{"error": "room not found"}
			}

			room.mu.RLock()
			defer room.mu.RUnlock()

			participants := []string{}
			for id := range room.Participants {
				participants = append(participants, id)
			}

			return map[string]interface{}{
				"id":           room.ID,
				"type":         room.Type,
				"participants": participants,
				"count":        len(participants),
			}
		},

		"listRooms": func() []map[string]interface{} {
			roomsMutex.RLock()
			defer roomsMutex.RUnlock()

			result := []map[string]interface{}{}
			for _, room := range rooms {
				room.mu.RLock()
				result = append(result, map[string]interface{}{
					"id":           room.ID,
					"type":         room.Type,
					"participants": len(room.Participants),
				})
				room.mu.RUnlock()
			}
			return result
		},

		"broadcast": func(roomId string, eventType string, data map[string]interface{}) map[string]interface{} {
			roomsMutex.RLock()
			room, exists := rooms[roomId]
			roomsMutex.RUnlock()

			if !exists {
				return map[string]interface{}{"error": "room not found"}
			}

			room.broadcastEvent(eventType, data)
			return map[string]interface{}{"success": true}
		},
	}
	vm.Set("webrtc", webrtcAPI)

	// ==================== PUBSUB API ====================
	pubsubAPI := map[string]interface{}{
		"publish": func(topic string, payload map[string]interface{}) {
			pubsub.Publish(topic, PubSubMessage{
				Topic:   topic,
				Payload: payload,
			})
		},

		"subscribe": func(topic string, callback goja.Callable) {
			ch := pubsub.Subscribe(topic)
			go func() {
				for {
					select {
					case msg := <-ch:
						ctx.mu.RLock()
						if !ctx.isRunning {
							ctx.mu.RUnlock()
							return
						}
						ctx.mu.RUnlock()

						data, _ := json.Marshal(msg.Payload)
						_, err := callback(goja.Undefined(), vm.ToValue(string(data)))
						if err != nil {
							log.Printf("[%s] Callback error: %v", ctx.Name, err)
						}

					case <-ctx.stopChan:
						return
					}
				}
			}()
		},
	}
	vm.Set("pubsub", pubsubAPI)

	// ==================== SOCIAL API ====================
	socialAPI := map[string]interface{}{
		"likePost": func(userId, postId, reaction string) map[string]interface{} {
			r, err := sm.app.FindCollectionByNameOrId("likes")
			if err != nil {
				return map[string]interface{}{"error": err.Error()}
			}
			like := core.NewRecord(r)
			like.Set("user", userId)
			like.Set("post", postId)
			like.Set("reaction", reaction)

			if err := sm.app.Save(like); err != nil {
				return map[string]interface{}{"error": err.Error()}
			}

			post, _ := sm.app.FindRecordById("posts", postId)
			if post != nil {
				count := post.GetInt("likesCount")
				post.Set("likesCount", count+1)
				sm.app.Save(post)
			}

			return map[string]interface{}{"success": true, "like_id": like.Id}
		},

		"getTrendingPosts": func(limit int) []map[string]interface{} {
			records, _ := sm.app.FindRecordsByFilter(
				"posts",
				"isPublic = true",
				"-likesCount,-commentsCount",
				limit,
				0,
			)

			result := make([]map[string]interface{}, len(records))
			for i, r := range records {
				result[i] = recordToMap(r)
			}
			return result
		},
	}
	vm.Set("social", socialAPI)

	// ==================== UTILS API ====================
	utilsAPI := map[string]interface{}{
		"jsonEncode": func(data interface{}) string {
			b, _ := json.Marshal(data)
			return string(b)
		},

		"jsonDecode": func(str string) map[string]interface{} {
			var result map[string]interface{}
			json.Unmarshal([]byte(str), &result)
			return result
		},

		"msgpackEncode": func(data interface{}) []byte {
			b, _ := msgpack.Marshal(data)
			return b
		},

		"msgpackDecode": func(data []byte) map[string]interface{} {
			var result map[string]interface{}
			msgpack.Unmarshal(data, &result)
			return result
		},

		"generateId": func() string {
			return generateID()
		},

		"uuid": func() string {
			return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
		},
	}
	vm.Set("utils", utilsAPI)

	// ==================== LOCATION API ====================
	locationAPI := map[string]interface{}{
		"updateLocation": func(userId string, lat, lng, accuracy float64, presence string) map[string]interface{} {
			location := Location{
				Point: Point{
					Lat: lat,
					Lng: lng,
				},
				Accuracy:  accuracy,
				Timestamp: time.Now(),
			}

			if presence == "" {
				presence = "online"
			}

			err := locationManager.UpdateLocation(userId, location, presence)
			if err != nil {
				return map[string]interface{}{"error": err.Error()}
			}

			return map[string]interface{}{"success": true, "user_id": userId}
		},

		"getLocation": func(userId string) map[string]interface{} {
			userLoc, exists := locationManager.GetLocation(userId)
			if !exists {
				return map[string]interface{}{"error": "location not found"}
			}

			return map[string]interface{}{
				"user_id":  userLoc.UserID,
				"lat":      userLoc.Location.Point.Lat,
				"lng":      userLoc.Location.Point.Lng,
				"presence": userLoc.Presence,
				"updated":  userLoc.UpdatedAt.Format(time.RFC3339),
			}
		},

		"findNearby": func(lat, lng, radius float64) []map[string]interface{} {
			point := Point{Lat: lat, Lng: lng}
			nearby := locationManager.FindNearby(point, radius, "")

			result := make([]map[string]interface{}, len(nearby))
			for i, user := range nearby {
				result[i] = map[string]interface{}{
					"user_id":  user.UserID,
					"lat":      user.Location.Point.Lat,
					"lng":      user.Location.Point.Lng,
					"presence": user.Presence,
					"distance": HaversineDistance(point, user.Location.Point),
				}
			}
			return result
		},

		"getUsersByPresence": func(presence string) []map[string]interface{} {
			users := locationManager.GetUsersByPresence(presence)

			result := make([]map[string]interface{}, len(users))
			for i, user := range users {
				result[i] = map[string]interface{}{
					"user_id":  user.UserID,
					"lat":      user.Location.Point.Lat,
					"lng":      user.Location.Point.Lng,
					"presence": user.Presence,
				}
			}
			return result
		},

		"distance": func(lat1, lng1, lat2, lng2 float64) float64 {
			p1 := Point{Lat: lat1, Lng: lng1}
			p2 := Point{Lat: lat2, Lng: lng2}
			return HaversineDistance(p1, p2)
		},
	}
	vm.Set("location", locationAPI)

	// ==================== CRON API ====================
	cronAPI := map[string]interface{}{
		"schedule": func(interval int, callback goja.Callable) {
			go func() {
				ticker := time.NewTicker(time.Duration(interval) * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						ctx.mu.RLock()
						if !ctx.isRunning {
							ctx.mu.RUnlock()
							return
						}
						ctx.mu.RUnlock()

						_, err := callback(goja.Undefined())
						if err != nil {
							log.Printf("[%s] Scheduled task error: %v", ctx.Name, err)
						}

					case <-ctx.stopChan:
						return
					}
				}
			}()
		},

		"setTimeout": func(delay int, callback goja.Callable) {
			go func() {
				select {
				case <-time.After(time.Duration(delay) * time.Millisecond):
					callback(goja.Undefined())
				case <-ctx.stopChan:
					return
				}
			}()
		},
	}
	vm.Set("cron", cronAPI)
}

// Helper
func recordToMap(record *core.Record) map[string]interface{} {
	result := make(map[string]interface{})
	result["id"] = record.Id
	result["created"] = record.GetDateTime("created")
	result["updated"] = record.GetDateTime("updated")

	for _, field := range record.Collection().Fields.AsMap() {
		result[field.GetName()] = record.Get(field.GetName())
	}

	return result
}

func SetupScriptsCollection(app core.App) error {
	return app.RunInTransaction(func(txApp core.App) error {
		// Check if collection already exists
		_, err := txApp.FindCollectionByNameOrId("scripts")
		if err == nil {
			return nil // Already exists
		}

		scripts := core.NewBaseCollection("scripts")
		scripts.Fields.Add(
			&core.TextField{
				Name:     "name",
				Required: true,
			},
			&core.TextField{
				Name:     "code",
				Required: true,
			},
			&core.TextField{
				Name: "description",
			},
			&core.BoolField{
				Name: "enabled",
			},
			&core.RelationField{
				Name:         "user",
				CollectionId: "_pb_users_auth_",
				MaxSelect:    1,
			},
		)
		return txApp.Save(scripts)
	})
}
