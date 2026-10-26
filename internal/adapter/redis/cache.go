package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"secunda/internal/domain"
	"secunda/internal/usecase"
)

const defaultCacheTTL = 5 * time.Minute

var _ usecase.TaskListCache = (*TaskListCache)(nil)

// TaskListCache кеширует список задач с версионированием ключей.
// Инвалидация — INCR версии команды, без SCAN и удаления пачками.
type TaskListCache struct {
	client *goredis.Client
	ttl    time.Duration
	log    *slog.Logger
}

func NewTaskListCache(client *goredis.Client, ttl time.Duration, log *slog.Logger) *TaskListCache {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &TaskListCache{client: client, ttl: ttl, log: log}
}

func (c *TaskListCache) Get(ctx context.Context, teamID int64, filterKey string) (usecase.TaskListResult, int64, bool) {
	version, ok := c.version(ctx, teamID)
	if !ok {
		return usecase.TaskListResult{}, 0, false
	}

	data, err := c.client.Get(ctx, listKey(teamID, version, filterKey)).Bytes()
	if err != nil {
		if !errors.Is(err, goredis.Nil) {
			c.log.Warn("redis get task list", "team_id", teamID, "error", err)
		}
		return usecase.TaskListResult{}, version, false
	}

	var result usecase.TaskListResult
	if err := json.Unmarshal(data, &result); err != nil {
		c.log.Warn("redis decode task list", "team_id", teamID, "error", err)
		return usecase.TaskListResult{}, version, false
	}
	if result.Tasks == nil {
		result.Tasks = []domain.Task{}
	}
	return result, version, true
}

func (c *TaskListCache) Set(ctx context.Context, teamID int64, filterKey string, version int64, result usecase.TaskListResult) {
	data, err := json.Marshal(result)
	if err != nil {
		c.log.Warn("redis encode task list", "team_id", teamID, "error", err)
		return
	}
	if err := c.client.Set(ctx, listKey(teamID, version, filterKey), data, c.ttl).Err(); err != nil {
		c.log.Warn("redis set task list", "team_id", teamID, "error", err)
	}
}

func (c *TaskListCache) InvalidateTeam(ctx context.Context, teamID int64) {
	if err := c.client.Incr(ctx, versionKey(teamID)).Err(); err != nil {
		c.log.Warn("redis invalidate task list", "team_id", teamID, "error", err)
	}
}

func (c *TaskListCache) version(ctx context.Context, teamID int64) (int64, bool) {
	version, err := c.client.Get(ctx, versionKey(teamID)).Int64()
	if errors.Is(err, goredis.Nil) {
		return 0, true
	}
	if err != nil {
		c.log.Warn("redis get task list version", "team_id", teamID, "error", err)
		return 0, false
	}
	return version, true
}

func versionKey(teamID int64) string {
	return fmt.Sprintf("secunda:tasks:ver:%d", teamID)
}

func listKey(teamID, version int64, filterKey string) string {
	sum := sha256.Sum256([]byte(filterKey))
	hash := hex.EncodeToString(sum[:12])
	return fmt.Sprintf("secunda:tasks:v%d:team:%d:%s", version, teamID, hash)
}
