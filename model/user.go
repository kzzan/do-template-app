package model

import "time"

// User 终端用户（user 服务持有；db 标签对应 users 表列名）。
type User struct {
	UserID    string    `db:"user_id"`
	Nickname  string    `db:"nickname"`
	Password  string    `db:"password"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
