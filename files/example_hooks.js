// ==================== STRUCTURE DES DOSSIERS ====================
/*
project/
├── pb_modules/          <- Modules partagés (espace commun)
│   ├── counter.js       <- Module counter partagé entre tous les scripts
│   ├── logger.js        <- Module logger personnalisé
│   └── cache.js         <- Module cache en mémoire
│
├── pb_hooks/            <- Scripts hooks (contextes indépendants)
│   ├── on-post-create.js    <- Hook création de post
│   ├── moderation.js        <- Modération automatique
│   ├── rewards.js           <- Système de récompenses
│   └── analytics.js         <- Analytics en temps réel
│
└── main.go
*/

// ==================== MODULE: pb_modules/counter.js ====================
// Ce module est PARTAGÉ entre tous les scripts qui l'importent
// La variable 'i' est COMMUNE à tous les scripts

let i = 0;
const history = [];

function inc() {
  i++;
  history.push({ value: i, timestamp: Date.now() });
  log("Counter incremented to:", i);
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

// Export dans l'espace partagé
exports.inc = inc;
exports.dec = dec;
exports.get = get;
exports.reset = reset;
exports.getHistory = getHistory;

// ==================== MODULE: pb_modules/logger.js ====================
// Module de logging personnalisé avec niveaux

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

// ==================== MODULE: pb_modules/cache.js ====================
// Cache en mémoire partagé entre tous les scripts

const cache = {};
const expirations = {};

function set(key, value, ttl) {
  cache[key] = value;
  
  if (ttl) {
    expirations[key] = Date.now() + (ttl * 1000);
    
    // Auto-cleanup
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
  // Vérifier expiration
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

// ==================== HOOK: pb_hooks/on-post-create.js ====================
// S'exécute automatiquement en arrière-plan dès le chargement
// Utilise les modules partagés

const counter = require("counter");
const logger = require("logger");
const cache = require("cache");

function main() {
  logger.info("Post creation hook initialized");
  
  // S'abonner aux nouveaux posts
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
  
  // Incrémenter le compteur partagé
  const count = counter.inc();
  logger.info(`Total posts created: ${count}`);
  
  // Récupérer le post
  const post = db.findById("posts", postId);
  
  if (post.error) {
    logger.error("Post not found:", postId);
    return;
  }
  
  // Vérifier si c'est le 10ème, 100ème, etc post
  if (count % 10 === 0) {
    logger.info(`🎉 Milestone reached: ${count} posts!`);
    
    // Récompenser l'utilisateur
    db.create("operations", {
      user: userId,
      montant: 50,
      operation: "cashin",
      desc: `Bonus: ${count}ème post de la communauté!`,
      status: "paye"
    });
    
    // Notifier
    pubsub.publish("notifications", {
      type: "milestone",
      user_id: userId,
      count: count,
      reward: 50
    });
  }
  
  // Mettre en cache pour analytics
  cache.set(`post:${postId}`, {
    id: postId,
    user: userId,
    created: Date.now()
  }, 3600); // TTL 1 heure
  
  logger.info("Post processed successfully");
}

// Auto-démarrage
main();

// ==================== HOOK: pb_hooks/moderation.js ====================
// Modération automatique avec utilisation des modules

const counter = require("counter");
const logger = require("logger");
const cache = require("cache");

// Liste noire partagée (persiste car dans le module)
const blacklist = ["spam", "scam", "abuse", "hate"];
const moderationStats = {
  total: 0,
  flagged: 0,
  removed: 0
};

function main() {
  logger.info("Moderation system started");
  
  // Écouter tous les nouveaux posts
  pubsub.subscribe("post_events", function(eventData) {
    const event = utils.jsonDecode(eventData);
    
    if (event.type === "new_post") {
      moderatePost(event.post_id);
    }
  });
  
  // Rapport de modération toutes les heures
  cron.schedule(3600, function() {
    generateModerationReport();
  });
}

function moderatePost(postId) {
  moderationStats.total++;
  
  const post = db.findById("posts", postId);
  
  if (post.error) {
    logger.error("Post not found for moderation:", postId);
    return;
  }
  
  const content = (post.content || "").toLowerCase();
  let flagged = false;
  let flagReasons = [];
  
  // Vérifier les mots interdits
  for (const word of blacklist) {
    if (content.includes(word)) {
      flagged = true;
      flagReasons.push(word);
    }
  }
  
  // Vérifier spam (trop de posts récents)
  const recentPosts = db.count("posts", `user = '${post.user}' && created >= '${getHourAgo()}'`);
  if (recentPosts > 10) {
    flagged = true;
    flagReasons.push("spam_rate_limit");
  }
  
  if (flagged) {
    moderationStats.flagged++;
    logger.warn(`Post ${postId} flagged:`, flagReasons);
    
    // Marquer comme non public
    db.update("posts", postId, {
      isPublic: false,
      dataAction: utils.jsonEncode({
        moderated: true,
        reasons: flagReasons,
        timestamp: timestamp()
      })
    });
    
    moderationStats.removed++;
    
    // Notifier l'admin
    pubsub.publish("admin_notifications", {
      type: "content_moderated",
      post_id: postId,
      user_id: post.user,
      reasons: flagReasons
    });
    
    // Incrémenter compteur de modération
    counter.inc();
    
    logger.info(`Post ${postId} removed from public view`);
  }
}

function generateModerationReport() {
  const report = {
    period: "last_hour",
    stats: moderationStats,
    total_moderated: counter.get(),
    timestamp: new Date().toISOString()
  };
  
  logger.info("Moderation report:", report);
  
  // Sauvegarder en cache
  cache.set("moderation_report", report, 3600);
  
  // Publier le rapport
  pubsub.publish("admin_notifications", {
    type: "moderation_report",
    report: report
  });
}

function getHourAgo() {
  const date = new Date();
  date.setHours(date.getHours() - 1);
  return date.toISOString();
}

main();

// ==================== HOOK: pb_hooks/rewards.js ====================
// Système de récompenses avec modules partagés

const counter = require("counter");
const logger = require("logger");
const cache = require("cache");

function main() {
  logger.info("Rewards system initialized");
  
  // Tâche quotidienne
  cron.schedule(86400, function() {
    distributeDaily Rewards();
  });
  
  // Récompenses pour engagement
  pubsub.subscribe("post_events", function(eventData) {
    const event = utils.jsonDecode(eventData);
    
    if (event.type === "like") {
      handleLikeReward(event.user_id, event.post_id);
    } else if (event.type === "comment") {
      handleCommentReward(event.user_id, event.post_id);
    }
  });
}

function distributeDailyRewards() {
  logger.info("🎁 Distributing daily rewards...");
  
  // Trouver les utilisateurs actifs aujourd'hui
  const posts = db.findAll("posts", `created >= '${getTodayStart()}'`, "-created", 1000);
  
  const userActivity = {};
  
  for (const post of posts) {
    const userId = post.user;
    if (!userActivity[userId]) {
      userActivity[userId] = {
        posts: 0,
        likes: 0,
        comments: 0
      };
    }
    userActivity[userId].posts++;
    userActivity[userId].likes += post.likesCount || 0;
    userActivity[userId].comments += post.commentsCount || 0;
  }
  
  // Calculer et distribuer les récompenses
  let totalRewarded = 0;
  
  for (const userId in userActivity) {
    const activity = userActivity[userId];
    const score = (activity.posts * 10) + (activity.likes * 1) + (activity.comments * 3);
    
    if (score >= 50) {
      const reward = Math.min(score, 500); // Max 500 coins/jour
      
      db.create("operations", {
        user: userId,
        montant: reward,
        operation: "cashin",
        desc: "Récompense quotidienne d'engagement",
        status: "paye"
      });
      
      totalRewarded += reward;
      logger.info(`Rewarded user ${userId}: ${reward} coins (score: ${score})`);
    }
  }
  
  logger.info(`Daily rewards distributed: ${totalRewarded} coins to ${Object.keys(userActivity).length} users`);
  
  // Publier les stats
  pubsub.publish("rewards", {
    type: "daily_distribution",
    total: totalRewarded,
    users: Object.keys(userActivity).length
  });
}

function handleLikeReward(userId, postId) {
  // Vérifier si déjà récompensé (via cache)
  const cacheKey = `like_reward:${userId}:${postId}`;
  if (cache.has(cacheKey)) {
    return; // Déjà récompensé
  }
  
  // Petit bonus pour l'engagement
  db.create("operations", {
    user: userId,
    montant: 1,
    operation: "cashin",
    desc: "Bonus engagement (like)",
    status: "paye"
  });
  
  // Marquer comme récompensé (24h)
  cache.set(cacheKey, true, 86400);
}

function handleCommentReward(userId, postId) {
  const cacheKey = `comment_reward:${userId}:${postId}`;
  if (cache.has(cacheKey)) {
    return;
  }
  
  db.create("operations", {
    user: userId,
    montant: 3,
    operation: "cashin",
    desc: "Bonus engagement (comment)",
    status: "paye"
  });
  
  cache.set(cacheKey, true, 86400);
}

function getTodayStart() {
  const date = new Date();
  date.setHours(0, 0, 0, 0);
  return date.toISOString();
}

main();

// ==================== HOOK: pb_hooks/analytics.js ====================
// Analytics en temps réel avec cache partagé

const counter = require("counter");
const logger = require("logger");
const cache = require("cache");

const analytics = {
  requests: 0,
  posts: 0,
  likes: 0,
  comments: 0,
  sales: 0
};

function main() {
  logger.info("Analytics system started");
  
  // Écouter tous les événements
  pubsub.subscribe("post_events", trackPostEvent);
  pubsub.subscribe("sales", trackSaleEvent);
  
  // Snapshot toutes les 5 minutes
  cron.schedule(300, function() {
    saveSnapshot();
  });
  
  // Rapport toutes les heures
  cron.schedule(3600, function() {
    generateHourlyReport();
  });
}

function trackPostEvent(eventData) {
  const event = utils.jsonDecode(eventData);
  
  analytics.requests++;
  
  if (event.type === "new_post") {
    analytics.posts++;
    counter.inc(); // Utilise le compteur global
  } else if (event.type === "like") {
    analytics.likes++;
  } else if (event.type === "comment") {
    analytics.comments++;
  }
  
  // Mettre en cache
  cache.set("analytics_current", analytics, 300);
}

function trackSaleEvent(eventData) {
  const event = utils.jsonDecode(eventData);
  
  if (event.type === "purchase") {
    analytics.sales++;
    cache.set("analytics_current", analytics, 300);
  }
}

function saveSnapshot() {
  const snapshot = {
    ...analytics,
    timestamp: new Date().toISOString(),
    total_posts: counter.get()
  };
  
  // Sauvegarder dans le cache avec un ID unique
  const snapshotId = `snapshot_${Date.now()}`;
  cache.set(snapshotId, snapshot, 3600);
  
  logger.info("Analytics snapshot saved:", snapshot);
}

function generateHourlyReport() {
  const report = {
    period: "last_hour",
    metrics: { ...analytics },
    global: {
      total_posts: counter.get(),
      cache_size: cache.size()
    },
    timestamp: new Date().toISOString()
  };
  
  logger.info("📊 Hourly analytics report:", report);
  
  // Publier
  pubsub.publish("admin_notifications", {
    type: "analytics_report",
    report: report
  });
  
  // Reset les compteurs horaires
  analytics.requests = 0;
  analytics.posts = 0;
  analytics.likes = 0;
  analytics.comments = 0;
  analytics.sales = 0;
}

main();

// ==================== UTILISATION DANS DIFFÉRENTS SCRIPTS ====================
/*
Script A (pb_hooks/script-a.js):
  const counter = require("counter");
  counter.inc(); // i devient 1 dans le module counter

Script B (pb_hooks/script-b.js):
  const counter = require("counter");
  counter.inc(); // i devient 2 (partagé!)
  counter.inc(); // i devient 3
  log(counter.get()); // Affiche: 3

Script C (pb_hooks/script-c.js):
  const counter = require("counter");
  log(counter.get()); // Affiche: 3 (même valeur!)
  counter.reset(); // Reset à 0

Tous les scripts partagent la MÊME instance du module counter!
*/
