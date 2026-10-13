-- 初始种子数据（幂等：WHERE NOT EXISTS 守卫，三库通用）
-- 超管：admin / admin123（bcrypt），user_id 固定为 '1'，RBAC 中间件对其跳过权限检查
INSERT INTO admin_users (user_id, username, nickname, password, email, phone, created_at, updated_at)
SELECT '1', 'admin', 'Super Admin',
       '$2a$10$IaJ949l7Xbd05zhJPX0qcOFUGvMCbVNvs39cuCaCUoPPkmq451W2i',
       'admin@do-template.dev', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM admin_users WHERE user_id = '1');

-- 示例角色：editor（可读管理员列表）
INSERT INTO roles (sid, name, created_at)
SELECT 'editor', 'Editor', CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE sid = 'editor');

INSERT INTO role_permissions (role_sid, resource, action)
SELECT 'editor', 'api:/v1/admin/users', 'GET'
WHERE NOT EXISTS (SELECT 1 FROM role_permissions WHERE role_sid = 'editor' AND resource = 'api:/v1/admin/users' AND action = 'GET');
