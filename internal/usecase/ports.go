package usecase

import (
	"context"
	"time"

	"secunda/internal/domain"
)

// UserRepository — порт хранения пользователей. Реализация живёт в adapter/mysql.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// TeamRepository — порт хранения команд и участников.
type TeamRepository interface {
	CreateWithOwner(ctx context.Context, team *domain.Team) error
	ListByUser(ctx context.Context, userID int64) ([]domain.TeamWithRole, error)
	Membership(ctx context.Context, teamID, userID int64) (domain.Role, error)
	AddMember(ctx context.Context, teamID, userID int64, role domain.Role) error
}

// TokenIssuer — выпуск access-токенов.
type TokenIssuer interface {
	Issue(userID int64) (token string, expiresAt time.Time, err error)
}

// PasswordHasher — хеширование и проверка паролей.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}
