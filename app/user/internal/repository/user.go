package repository

import (
	"context"
	"errors"
	"time"

	"github.com/samber/do/v2"

	"do-template-app/model"
	"do-template-app/pkg/database"
)

const userColumns = "user_id, nickname, password, email, created_at, updated_at"

// UserRepository 终端用户数据访问接口。
type UserRepository interface {
	GetByID(ctx context.Context, userID string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, page, pageSize int32) ([]model.User, int64, error)
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, userID string) error
}

func NewUserRepository(injector do.Injector) (UserRepository, error) {
	return &userRepository{
		Repository: do.MustInvoke[*database.Repository](injector),
	}, nil
}

type userRepository struct {
	*database.Repository
}

func (r *userRepository) GetByID(ctx context.Context, userID string) (*model.User, error) {
	return r.getOneBy(ctx, "user_id = ?", userID)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.getOneBy(ctx, "email = ?", email)
}

// ErrNotFound 语义：查无记录返回 (nil, nil)，与 admin 仓库区分，由 service 判定。
func (r *userRepository) getOneBy(ctx context.Context, where string, arg any) (*model.User, error) {
	var user model.User
	query := r.DB(ctx).Rebind("SELECT " + userColumns + " FROM users WHERE " + where)
	err := r.DB(ctx).GetContext(ctx, &user, query, arg)
	if errors.Is(err, database.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) List(ctx context.Context, page, pageSize int32) ([]model.User, int64, error) {
	var total int64
	if err := r.DB(ctx).GetContext(ctx, &total, r.DB(ctx).Rebind("SELECT COUNT(*) FROM users")); err != nil {
		return nil, 0, err
	}

	query := r.DB(ctx).Rebind("SELECT " + userColumns + " FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?")
	users := []model.User{}
	if err := r.DB(ctx).SelectContext(ctx, &users, query, pageSize, (page-1)*pageSize); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.DB(ctx).GetContext(ctx, &total, r.DB(ctx).Rebind("SELECT COUNT(*) FROM users"))
	return total, err
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	now := time.Now().UTC()
	user.CreatedAt, user.UpdatedAt = now, now
	query := r.DB(ctx).Rebind("INSERT INTO users (" + userColumns + ") VALUES (?, ?, ?, ?, ?, ?)")
	_, err := r.DB(ctx).ExecContext(ctx, query,
		user.UserID, user.Nickname, user.Password, user.Email, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now().UTC()
	query := r.DB(ctx).Rebind("UPDATE users SET nickname = ?, password = ?, email = ?, updated_at = ? WHERE user_id = ?")
	_, err := r.DB(ctx).ExecContext(ctx, query,
		user.Nickname, user.Password, user.Email, user.UpdatedAt, user.UserID,
	)
	return err
}

func (r *userRepository) Delete(ctx context.Context, userID string) error {
	query := r.DB(ctx).Rebind("DELETE FROM users WHERE user_id = ?")
	_, err := r.DB(ctx).ExecContext(ctx, query, userID)
	return err
}
