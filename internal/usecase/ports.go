package usecase

//go:generate go run github.com/golang/mock/mockgen -source=ports.go -package usecase -destination=mocks_ports.go

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
	Create(ctx context.Context, team *domain.Team) error
	ListByUser(ctx context.Context, userID int64) ([]domain.TeamWithRole, error)
	ListMembers(ctx context.Context, teamID int64) ([]domain.TeamMemberInfo, error)
	Membership(ctx context.Context, teamID, userID int64) (domain.Role, error)
	AddMember(ctx context.Context, teamID, userID int64, role domain.Role) error
}

// TeamStatsRepository выполняет агрегированный SQL-отчёт по команде.
// Авторизация остаётся в use case, чтобы SQL-адаптер не знал о правилах доступа.
type TeamStatsRepository interface {
	Stats(ctx context.Context, teamID int64) (*domain.TeamStats, error)
}

// TaskRepository — порт хранения задач.
type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
	GetForUpdate(ctx context.Context, id int64) (*domain.Task, error)
	Update(ctx context.Context, task *domain.Task, expectedVersion int64) error
	List(ctx context.Context, filter TaskFilter) ([]domain.Task, int, error)
}

// TaskHistoryRepository — порт истории изменений задач.
type TaskHistoryRepository interface {
	Add(ctx context.Context, entry *domain.TaskHistoryEntry) error
	List(ctx context.Context, taskID int64) ([]domain.TaskHistoryEntry, error)
}

// TaskCommentRepository — порт комментариев задач.
type TaskCommentRepository interface {
	Add(ctx context.Context, comment *domain.TaskComment) error
	List(ctx context.Context, taskID int64, limit, offset int) ([]domain.TaskComment, int, error)
}

// TaskListCache — кеш списка задач. Реализация не должна ронять запрос:
// при недоступном Redis это просто промах кеша.
type TaskListCache interface {
	// Get возвращает результат, версию команды и признак попадания.
	// Версия нужна, чтобы записать результат под тем же снимком,
	// даже если параллельно произошла инвалидация.
	Get(ctx context.Context, teamID int64, filterKey string) (TaskListResult, int64, bool)
	Set(ctx context.Context, teamID int64, filterKey string, version int64, result TaskListResult)
	InvalidateTeam(ctx context.Context, teamID int64)
}

// TransactionManager — граница транзакции на уровне сценария.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
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
