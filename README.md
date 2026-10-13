# do-template-app

monorepo 多服务后端模板：**gin + sqlx + zerolog + samber/do/v2**，
目录结构对齐 [nunu-layout-monorepo](https://github.com/go-nunu/nunu-layout-monorepo)（去前端），
换栈处理方式与 [do-template-api](../do-template-api) 一致。

## 目录结构（对齐 nunu-layout-monorepo）

```
app/
  gateway/                网关（:8020）：etcd 发现后端 + 路径反代（/v1/site→home，其余→admin）
    cmd/server/  internal/server
  admin/                 管理端服务（:8010）：登录 + 管理员/角色/权限 RBAC；终端用户数据经 gRPC 取自 user
    api/v1/              DTO 与业务错误码
    cmd/server/          入口：do.New(packages...) 装配
    cmd/migration/       执行 storage/admin-{schema,seed}.sql（幂等）
    internal/            handler(admin+member) / service / repository / middleware(jwt,rbac) / server
  home/                  门户服务（:8002）：轻量壳（Health/Meta/Manifest/Stats）
    api/v1/  cmd/server/  internal/{handler,service,server}
  user/                  用户服务（唯一持有终端用户数据，DB: users）：
    gRPC :9500（GetProfile/ListUsers/CountUsers）
    HTTP :8003（注册/登录/资料/自删 CRUD）
    cmd/{server,migration}/  internal/{handler(grpc+http),service,repository,server}
api/
  proto/userv1/          跨服务 gRPC 契约源（make proto 重新生成到 api/pb/userv1）
config/
  admin/  home/  user/   按服务分目录（admin/user 带 DB，home 仅 http+grpc 客户端）
model/                   共享模型（AdminUser/Role/UserRole/Permission）
pkg/                     共享层（按职责切割）：
  bootstrap(装配基础) / config / log / jwt / database(基座+redis)
  middleware(通用) / grpcx(gRPC客户端) / discovery(etcd) / migrate(SQL迁移)
storage/
  admin/ user/           各服务 schema.sql + seed.sql
```

与 nunu 的对应关系：`wire → samber/do`、`gorm → sqlx`、`zap → zerolog`、
`casbin+gorm-adapter → 四表直查`（语义等价：g:user→role、p:role→resource+action）、
`pkg/app 生命周期 → do.ShutdownOnSignals`。砍掉：web 前端、job/task、swagger、sign 中间件、
home 的静态文件 Index。

## 网关与 etcd 服务发现

```
client → gateway :8020 ──etcd 发现──► admin :8010 / home :8002（HTTP 反代透传）
admin ──etcd gRPC resolver（watch 热更新）──► user gRPC :9500
```

- **注册**：各服务启动时向 etcd 写 `do-template/{service}/{addr}`（lease TTL 15s 保活），
  容器优雅关闭即注销；进程崩溃后键 15s 内自动过期（防脑裂）
- **发现**：gateway 按 `/v1/site/*`→home、其余 `/v1/*`→admin 反代（5s 缓存）；
  admin→user 走自定义 scheme `dtcd://`（watch 实时推送，实例换端口免重启）
- **本地 etcd**：`make up`（docker compose）；测试无 etcd 自动跳过
- ponytail：gateway 单实例取第一个地址，无负载均衡/重试熔断——上生产换专职网关（如 APISIX/Envoy）

## 服务间数据流

```
app/user (gRPC :9500, users 表) ◄──┬── admin :8010  GET /v1/admin/members（RBAC 保护）
                                   └── home  :8002  GET /v1/site/stats
proto 契约：api/proto/userv1 → api/pb/userv1；客户端连接：pkg/grpcx（grpc.client_addr）
```

## 快速开始

```bash
make migration       # admin 建表 + 种子（超管 admin/admin123、角色 editor）
make migration-user  # user 建表 + 种子（alice/bob，密码 123456）
make run-user        # gRPC :9500（先起）
make up              # 启动本地 etcd（docker）
make run             # 一键启动四服务（需先 make up）
make run-admin       # http://127.0.0.1:8010
make run-home        # http://127.0.0.1:8002

# admin：超管登录（RBAC 跳过）→ 列表 → 建 editor 角色账号
curl -X POST localhost:8010/v1/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
curl "localhost:8010/v1/admin/users?page=1&pageSize=10" -H "Authorization: Bearer <token>"
curl -X POST localhost:8010/v1/admin/users -H "Authorization: Bearer <token>" \
  -H 'Content-Type: application/json' \
  -d '{"username":"editor1","password":"123456","roles":["editor"]}'

# home：门户 API（auth 经 etcd 发现转发 user 服务；stats 经 gRPC 取 user 服务）
curl -X POST localhost:8002/v1/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","password":"123456","nickname":"A"}'
curl -X POST localhost:8002/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","password":"123456"}'

# user HTTP CRUD（直连 :8003，Bearer 自查/自改/自删）
curl localhost:8003/v1/user -H "Authorization: Bearer <token>"
curl -X DELETE localhost:8003/v1/user -H "Authorization: Bearer <token>"
curl localhost:8002/v1/site/health
curl localhost:8002/v1/site/stats

# admin：终端用户管理（经 gRPC 调 user 服务，RBAC 保护）
curl "localhost:8010/v1/admin/members?page=1&pageSize=10" -H "Authorization: Bearer <token>"

# user 服务可用 grpcurl 直连调试（已开 reflection）
grpcurl -plaintext -d '{"page":1,"pageSize":10}' 127.0.0.1:9500 userv1.UserService/ListUsers

# 统一入口（gateway，路由透传）
curl localhost:8020/v1/site/health           # → home
curl -X POST localhost:8020/v1/login ...     # → admin

# 观察服务注册
docker compose exec etcd etcdctl get do-template/ --prefix
```

## RBAC（casbin 最简模型的等价实现）

```
user_roles(user_id, role_sid)                    ≈ casbin g: user, role
role_permissions(role_sid, resource, action)     ≈ casbin p: role, resource, action
Enforce = SELECT EXISTS(user→role→perm join)     ≈ e.Enforce(user, resource, action)
```

- 权限点：`resource = "api:" + 路由路径`，`action = HTTP 方法`
- 超管（`user_id = "1"`，seed 固定）跳过检查（对齐 nunu 防呆设计）
- 中间件链：`StrictAuth(JWT) → RBAC`，未授权 403

## samber/do 用法（与 do-template-api 一致）

各层自带 `Package`（`do.Package`+`do.Lazy`），main.go 组合装配：

```go
container := do.New(
    basePackage(*confPath),   // config/log(/jwt)
    database.Package,         // 共享基座：DB/Redis/Repository/Bind(Transaction)（admin 用）
    repository.Package,       // app 实体 repo
    service.Package,
    handler.Package,
    server.Package,
)
```

`ShutdownOnSignalsWithContext`（10s 限时）收信号按依赖序优雅关闭；构造器统一
`NewX(injector do.Injector)`，`do:""` 标签注入，不存 injector 运行时调用（官方反模式）。

## 切换数据库

同 do-template-api：`config/admin/*.yml` 改 `data.db.driver/dsn`
（sqlite/mysql/postgres），schema/seed 用 `WHERE NOT EXISTS` 守卫三库幂等。

## 测试

```bash
make test
```

- `app/admin/internal/repository`：sqlite `:memory:` 真库——CRUD/分页/过滤、RBAC Enforce 判定链、角色连带清理
- `app/admin/internal/service`：假仓库经 do 容器装配——登录、重名、超管保护
- `pkg/jwt`、`pkg/database`：JWT 往返、redis 装配（无 redis 自动跳过）

## 统一模板家族

`do-template-api`（用户端单体，:8000）独立 module；本仓库是 **monorepo 双服务**
（admin :8001 + home :8002，共享 model/pkg）。新增服务：复制 `app/home` 骨架 +
`config/<name>/`，在 Makefile 加一行 run。
