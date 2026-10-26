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

var _ usecase.TaskRepository = (*TaskRepository)(nil)

const taskColumns = `id, team_id, title, description, status, created_by, assignee_id,
	created_at, updated_at, closed_at, version`

type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	res, err := ext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO tasks
			(team_id, title, description, status, created_by, assignee_id,
			 created_at, updated_at, closed_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.TeamID, task.Title, task.Description, string(task.Status), task.CreatedBy,
		task.AssigneeID, task.CreatedAt, task.UpdatedAt, task.ClosedAt, task.Version,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("task last insert id: %w", err)
	}
	task.ID = id
	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	var task domain.Task
	err := sqlx.GetContext(ctx, ext(ctx, r.db), &task,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTaskNotFound.WithErr(err)
		}
		return nil, fmt.Errorf("select task: %w", err)
	}
	return &task, nil
}

// GetForUpdate блокирует строку до конца транзакции.
func (r *TaskRepository) GetForUpdate(ctx context.Context, id int64) (*domain.Task, error) {
	var task domain.Task
	err := sqlx.GetContext(ctx, ext(ctx, r.db), &task,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ? FOR UPDATE`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTaskNotFound.WithErr(err)
		}
		return nil, fmt.Errorf("select task for update: %w", err)
	}
	return &task, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *domain.Task, expectedVersion int64) error {
	res, err := ext(ctx, r.db).ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, assignee_id = ?,
		    updated_at = ?, closed_at = ?, version = ?
		WHERE id = ? AND version = ?`,
		task.Title, task.Description, string(task.Status), task.AssigneeID,
		task.UpdatedAt, task.ClosedAt, task.Version, task.ID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("task rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

func (r *TaskRepository) List(ctx context.Context, filter usecase.TaskFilter) ([]domain.Task, int, error) {
	where := `WHERE team_id = ?`
	args := []any{filter.TeamID}

	if filter.Status != nil {
		where += ` AND status = ?`
		args = append(args, string(*filter.Status))
	}
	if filter.AssigneeID != nil {
		where += ` AND assignee_id = ?`
		args = append(args, *filter.AssigneeID)
	}

	db := ext(ctx, r.db)

	var total int
	if err := sqlx.GetContext(ctx, db, &total, `SELECT COUNT(*) FROM tasks `+where, args...); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	listArgs := append(append([]any{}, args...), filter.Limit, filter.Offset)
	tasks := make([]domain.Task, 0, filter.Limit)
	query := `SELECT ` + taskColumns + ` FROM tasks ` + where +
		` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	if err := sqlx.SelectContext(ctx, db, &tasks, query, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("select tasks: %w", err)
	}

	return tasks, total, nil
}
