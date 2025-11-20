-- Test data for integration tests

-- Insert test users
INSERT INTO users (id, email, username, password) VALUES
('test_user_1', 'user1@test.com', 'user1', 'hashed_password'),
('test_user_2', 'user2@test.com', 'user2', 'hashed_password'),
('test_user_3', 'user3@test.com', 'user3', 'hashed_password');

-- Insert test posts
INSERT INTO posts (id, user, type, content, isPublic, likesCount, commentsCount) VALUES
('test_post_1', 'test_user_1', 'html', 'Test post 1', true, 0, 0),
('test_post_2', 'test_user_2', 'html', 'Test post 2', true, 0, 0);

-- Insert test articles
INSERT INTO articles (id, title, prix, quantite, user) VALUES
('test_article_1', 'Test Product', 99.99, 10, 'test_user_1');

-- Insert test rooms
INSERT INTO rooms (id, roomType, name, owner, joinType, maxParticipants, isActive) VALUES
('test_room_1', 'audio', 'Test Room', 'test_user_1', 'free', 50, true);
