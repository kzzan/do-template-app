-- user 服务种子（幂等）：两个示例用户，密码均为 123456
INSERT INTO users (user_id, nickname, password, email, created_at, updated_at)
SELECT 'u-alice', 'Alice', '$2a$10$RStp0ejiZdT3kzc1qfMq9OzfK/OeOCpJbtU.CW9ojWIchXzMvYLrS',
       'alice@do-template.dev', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM users WHERE user_id = 'u-alice');

INSERT INTO users (user_id, nickname, password, email, created_at, updated_at)
SELECT 'u-bob', 'Bob', '$2a$10$GTL5YNWJkf9BLzn0E/D8c.rZuowwpa3uwoCBoLxDfT0XT0Z9SPgQ.',
       'bob@do-template.dev', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM users WHERE user_id = 'u-bob');
