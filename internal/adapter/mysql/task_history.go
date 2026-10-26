package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"secunda/internal/domain"
	"secunda/internal/usecase"
)

var _ usecase.TaskHistoryRepository = (*TaskHistoryRepository)(nil)

type TaskHistoryRepository struct {
	db *sqlx.DB
}

func NewTaskHistoryRepository(db *sqlx.DB) *TaskHistoryRepository {
	return &TaskHistoryRepository{db: db}
}

func (r *TaskHistoryRepository) Add(ctx context.Context, entry *domain.TaskHistoryEntry) error {
	changes, err := json.Marshal(entry.Changes)
	if err != nil {
		return fmt.Errorf("marshal task changes: %w", err)
	}

	res, err := ext(ctx, r.db).ExecContext(ctx, `
		INSERT INTO task_history (task_id, changed_by, changes, created_at)
		VALUES (?, ?, ?, ?)`,
		entry.TaskID, entry.ChangedBy, changes, entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task history: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("task history last insert id: %w", err)
	}
	entry.ID = id
	return nil
}

type historyRow struct {
	ID        int64     `db:"id"`
	TaskID    int64     `db:"task_id"`
	ChangedBy int64     `db:"changed_by"`
	Changes   []byte    `db:"changes"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *TaskHistoryRepository) List(ctx context.Context, taskID int64) ([]domain.TaskHistoryEntry, error) {
	rows := make([]historyRow, 0)
	err := sqlx.SelectContext(ctx, ext(ctx, r.db), &rows, `
		SELECT id, task_id, changed_by, changes, created_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY created_at ASC, id ASC`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("select task history: %w", err)
	}

	entries := make([]domain.TaskHistoryEntry, 0, len(rows))
	for _, row := range rows {
		changes := make([]domain.TaskChange, 0)
		if err := json.Unmarshal(row.Changes, &changes); err != nil {
			return nil, fmt.Errorf("unmarshal task changes: %w", err)
		}
		entries = append(entries, domain.TaskHistoryEntry{
			ID:        row.ID,
			TaskID:    row.TaskID,
			ChangedBy: row.ChangedBy,
			Changes:   changes,
			CreatedAt: row.CreatedAt,
		})
	}
	return entries, nil
}
