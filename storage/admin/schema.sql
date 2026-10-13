-- do-template-app 初始化 schema（幂等，可重复执行）
-- user_id 由应用侧 uuid 生成；超管由 seed 固定为 '1'，主键不依赖自增（三库通用）
-- RBAC：user_roles + role_permissions 两表表达 user→role→(resource, action)，
-- 语义等价于 nunu 的 casbin 最简模型（p: role,resource,action + g: user,role）
CREATE TABLE IF NOT EXISTS admin_users (
    user_id    VARCHAR(36)  PRIMARY KEY,
    username   VARCHAR(50)  NOT NULL,
    nickname   VARCHAR(50)  NOT NULL DEFAULT '',
    password   VARCHAR(128) NOT NULL,
    email      VARCHAR(100) NOT NULL DEFAULT '',
    phone      VARCHAR(20)  NOT NULL DEFAULT '',
    created_at TIMESTAMP    NOT NULL,
    updated_at TIMESTAMP    NOT NULL,
    CONSTRAINT uq_admin_users_username UNIQUE (username)
);

CREATE TABLE IF NOT EXISTS roles (
    sid        VARCHAR(50) PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMP   NOT NULL
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id  VARCHAR(36) NOT NULL,
    role_sid VARCHAR(50) NOT NULL,
    CONSTRAINT pk_user_roles PRIMARY KEY (user_id, role_sid)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_sid VARCHAR(50)  NOT NULL,
    resource VARCHAR(255) NOT NULL,
    action   VARCHAR(20)  NOT NULL,
    CONSTRAINT pk_role_permissions PRIMARY KEY (role_sid, resource, action)
);
