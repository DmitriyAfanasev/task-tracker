package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"secunda/internal/domain"
	"secunda/internal/usecase"
)

var _ usecase.TeamStatsRepository = (*TeamStatsRepository)(nil)

// TeamStatsRepository хранит SQL-реализацию отчёта, чтобы она не расползалась
// по HTTP-слою и use case.
type TeamStatsRepository struct {
	db *sqlx.DB
}

func NewTeamStatsRepository(db *sqlx.DB) *TeamStatsRepository {
	return &TeamStatsRepository{db: db}
}

type teamStatsRow struct {
	Todo       int64           `db:"todo"`
	InProgress int64           `db:"in_progress"`
	Done       int64           `db:"done"`
	Cancelled  int64           `db:"cancelled"`
	Average    sql.NullFloat64 `db:"average_close_seconds"`
	Comments   int64           `db:"comments_count"`
	Top        []byte          `db:"top_assignees"`
}

func (r *TeamStatsRepository) Stats(ctx context.Context, teamID int64) (*domain.TeamStats, error) {
	const query = `
		WITH task_metrics AS (
			SELECT
				COALESCE(SUM(status = 'todo'), 0) AS todo,
				COALESCE(SUM(status = 'in_progress'), 0) AS in_progress,
				COALESCE(SUM(status = 'done'), 0) AS done,
				COALESCE(SUM(status = 'cancelled'), 0) AS cancelled,
				AVG(CASE WHEN status = 'done' AND closed_at IS NOT NULL
					THEN TIMESTAMPDIFF(MICROSECOND, created_at, closed_at) / 1000000.0 END) AS average_close_seconds
			FROM tasks
			WHERE team_id = ?
		),
		comment_metrics AS (
			SELECT COUNT(*) AS comments_count
			FROM task_comments tc
			JOIN tasks t ON t.id = tc.task_id
			WHERE t.team_id = ?
		),
		top_assignee_rows AS (
			SELECT t.assignee_id AS user_id, u.name, COUNT(*) AS closed_tasks
			FROM tasks t
			JOIN users u ON u.id = t.assignee_id
			WHERE t.team_id = ?
			  AND t.status = 'done'
			  AND t.closed_at >= UTC_TIMESTAMP(6) - INTERVAL 30 DAY
			GROUP BY t.assignee_id, u.name
			ORDER BY closed_tasks DESC, t.assignee_id ASC
			LIMIT 3
		),
		top_assignee_json AS (
			SELECT COALESCE(
				JSON_ARRAYAGG(JSON_OBJECT('user_id', user_id, 'name', name, 'closed_tasks', closed_tasks)),
				JSON_ARRAY()
			) AS top_assignees
			FROM top_assignee_rows
		)
		SELECT tm.todo, tm.in_progress, tm.done, tm.cancelled,
		       tm.average_close_seconds, cm.comments_count, taj.top_assignees
		FROM task_metrics tm
		CROSS JOIN comment_metrics cm
		CROSS JOIN top_assignee_json taj`

	var row teamStatsRow
	if err := sqlx.GetContext(ctx, r.db, &row, query, teamID, teamID, teamID); err != nil {
		return nil, fmt.Errorf("select team stats: %w", err)
	}

	stats := domain.TeamStats{
		TeamID: teamID,
		TasksByStatus: domain.TasksByStatus{
			Todo: row.Todo, InProgress: row.InProgress, Done: row.Done, Cancelled: row.Cancelled,
		},
		CommentsCount: row.Comments,
	}
	if row.Average.Valid {
		stats.AverageCloseSeconds = &row.Average.Float64
	}
	if err := json.Unmarshal(row.Top, &stats.TopAssignees); err != nil {
		return nil, fmt.Errorf("decode top assignees: %w", err)
	}
	if stats.TopAssignees == nil {
		stats.TopAssignees = make([]domain.TopAssignee, 0)
	}
	return &stats, nil
}
