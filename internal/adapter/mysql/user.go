package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"secunda/internal/domain"
	"secunda/internal/usecase"
)

var _ usecase.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	res, err := r.db.ExecContext(
		ctx,
		`INSERT INTO users (email, password_hash, name, created_at) VALUES (?, ?, ?, ?)`,
		user.Email, user.PasswordHash, user.Name, user.CreatedAt,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return domain.ErrEmailTaken.WithErr(err)
		}
		return fmt.Errorf("insert user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("user last insert id: %w", err)
	}
	user.ID = id
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var user domain.User
	err := r.db.GetContext(ctx, &user,
		`SELECT id, email, password_hash, name, created_at FROM users WHERE id = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound.WithErr(err)
		}
		return nil, fmt.Errorf("select user by id: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.GetContext(ctx, &user,
		`SELECT id, email, password_hash, name, created_at FROM users WHERE email = ?`, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound.WithErr(err)
		}
		return nil, fmt.Errorf("select user by email: %w", err)
	}
	return &user, nil
}
