// Package database 数据访问基座：连接池、事务传播与 redis（多 app 共享）。
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"do-template-app/pkg/log"
)

// Package 注册本层全部服务（官方 do-template-app 的模块化注册方式）。
// Redis 为懒加载，业务侧需要时 do.MustInvoke[*Redis] 即可，未用不连接。
var Package = do.Package(
	do.Lazy(NewDB),
	do.Lazy(NewRedis),
	do.Lazy(NewRepository),
	do.Bind[*Repository, Transaction](),
)

// Database 包装 *sqlx.DB；实现 ShutdownerWithError 与 HealthcheckerWithContext，
// 由 do 容器统一调度关闭与健康检查。
type Database struct {
	*sqlx.DB
}

// Shutdown 实现 do.ShutdownerWithError。
func (d *Database) Shutdown() error {
	return d.DB.Close()
}

// HealthCheck 实现 do.HealthcheckerWithContext。
func (d *Database) HealthCheck(ctx context.Context) error {
	return d.PingContext(ctx)
}

// NewDB 按配置驱动建立连接池：driver 支持 sqlite / mysql / postgres。
func NewDB(injector do.Injector) (*Database, error) {
	conf := do.MustInvoke[*viper.Viper](injector)
	logger := do.MustInvoke[*log.Logger](injector)

	driver := conf.GetString("data.db.driver")
	dsn := conf.GetString("data.db.dsn")

	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", driver, err)
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)

	// sqlite 单写者：连接数限制为 1，避免 SQLITE_BUSY
	if driver == "sqlite" {
		db.SetMaxOpenConns(1)
	}

	logger.Info().Str("driver", driver).Msg("database connected")
	return &Database{DB: db}, nil
}

// Queryer 是 *sqlx.DB 与 *sqlx.Tx 的公共查询接口，
// repo 方法通过它自动兼容「事务内 / 连接池」两种执行路径。
type Queryer interface {
	sqlx.ExtContext
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	Rebind(query string) string
}

type ctxTxKey struct{}

// Transaction 事务接口：service 层依赖此接口而非具体 Repository。
type Transaction interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// Repository 是各实体 repo 的公共基座，承载事务传播。
type Repository struct {
	db     *Database
	logger *log.Logger
}

var _ Transaction = (*Repository)(nil)

func NewRepository(injector do.Injector) (*Repository, error) {
	return New(do.MustInvoke[*log.Logger](injector), do.MustInvoke[*Database](injector)), nil
}

// New 纯构造：不经容器，测试与手动组装用。
func New(logger *log.Logger, db *Database) *Repository {
	return &Repository{db: db, logger: logger}
}

// Transaction 在事务中执行 fn；fn 内所有经过 DB(ctx) 的查询走同一事务。
// 已处于事务中时直接执行 fn（扁平加入外层，由外层统一提交/回滚），
// 避免嵌套调用产生独立事务破坏原子性。
// fn 返回 error 时回滚，否则提交；回滚错误与业务错误用 errors.Join 合并。
func (r *Repository) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(ctxTxKey{}).(*sqlx.Tx); ok {
		return fn(ctx)
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	ctx = context.WithValue(ctx, ctxTxKey{}, tx)
	if err := fn(ctx); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

// DB 返回 ctx 当前绑定的查询器：事务内返回 *sqlx.Tx，否则返回连接池。
func (r *Repository) DB(ctx context.Context) Queryer {
	if tx, ok := ctx.Value(ctxTxKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return r.db.DB
}

// ErrRecordNotFound 统一的查无记录错误。
var ErrRecordNotFound = sql.ErrNoRows
