package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"secunda/internal/domain"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
	maxTitleLength   = 255
	maxTextLength    = 5000
)

type TaskFilter struct {
	TeamID     int64
	Status     *domain.TaskStatus
	AssigneeID *int64
	Limit      int
	Offset     int
}

type TaskListResult struct {
	Tasks  []domain.Task `json:"tasks"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type CommentListResult struct {
	Comments []domain.TaskComment `json:"comments"`
	Total    int                  `json:"total"`
	Limit    int                  `json:"limit"`
	Offset   int                  `json:"offset"`
}

type CreateTaskRequest struct {
	UserID      int64
	TeamID      int64
	Title       string
	Description *string
	Status      *domain.TaskStatus
}

type ListTasksQuery struct {
	UserID     int64
	TeamID     int64
	Status     *domain.TaskStatus
	AssigneeID *int64
	Limit      int
	Offset     int
}

type UpdateTaskRequest struct {
	UserID          int64
	TaskID          int64
	Title           *string
	Description     *string
	Status          *domain.TaskStatus
	AssigneeID      *int64 // 0 — снять исполнителя, nil — не менять
	ExpectedVersion *int64
}

type AddCommentRequest struct {
	UserID  int64
	TaskID  int64
	Content string
}

type ListCommentsQuery struct {
	UserID int64
	TaskID int64
	Limit  int
	Offset int
}

type TaskUseCase struct {
	tasks    TaskRepository
	history  TaskHistoryRepository
	comments TaskCommentRepository
	teams    TeamRepository
	cache    TaskListCache
	tx       TransactionManager
}

func NewTaskUseCase(
	tasks TaskRepository,
	history TaskHistoryRepository,
	comments TaskCommentRepository,
	teams TeamRepository,
	cache TaskListCache,
	tx TransactionManager,
) *TaskUseCase {
	return &TaskUseCase{
		tasks:    tasks,
		history:  history,
		comments: comments,
		teams:    teams,
		cache:    cache,
		tx:       tx,
	}
}

func (uc *TaskUseCase) Create(ctx context.Context, cmd CreateTaskRequest) (*domain.Task, error) {
	details := make(map[string]string)

	title := strings.TrimSpace(cmd.Title)
	switch {
	case cmd.TeamID <= 0:
		details["team_id"] = "должен быть больше 0"
	case title == "":
		details["title"] = "поле обязательно"
	case utf8.RuneCountInString(title) > maxTitleLength:
		details["title"] = fmt.Sprintf("не более %d символов", maxTitleLength)
	}

	description := ""
	if cmd.Description != nil {
		description = *cmd.Description
		if utf8.RuneCountInString(description) > maxTextLength {
			details["description"] = fmt.Sprintf("не более %d символов", maxTextLength)
		}
	}

	status := domain.TaskStatusTodo
	if cmd.Status != nil {
		status = *cmd.Status
		if !status.Valid() {
			details["status"] = "недопустимый статус"
		}
	}

	if len(details) > 0 {
		return nil, domain.ErrValidation.WithDetails(details)
	}

	if _, err := uc.teams.Membership(ctx, cmd.TeamID, cmd.UserID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	assigneeID := cmd.UserID
	task := &domain.Task{
		TeamID:      cmd.TeamID,
		Title:       title,
		Description: description,
		Status:      status,
		CreatedBy:   cmd.UserID,
		AssigneeID:  &assigneeID,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}
	if status == domain.TaskStatusDone {
		task.ClosedAt = &now
	}

	changes := creationChanges(task)
	err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.tasks.Create(txCtx, task); err != nil {
			return err
		}
		return uc.history.Add(txCtx, &domain.TaskHistoryEntry{
			TaskID:    task.ID,
			ChangedBy: cmd.UserID,
			Changes:   changes,
			CreatedAt: now,
		})
	})
	if err != nil {
		return nil, err
	}

	uc.cache.InvalidateTeam(ctx, cmd.TeamID)
	return task, nil
}

func (uc *TaskUseCase) List(ctx context.Context, query ListTasksQuery) (*TaskListResult, error) {
	limit, offset, err := normalizePagination(query.Limit, query.Offset)
	if err != nil {
		return nil, err
	}
	if query.TeamID <= 0 {
		return nil, domain.ErrValidation.WithDetails(map[string]string{"team_id": "должен быть больше 0"})
	}
	if query.Status != nil && !query.Status.Valid() {
		return nil, domain.ErrValidation.WithDetails(map[string]string{"status": "недопустимый статус"})
	}
	if query.AssigneeID != nil && *query.AssigneeID <= 0 {
		return nil, domain.ErrValidation.WithDetails(map[string]string{"assignee_id": "должен быть больше 0"})
	}

	if _, err := uc.teams.Membership(ctx, query.TeamID, query.UserID); err != nil {
		return nil, err
	}

	filterKey := taskFilterKey(query.Status, query.AssigneeID, limit, offset)
	cached, version, hit := uc.cache.Get(ctx, query.TeamID, filterKey)
	if hit {
		return &cached, nil
	}

	tasks, total, err := uc.tasks.List(ctx, TaskFilter{
		TeamID:     query.TeamID,
		Status:     query.Status,
		AssigneeID: query.AssigneeID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, err
	}

	result := TaskListResult{Tasks: tasks, Total: total, Limit: limit, Offset: offset}
	uc.cache.Set(ctx, query.TeamID, filterKey, version, result)
	return &result, nil
}

func (uc *TaskUseCase) Update(ctx context.Context, cmd UpdateTaskRequest) (*domain.Task, error) {
	if err := validateTaskUpdate(cmd); err != nil {
		return nil, err
	}

	var (
		result  *domain.Task
		changed bool
	)

	err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, err := uc.tasks.GetForUpdate(txCtx, cmd.TaskID)
		if err != nil {
			return err
		}

		role, err := uc.teams.Membership(txCtx, current.TeamID, cmd.UserID)
		if err != nil {
			return err
		}

		scope := domain.TaskScope(role, cmd.UserID, current.CreatedBy, current.AssigneeID)
		if scope == domain.TaskEditDenied {
			return domain.ErrForbidden
		}
		if cmd.ExpectedVersion != nil && *cmd.ExpectedVersion != current.Version {
			return domain.ErrVersionConflict
		}

		updated := *current
		if cmd.Title != nil {
			updated.Title = strings.TrimSpace(*cmd.Title)
		}
		if cmd.Description != nil {
			updated.Description = *cmd.Description
		}
		if cmd.Status != nil {
			updated.Status = *cmd.Status
		}
		if cmd.AssigneeID != nil {
			if *cmd.AssigneeID == 0 {
				updated.AssigneeID = nil
			} else {
				assigneeID := *cmd.AssigneeID
				updated.AssigneeID = &assigneeID
			}
		}

		if updated.Status != current.Status {
			if updated.Status == domain.TaskStatusDone {
				now := time.Now().UTC()
				updated.ClosedAt = &now
			} else {
				updated.ClosedAt = nil
			}
		}

		changes := diffTask(current, &updated)
		if len(changes) == 0 {
			result = current
			return nil
		}
		for _, change := range changes {
			if !scope.Allows(change.Field) {
				return domain.ErrForbidden
			}
		}

		if updated.AssigneeID != nil {
			if _, err := uc.teams.Membership(txCtx, updated.TeamID, *updated.AssigneeID); err != nil {
				if errors.Is(err, domain.ErrTeamNotFound) {
					return domain.ErrValidation.WithDetails(map[string]string{
						"assignee_id": "исполнитель должен быть участником команды",
					})
				}
				return err
			}
		}

		now := time.Now().UTC()
		updated.UpdatedAt = now
		updated.Version = current.Version + 1
		if err := uc.tasks.Update(txCtx, &updated, current.Version); err != nil {
			return err
		}
		if err := uc.history.Add(txCtx, &domain.TaskHistoryEntry{
			TaskID:    updated.ID,
			ChangedBy: cmd.UserID,
			Changes:   changes,
			CreatedAt: now,
		}); err != nil {
			return err
		}

		result = &updated
		changed = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	if changed {
		uc.cache.InvalidateTeam(ctx, result.TeamID)
	}
	return result, nil
}

func (uc *TaskUseCase) History(ctx context.Context, userID, taskID int64) ([]domain.TaskHistoryEntry, error) {
	task, err := uc.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := uc.teams.Membership(ctx, task.TeamID, userID); err != nil {
		return nil, err
	}
	return uc.history.List(ctx, taskID)
}

func (uc *TaskUseCase) AddComment(ctx context.Context, cmd AddCommentRequest) (*domain.TaskComment, error) {
	content := strings.TrimSpace(cmd.Content)
	if content == "" {
		return nil, domain.ErrValidation.WithDetails(map[string]string{"content": "поле обязательно"})
	}
	if utf8.RuneCountInString(content) > maxTextLength {
		return nil, domain.ErrValidation.WithDetails(map[string]string{
			"content": fmt.Sprintf("не более %d символов", maxTextLength),
		})
	}

	task, err := uc.tasks.GetByID(ctx, cmd.TaskID)
	if err != nil {
		return nil, err
	}
	if _, err := uc.teams.Membership(ctx, task.TeamID, cmd.UserID); err != nil {
		return nil, err
	}

	comment := &domain.TaskComment{
		TaskID:    cmd.TaskID,
		UserID:    cmd.UserID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
	if err := uc.comments.Add(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (uc *TaskUseCase) Comments(ctx context.Context, query ListCommentsQuery) (*CommentListResult, error) {
	limit, offset, err := normalizePagination(query.Limit, query.Offset)
	if err != nil {
		return nil, err
	}

	task, err := uc.tasks.GetByID(ctx, query.TaskID)
	if err != nil {
		return nil, err
	}
	if _, err := uc.teams.Membership(ctx, task.TeamID, query.UserID); err != nil {
		return nil, err
	}

	comments, total, err := uc.comments.List(ctx, query.TaskID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &CommentListResult{Comments: comments, Total: total, Limit: limit, Offset: offset}, nil
}

func normalizePagination(limit, offset int) (int, int, error) {
	if limit == 0 {
		limit = defaultPageLimit
	}
	details := make(map[string]string)
	if limit < 1 || limit > maxPageLimit {
		details["limit"] = fmt.Sprintf("от 1 до %d", maxPageLimit)
	}
	if offset < 0 {
		details["offset"] = "не может быть отрицательным"
	}
	if len(details) > 0 {
		return 0, 0, domain.ErrValidation.WithDetails(details)
	}
	return limit, offset, nil
}

func taskFilterKey(status *domain.TaskStatus, assigneeID *int64, limit, offset int) string {
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	assigneeValue := ""
	if assigneeID != nil {
		assigneeValue = strconv.FormatInt(*assigneeID, 10)
	}
	return fmt.Sprintf("status=%s;assignee=%s;limit=%d;offset=%d", statusValue, assigneeValue, limit, offset)
}

func validateTaskUpdate(cmd UpdateTaskRequest) error {
	details := make(map[string]string)

	if cmd.TaskID <= 0 {
		details["id"] = "должен быть больше 0"
	}
	if cmd.Title != nil {
		title := strings.TrimSpace(*cmd.Title)
		switch {
		case title == "":
			details["title"] = "поле обязательно"
		case utf8.RuneCountInString(title) > maxTitleLength:
			details["title"] = fmt.Sprintf("не более %d символов", maxTitleLength)
		}
	}
	if cmd.Description != nil && utf8.RuneCountInString(*cmd.Description) > maxTextLength {
		details["description"] = fmt.Sprintf("не более %d символов", maxTextLength)
	}
	if cmd.Status != nil && !cmd.Status.Valid() {
		details["status"] = "недопустимый статус"
	}
	if cmd.AssigneeID != nil && *cmd.AssigneeID < 0 {
		details["assignee_id"] = "не может быть отрицательным"
	}
	if cmd.ExpectedVersion == nil {
		details["version"] = "поле обязательно"
	} else if *cmd.ExpectedVersion <= 0 {
		details["version"] = "должен быть больше 0"
	}
	if len(details) > 0 {
		return domain.ErrValidation.WithDetails(details)
	}
	return nil
}

func creationChanges(task *domain.Task) []domain.TaskChange {
	changes := []domain.TaskChange{
		{Field: "title", Old: nil, New: task.Title},
		{Field: "description", Old: nil, New: task.Description},
		{Field: "status", Old: nil, New: string(task.Status)},
	}
	if task.AssigneeID != nil {
		changes = append(changes, domain.TaskChange{Field: "assignee_id", Old: nil, New: *task.AssigneeID})
	}
	if task.ClosedAt != nil {
		changes = append(changes, domain.TaskChange{Field: "closed_at", Old: nil, New: formatTime(task.ClosedAt)})
	}
	return changes
}

func diffTask(before, after *domain.Task) []domain.TaskChange {
	changes := make([]domain.TaskChange, 0, 5)

	if before.Title != after.Title {
		changes = append(changes, domain.TaskChange{Field: "title", Old: before.Title, New: after.Title})
	}
	if before.Description != after.Description {
		changes = append(changes, domain.TaskChange{Field: "description", Old: before.Description, New: after.Description})
	}
	if before.Status != after.Status {
		changes = append(changes, domain.TaskChange{
			Field: "status",
			Old:   string(before.Status),
			New:   string(after.Status),
		})
	}
	if !equalInt64Ptr(before.AssigneeID, after.AssigneeID) {
		changes = append(changes, domain.TaskChange{
			Field: "assignee_id",
			Old:   int64Value(before.AssigneeID),
			New:   int64Value(after.AssigneeID),
		})
	}
	if !equalTimePtr(before.ClosedAt, after.ClosedAt) {
		changes = append(changes, domain.TaskChange{
			Field: "closed_at",
			Old:   formatTime(before.ClosedAt),
			New:   formatTime(after.ClosedAt),
		})
	}
	return changes
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func int64Value(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func formatTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
