package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"

	"do-template-app/model"
	"do-template-app/pkg/database"
	"do-template-app/pkg/log"
)

const userTestSchema = `
CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(36) PRIMARY KEY, nickname VARCHAR(64) NOT NULL DEFAULT '',
    password VARCHAR(128) NOT NULL, email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL
);
`

func newUserTestRepo(t *testing.T) (*userRepository, func()) {
	t.Helper()
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	if _, err := db.Exec(userTestSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	r := &userRepository{Repository: database.New(log.NewLogger(viper.New()), &database.Database{DB: db})}
	return r, func() { db.Close() }
}

func TestUserRepository(t *testing.T) {
	repo, cleanup := newUserTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	for _, u := range []model.User{
		{UserID: "u-1", Nickname: "alice", Email: "a@b.com", Password: "h"},
		{UserID: "u-2", Nickname: "bob", Email: "b@b.com", Password: "h"},
		{UserID: "u-3", Nickname: "carol", Email: "c@b.com", Password: "h"},
	} {
		if err := repo.Create(ctx, &u); err != nil {
			t.Fatalf("create %s: %v", u.UserID, err)
		}
	}

	if total, err := repo.Count(ctx); err != nil || total != 3 {
		t.Errorf("Count = %d (err %v), want 3", total, err)
	}

	users, total, err := repo.List(ctx, 1, 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 || len(users) != 2 {
		t.Errorf("List page1: total=%d len=%d, want 3/2", total, len(users))
	}

	got, err := repo.GetByID(ctx, "u-2")
	if err != nil || got.Email != "b@b.com" {
		t.Errorf("GetByID: %v %+v", err, got)
	}
}
