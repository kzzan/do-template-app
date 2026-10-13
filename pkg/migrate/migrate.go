// Package migrate 简易 SQL 迁移：按服务名执行 storage/{service}/ 下的 schema.sql 与 seed.sql。
package migrate

import (
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"github.com/jmoiron/sqlx"

	"do-template-app/pkg/config"
)

// Run 执行指定服务的迁移（幂等可重复）。ponytail: 无版本管理；
// 正式项目需要演进式迁移时换 golang-migrate / goose
func Run(confPath, service string) {
	conf := config.NewConfig(confPath)
	driver, dsn := conf.GetString("data.db.driver"), conf.GetString("data.db.dsn")

	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		log.Fatalf("connect %s: %v", driver, err)
	}
	defer db.Close()

	for _, name := range []string{"schema", "seed"} {
		path := fmt.Sprintf("storage/%s/%s.sql", service, name)
		if err := execFile(db, path); err != nil {
			log.Fatalf("%s: %v", path, err)
		}
	}
	fmt.Printf("migration done: %s\n", service)
}

func execFile(db *sqlx.DB, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, stmt := range strings.Split(strings.TrimSpace(string(b)), ";") {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", strings.TrimSpace(stmt), err)
		}
	}
	return nil
}
