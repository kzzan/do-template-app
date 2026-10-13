-- user 服务初始化 schema（幂等，三库通用）
CREATE TABLE IF NOT EXISTS users (
    user_id    VARCHAR(36)  PRIMARY KEY,
    nickname   VARCHAR(64)  NOT NULL DEFAULT '',
    password   VARCHAR(128) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    created_at TIMESTAMP    NOT NULL,
    updated_at TIMESTAMP    NOT NULL,
    CONSTRAINT uq_users_email UNIQUE (email)
);
