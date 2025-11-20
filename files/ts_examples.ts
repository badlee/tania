// ==================== EXEMPLES TYPESCRIPT/JAVASCRIPT ====================
// Tous ces scripts peuvent être exécutés via l'interpréteur Goja

// ==================== EXEMPLE 1: MODÉRATION AUTOMATIQUE ====================
function moderateNewPost(postId: string) {
  // Récupérer le post
  const post = db.findById("posts", postId);
  
  if (post.error) {
    log("Post not found:", postId);
    return { error: post.error };
  }
  
  // Vérifier le contenu
  const content = post.content || "";
  const badWords = ["spam", "scam", "abuse"];
  
  let isSuspicious = false;
  for (const word of badWords) {
    if (content.toLowerCase().includes(word)) {
      isSuspicious = true;
      break;
    }
  }
  
  if (isSuspicious) {
    // Marquer le post comme non public
    db.update("posts", postId, {
      isPublic: false,
      dataAction: JSON.stringify({
        moderated: true,
        reason: "suspicious_content",
        timestamp: timestamp()
      })
    });
    
    log("Post moderated:", postId);
    
    // Notifier l'admin via pubsub
    pubsub.publish("admin_notifications", {
      type: "content_moderation",
      post_id: postId,
      reason: "suspicious_content"
    });
    
    return { moderated: true, postId };
  }
  
  return { moderated: false, postId };
}

// ==================== EXEMPLE 2: SYSTÈME DE RÉCOMPENSES ====================
function rewardActiveUsers() {
  log("Running reward system...");
  
  // Trouver les utilisateurs avec plus de 10 posts ce mois
  const posts = db.findAll(
    "posts",
    `created >= "${getMonthStart()}"`,
    "-created",
    1000
  );
  
  // Compter les posts par utilisateur
  const userPostCount = {};
  for (const post of posts) {
    const userId = post.user;
    userPostCount[userId] = (userPostCount[userId] || 0) + 1;
  }
  
  // Récompenser les utilisateurs actifs
  const rewards = [];
  for (const userId in userPostCount) {
    const count = userPostCount[userId];
    
    if (count >= 10) {
      const rewardAmount = count * 10; // 10 coins par post
      
      // Créer une opération de cashin
      const operation = db.create("operations", {
        user: userId,
        montant: rewardAmount,
        operation: "cashin",
        desc: `Récompense pour ${count} posts ce mois`,
        status: "paye"
      });
      
      rewards.push({
        userId,
        posts: count,
        reward: rewardAmount
      });
      
      log(`Rewarded user ${userId}: ${rewardAmount} coins`);
    }
  }
  
  // Publier les résultats
  pubsub.publish("rewards", {
    type: "monthly_rewards",
    rewards: rewards,
    total: rewards.length
  });
  
  return {
    success: true,
    rewarded: rewards.length,
    details: rewards
  };
}

function getMonthStart() {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`;
}

// ==================== EXEMPLE 3: ANALYTICS EN TEMPS RÉEL ====================
function calculateEngagementScore(postId: string) {
  const post = db.findById("posts", postId);
  
  if (post.error) {
    return { error: post.error };
  }
  
  const likesCount = post.likesCount || 0;
  const commentsCount = post.commentsCount || 0;
  
  // Calcul du score d'engagement
  const likeWeight = 1;
  const commentWeight = 3;
  const engagementScore = (likesCount * likeWeight) + (commentsCount * commentWeight);
  
  // Vérifier l'âge du post
  const postAge = timestamp() - new Date(post.created).getTime() / 1000;
  const ageHours = postAge / 3600;
  
  // Score normalisé par heure
  const scorePerHour = ageHours > 0 ? engagementScore / ageHours : engagementScore;
  
  // Déterminer si c'est "trending"
  const isTrending = scorePerHour > 10 && ageHours < 24;
  
  // Mettre à jour les dataAction du post
  db.update("posts", postId, {
    dataAction: JSON.stringify({
      engagementScore: Math.round(engagementScore),
      scorePerHour: Math.round(scorePerHour * 100) / 100,
      isTrending: isTrending,
      lastCalculated: timestamp()
    })
  });
  
  if (isTrending) {
    // Notifier via pubsub
    pubsub.publish("post_events", {
      type: "trending_post",
      post_id: postId,
      score: engagementScore
    });
  }
  
  return {
    postId,
    engagementScore,
    scorePerHour,
    isTrending
  };
}

// ==================== EXEMPLE 4: GESTION AUTOMATIQUE DU STOCK ====================
function checkLowStock() {
  log("Checking low stock items...");
  
  // Trouver les articles avec stock < 5
  const lowStockArticles = db.findAll(
    "articles",
    "quantite < 5 && quantite > 0",
    "-quantite",
    100
  );
  
  const notifications = [];
  
  for (const article of lowStockArticles) {
    const userId = article.user;
    
    // Créer une notification pour le vendeur
    log(`Low stock alert for article ${article.id}: ${article.quantite} remaining`);
    
    notifications.push({
      userId,
      articleId: article.id,
      articleTitle: article.title,
      remaining: article.quantite
    });
    
    // Publier via pubsub
    pubsub.publish("notifications", {
      type: "low_stock",
      user_id: userId,
      article: {
        id: article.id,
        title: article.title,
        quantity: article.quantite
      }
    });
  }
  
  return {
    success: true,
    lowStockCount: lowStockArticles.length,
    notifications
  };
}

// ==================== EXEMPLE 5: SYSTÈME DE RECOMMANDATIONS ====================
function getRecommendedPosts(userId: string, limit: number = 10) {
  // Récupérer les posts que l'utilisateur a likés
  const userLikes = db.findAll(
    "likes",
    `user = "${userId}"`,
    "-created",
    100
  );
  
  // Extraire les catégories préférées
  const categoryPreferences = {};
  
  for (const like of userLikes) {
    const post = db.findById("posts", like.post);
    if (!post.error && post.categories) {
      for (const category of post.categories) {
        categoryPreferences[category] = (categoryPreferences[category] || 0) + 1;
      }
    }
  }
  
  // Trouver la catégorie la plus aimée
  let topCategory = null;
  let maxCount = 0;
  
  for (const category in categoryPreferences) {
    if (categoryPreferences[category] > maxCount) {
      topCategory = category;
      maxCount = categoryPreferences[category];
    }
  }
  
  if (!topCategory) {
    // Si pas de préférences, retourner les posts tendances
    return social.getTrendingPosts(limit);
  }
  
  // Récupérer les posts de la catégorie préférée
  const recommendedPosts = db.findAll(
    "posts",
    `isPublic = true && categories ~ "${topCategory}"`,
    "-likesCount",
    limit
  );
  
  return {
    userId,
    topCategory,
    categoryScore: maxCount,
    recommendations: recommendedPosts
  };
}

// ==================== EXEMPLE 6: STATISTIQUES VENDEUR ====================
function getSellerDashboard(userId: string) {
  // Stats des articles
  const articles = db.findAll(
    "articles",
    `user = "${userId}"`,
    "-created",
    1000
  );
  
  let totalStock = 0;
  let totalValue = 0;
  
  for (const article of articles) {
    totalStock += article.quantite || 0;
    totalValue += (article.quantite || 0) * (article.prix || 0);
  }
  
  // Stats des ventes
  const salesStats = marketplace.getSalesStats(userId);
  
  // Balance wallet
  const balance = marketplace.getWalletBalance(userId);
  
  // Posts récents
  const recentPosts = db.findAll(
    "posts",
    `user = "${userId}"`,
    "-created",
    10
  );
  
  let totalEngagement = 0;
  for (const post of recentPosts) {
    totalEngagement += (post.likesCount || 0) + (post.commentsCount || 0);
  }
  
  return {
    seller: {
      userId,
      articlesCount: articles.length,
      totalStock,
      totalValue,
      balance
    },
    sales: salesStats,
    engagement: {
      recentPosts: recentPosts.length,
      totalEngagement,
      avgPerPost: recentPosts.length > 0 ? totalEngagement / recentPosts.length : 0
    }
  };
}

// ==================== EXEMPLE 7: BOT DE ROOM AUDIO ====================
function createMusicBot(roomId: string) {
  const room = webrtc.getRoom(roomId);
  
  if (room.error) {
    return { error: "Room not found" };
  }
  
  log(`Music bot joining room: ${roomId}`);
  
  // Broadcast welcome message
  webrtc.broadcast(roomId, "chat", {
    from: "MusicBot",
    message: "🎵 Music Bot has joined! Type /play <song> to request music"
  });
  
  // Store bot state
  storage.set(`bot_${roomId}`, {
    active: true,
    roomId: roomId,
    playlist: [],
    currentSong: null
  });
  
  return {
    success: true,
    roomId,
    botActive: true
  };
}

// ==================== EXEMPLE 8: SCHEDULED TASKS ====================
function setupScheduledTasks() {
  // Tâche toutes les heures: calculer engagement
  cron.schedule(3600, function() {
    log("Running hourly engagement calculation...");
    
    const recentPosts = db.findAll(
      "posts",
      `created >= "${getHourAgo()}"`,
      "-created",
      100
    );
    
    for (const post of recentPosts) {
      calculateEngagementScore(post.id);
    }
    
    log("Engagement calculation complete");
  });
  
  // Tâche toutes les 10 minutes: vérifier le stock
  cron.schedule(600, function() {
    checkLowStock();
  });
  
  // Tâche quotidienne: récompenses
  cron.schedule(86400, function() {
    log("Running daily rewards...");
    rewardActiveUsers();
  });
  
  log("Scheduled tasks setup complete");
  
  return { success: true };
}

function getHourAgo() {
  const now = new Date();
  now.setHours(now.getHours() - 1);
  return now.toISOString();
}

// ==================== EXEMPLE 9: WEBHOOK HANDLER ====================
function handleWebhook(event: string, data: any) {
  log("Webhook received:", event);
  
  switch (event) {
    case "payment_success":
      handlePaymentSuccess(data);
      break;
    
    case "payment_failed":
      handlePaymentFailed(data);
      break;
    
    case "user_signup":
      handleUserSignup(data);
      break;
    
    default:
      log("Unknown webhook event:", event);
  }
  
  return { processed: true, event };
}

function handlePaymentSuccess(data: any) {
  const venteId = data.vente_id;
  
  // Mettre à jour le statut de la vente
  db.update("ventesArticle", venteId, {
    status: "paye",
    paiementDate: new Date().toISOString()
  });
  
  // Mettre à jour l'opération associée
  const vente = db.findById("ventesArticle", venteId);
  if (vente.operation) {
    db.update("operations", vente.operation, {
      status: "paye"
    });
  }
  
  // Notifier le vendeur
  pubsub.publish("notifications", {
    type: "payment_received",
    vente_id: venteId,
    amount: data.amount
  });
  
  log("Payment processed:", venteId);
}

function handlePaymentFailed(data: any) {
  const venteId = data.vente_id;
  
  db.update("ventesArticle", venteId, {
    status: "echec",
    failDate: new Date().toISOString()
  });
  
  log("Payment failed:", venteId);
}

function handleUserSignup(data: any) {
  const userId = data.user_id;
  
  // Créer un bonus de bienvenue
  db.create("operations", {
    user: userId,
    montant: 100,
    operation: "cashin",
    desc: "Bonus de bienvenue",
    status: "paye"
  });
  
  log("Welcome bonus granted to:", userId);
}

// ==================== EXEMPLE 10: CUSTOM API LOGIC ====================
function customBusinessLogic(params: any) {
  log("Custom business logic triggered with params:", params);
  
  // Exemple: Créer un post et un article simultanément
  if (params.createProductPost) {
    // 1. Créer l'article
    const article = db.create("articles", {
      title: params.title,
      desc: params.description,
      prix: params.price,
      prixOriginal: params.originalPrice,
      quantite: params.quantity,
      user: params.userId
    });
    
    if (article.error) {
      return { error: article.error };
    }
    
    // 2. Créer le post
    const post = db.create("posts", {
      user: params.userId,
      type: "images",
      content: params.content,
      article: article.id,
      action: "buy",
      actionText: "Acheter maintenant",
      isPublic: true,
      categories: params.categories || ["other"]
    });
    
    if (post.error) {
      return { error: post.error };
    }
    
    // 3. Publier l'événement
    pubsub.publish("post_events", {
      type: "new_product",
      post_id: post.id,
      article_id: article.id,
      user_id: params.userId
    });
    
    return {
      success: true,
      article: article,
      post: post
    };
  }
  
  return { error: "Unknown action" };
}

// ==================== MAIN ENTRY POINT ====================
function main() {
  log("TypeScript interpreter initialized!");
  log("Available APIs: db, webrtc, pubsub, social, marketplace, utils, storage, cron, auth");
  
  // Setup scheduled tasks
  // setupScheduledTasks();
  
  return {
    status: "ready",
    timestamp: timestamp(),
    apis: ["db", "webrtc", "pubsub", "social", "marketplace", "utils", "storage", "cron", "auth", "http"]
  };
}

// Auto-run main on load
// main();
