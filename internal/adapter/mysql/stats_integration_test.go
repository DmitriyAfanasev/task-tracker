package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"secunda/migrations"
)

// TestTeamStatsRepositoryIntegration запускается против настоящего MySQL 8.
// Передайте отдельную тестовую БД в TEST_MYSQL_DSN; без неё обычный go test
// остаётся полностью самодостаточным.
func TestTeamStatsRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not set")
	}

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Fatalf("connect MySQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.Up(db.DB, slog.New(slog.NewTextHandler(os.Stderr, nil))); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	ctx := context.Background()
	stamp := time.Now().UTC().UnixNano()
	insertUser := func(name string) int64 {
		t.Helper()
		result, err := db.ExecContext(ctx,
			`INSERT INTO users (email, password_hash, name, created_at) VALUES (?, 'hash', ?, UTC_TIMESTAMP(6))`,
			fmt.Sprintf("stats-%d-%s@example.test", stamp, name), name,
		)
		if err != nil {
			t.Fatalf("insert user %s: %v", name, err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("user id: %v", err)
		}
		return id
	}
	insertTeam := func(ownerID int64, name string) int64 {
		t.Helper()
		result, err := db.ExecContext(ctx,
			`INSERT INTO teams (name, created_by, created_at) VALUES (?, ?, UTC_TIMESTAMP(6))`, name, ownerID)
		if err != nil {
			t.Fatalf("insert team: %v", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("team id: %v", err)
		}
		return id
	}

	ownerID := insertUser("Owner")
	aliceID := insertUser("Alice")
	bobID := insertUser("Bob")
	otherOwnerID := insertUser("Other")
	teamID := insertTeam(ownerID, "stats target")
	otherTeamID := insertTeam(otherOwnerID, "stats other")
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM teams WHERE id IN (?, ?)`, teamID, otherTeamID)
		_, _ = db.ExecContext(ctx, `DELETE FROM users WHERE id IN (?, ?, ?, ?)`, ownerID, aliceID, bobID, otherOwnerID)
	})

	for _, membership := range []struct {
		teamID, userID int64
		role           string
	}{
		{teamID, ownerID, "owner"}, {teamID, aliceID, "member"}, {teamID, bobID, "member"},
		{otherTeamID, otherOwnerID, "owner"},
	} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO team_members (team_id, user_id, role, created_at) VALUES (?, ?, ?, UTC_TIMESTAMP(6))`,
			membership.teamID, membership.userID, membership.role,
		); err != nil {
			t.Fatalf("insert membership: %v", err)
		}
	}

	base := time.Now().UTC().Truncate(time.Microsecond)
	insertTask := func(team, assignee int64, status string, createdAt time.Time, closedAt *time.Time) int64 {
		t.Helper()
		result, err := db.ExecContext(ctx, `
			INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
			VALUES (?, 'task', '', ?, ?, ?, ?, ?, ?, 1)`,
			team, status, ownerID, assignee, createdAt, createdAt, closedAt)
		if err != nil {
			t.Fatalf("insert task: %v", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("task id: %v", err)
		}
		return id
	}
	doneAt := base
	doneOne := insertTask(teamID, aliceID, "done", base.Add(-24*time.Hour), &doneAt)
	_ = insertTask(teamID, aliceID, "done", base.Add(-48*time.Hour), &doneAt)
	oldClosed := base.Add(-40 * 24 * time.Hour)
	_ = insertTask(teamID, bobID, "done", base.Add(-50*24*time.Hour), &oldClosed)
	_ = insertTask(teamID, aliceID, "todo", base, nil)
	_ = insertTask(teamID, bobID, "in_progress", base, nil)
	_ = insertTask(teamID, bobID, "cancelled", base, nil)
	otherDone := insertTask(otherTeamID, otherOwnerID, "done", base.Add(-24*time.Hour), &doneAt)

	for _, taskID := range []int64{doneOne, doneOne, doneOne, otherDone} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO task_comments (task_id, user_id, content, created_at) VALUES (?, ?, 'comment', UTC_TIMESTAMP(6))`, taskID, ownerID); err != nil {
			t.Fatalf("insert comment: %v", err)
		}
	}

	stats, err := NewTeamStatsRepository(db).Stats(ctx, teamID)
	if err != nil {
		t.Fatalf("Stats() error: %v", err)
	}
	if stats.TeamID != teamID || stats.TasksByStatus.Todo != 1 || stats.TasksByStatus.InProgress != 1 || stats.TasksByStatus.Done != 3 || stats.TasksByStatus.Cancelled != 1 {
		t.Fatalf("unexpected status counts: %+v", stats)
	}
	if stats.CommentsCount != 3 {
		t.Fatalf("comments_count = %d, want 3", stats.CommentsCount)
	}
	if stats.AverageCloseSeconds == nil || math.Abs(*stats.AverageCloseSeconds-374400) > 0.001 {
		t.Fatalf("average_close_seconds = %v, want 374400", stats.AverageCloseSeconds)
	}
	if len(stats.TopAssignees) != 1 || stats.TopAssignees[0].UserID != aliceID || stats.TopAssignees[0].ClosedTasks != 2 {
		t.Fatalf("top_assignees = %+v, want Alice with 2 tasks", stats.TopAssignees)
	}
}
