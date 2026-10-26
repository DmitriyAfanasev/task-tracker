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

var _ usecase.TeamRepository = (*TeamRepository)(nil)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// CreateWithOwner атомарно создаёт команду и добавляет создателя как owner.
func (r *TeamRepository) CreateWithOwner(ctx context.Context, team *domain.Team) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO teams (name, created_by, created_at) VALUES (?, ?, ?)`,
		team.Name, team.CreatedBy, team.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert team: %w", err)
	}

	teamID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("team last insert id: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role, created_at) VALUES (?, ?, ?, ?)`,
		teamID, team.CreatedBy, string(domain.RoleOwner), team.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert team owner: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit team tx: %w", err)
	}

	team.ID = teamID
	return nil
}

func (r *TeamRepository) ListByUser(ctx context.Context, userID int64) ([]domain.TeamWithRole, error) {
	teams := make([]domain.TeamWithRole, 0)
	err := r.db.SelectContext(ctx, &teams, `
		SELECT t.id, t.name, t.created_by, t.created_at, tm.role
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC, t.id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("select teams by user: %w", err)
	}
	return teams, nil
}

func (r *TeamRepository) Membership(ctx context.Context, teamID, userID int64) (domain.Role, error) {
	var role domain.Role
	err := r.db.GetContext(ctx, &role,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`, teamID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Не раскрываем существование чужой команды.
			return "", domain.ErrTeamNotFound.WithErr(err)
		}
		return "", fmt.Errorf("select membership: %w", err)
	}
	return role, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, teamID, userID int64, role domain.Role) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role, created_at) VALUES (?, ?, ?, UTC_TIMESTAMP(6))`,
		teamID, userID, string(role),
	)
	if err != nil {
		switch {
		case isDuplicateKey(err):
			return domain.ErrAlreadyMember.WithErr(err)
		case isForeignKeyViolation(err):
			return domain.ErrNotFound.WithErr(err)
		default:
			return fmt.Errorf("insert team member: %w", err)
		}
	}
	return nil
}
