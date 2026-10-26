package mysql

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"secunda/internal/domain"
	"secunda/internal/usecase"
)

var _ usecase.TaskCommentRepository = (*TaskCommentRepository)(nil)

type TaskCommentRepository struct {
	db *sqlx.DB
}

func NewTaskCommentRepository(db *sqlx.DB) *TaskCommentRepository {
	return &TaskCommentRepository{db: db}
}

func (r *TaskCommentRepository) Add(ctx context.Context, comment *domain.TaskComment) error {
	res, err := ext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO task_comments (task_id, user_id, content, created_at)
		VALUES (?, ?, ?, ?)`,
		comment.TaskID, comment.UserID, comment.Content, comment.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task comment: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("task comment last insert id: %w", err)
	}
	comment.ID = id
	return nil
}

func (r *TaskCommentRepository) List(ctx context.Context, taskID int64, limit, offset int) ([]domain.TaskComment, int, error) {
	db := ext(ctx, r.db)

	var total int
	if err := sqlx.GetContext(ctx, db, &total,
		`SELECT COUNT(*) FROM task_comments WHERE task_id = ?`, taskID); err != nil {
		return nil, 0, fmt.Errorf("count task comments: %w", err)
	}

	comments := make([]domain.TaskComment, 0, limit)
	err := sqlx.SelectContext(ctx, db, &comments, `
		SELECT id, task_id, user_id, content, created_at
		FROM task_comments
		WHERE task_id = ?
		ORDER BY created_at ASC, id ASC
		LIMIT ? OFFSET ?`,
		taskID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("select task comments: %w", err)
	}
	return comments, total, nil
}
