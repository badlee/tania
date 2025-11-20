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
