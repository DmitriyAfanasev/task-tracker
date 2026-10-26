package usecase

import (
	"context"
	"errors"
	"testing"

	"secunda/internal/domain"
)

type fakeTaskRepository struct {
	task       domain.Task
	updates    int
	expected   int64
	createCall int
}

func (f *fakeTaskRepository) Create(_ context.Context, task *domain.Task) error {
	f.createCall++
	task.ID = 1
	f.task = *task
	return nil
}

func (f *fakeTaskRepository) GetByID(_ context.Context, _ int64) (*domain.Task, error) {
	copy := f.task
	return &copy, nil
}

func (f *fakeTaskRepository) GetForUpdate(_ context.Context, _ int64) (*domain.Task, error) {
	copy := f.task
	return &copy, nil
}

func (f *fakeTaskRepository) Update(_ context.Context, task *domain.Task, expectedVersion int64) error {
	if expectedVersion != f.task.Version {
		return domain.ErrVersionConflict
	}
	f.updates++
	f.expected = expectedVersion
	f.task = *task
	return nil
}

func (f *fakeTaskRepository) List(context.Context, TaskFilter) ([]domain.Task, int, error) {
	return nil, 0, errors.New("not implemented")
}

type fakeHistoryRepository struct {
	entries []domain.TaskHistoryEntry
}

func (f *fakeHistoryRepository) Add(_ context.Context, entry *domain.TaskHistoryEntry) error {
	entry.ID = int64(len(f.entries) + 1)
	f.entries = append(f.entries, *entry)
	return nil
}

func (f *fakeHistoryRepository) List(context.Context, int64) ([]domain.TaskHistoryEntry, error) {
	return f.entries, nil
}

type fakeCommentRepository struct{}

func (f *fakeCommentRepository) Add(context.Context, *domain.TaskComment) error { return nil }
func (f *fakeCommentRepository) List(context.Context, int64, int, int) ([]domain.TaskComment, int, error) {
	return nil, 0, nil
}

type fakeTeamRepository struct {
	role          domain.Role
	membershipErr error
}

func (f *fakeTeamRepository) Create(context.Context, *domain.Team) error { return nil }
func (f *fakeTeamRepository) ListByUser(context.Context, int64) ([]domain.TeamWithRole, error) {
	return nil, nil
}
func (f *fakeTeamRepository) Membership(context.Context, int64, int64) (domain.Role, error) {
	return f.role, f.membershipErr
}
func (f *fakeTeamRepository) AddMember(context.Context, int64, int64, domain.Role) error {
	return nil
}

type fakeTaskCache struct {
	invalidated bool
}

func (f *fakeTaskCache) Get(context.Context, int64, string) (TaskListResult, int64, bool) {
	return TaskListResult{}, 0, false
}
func (f *fakeTaskCache) Set(context.Context, int64, string, int64, TaskListResult) {}
func (f *fakeTaskCache) InvalidateTeam(context.Context, int64)                     { f.invalidated = true }

type immediateTx struct{}

func (immediateTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func taskFixture(role domain.Role) (*TaskUseCase, *fakeTaskRepository, *fakeHistoryRepository, *fakeTaskCache) {
	assignee := int64(2)
	tasks := &fakeTaskRepository{task: domain.Task{
		ID:         10,
		TeamID:     1,
		Title:      "Task",
		Status:     domain.TaskStatusTodo,
		CreatedBy:  1,
		AssigneeID: &assignee,
		Version:    1,
	}}
	history := &fakeHistoryRepository{}
	cache := &fakeTaskCache{}

	uc := NewTaskUseCase(tasks, history, &fakeCommentRepository{}, &fakeTeamRepository{role: role}, cache, immediateTx{})
	return uc, tasks, history, cache
}

func TestUpdateTaskBumpsVersionAndWritesHistory(t *testing.T) {
	uc, tasks, history, cache := taskFixture(domain.RoleMember)
	status := domain.TaskStatusInProgress
	version := int64(1)

	task, err := uc.Update(context.Background(), UpdateTaskCommand{
		UserID:          2,
		TaskID:          10,
		Status:          &status,
		ExpectedVersion: &version,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if task.Version != 2 {
		t.Fatalf("version = %d, want 2", task.Version)
	}
	if tasks.task.Status != domain.TaskStatusInProgress {
		t.Fatalf("stored status = %q, want in_progress", tasks.task.Status)
	}
	if len(history.entries) != 1 {
		t.Fatalf("history entries = %d, want 1", len(history.entries))
	}
	if !cache.invalidated {
		t.Fatal("cache was not invalidated")
	}
}

func TestAssigneeCannotEditTitle(t *testing.T) {
	uc, tasks, _, _ := taskFixture(domain.RoleMember)
	title := "Hacked"
	version := int64(1)

	_, err := uc.Update(context.Background(), UpdateTaskCommand{
		UserID:          2,
		TaskID:          10,
		Title:           &title,
		ExpectedVersion: &version,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
	if tasks.updates != 0 {
		t.Fatalf("updates = %d, want 0", tasks.updates)
	}
}

func TestOwnerCanEditTask(t *testing.T) {
	uc, tasks, _, _ := taskFixture(domain.RoleOwner)
	title := "Renamed"
	version := int64(1)

	task, err := uc.Update(context.Background(), UpdateTaskCommand{
		UserID:          1,
		TaskID:          10,
		Title:           &title,
		ExpectedVersion: &version,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if task.Title != "Renamed" {
		t.Fatalf("title = %q, want Renamed", task.Title)
	}
	if tasks.updates != 1 {
		t.Fatalf("updates = %d, want 1", tasks.updates)
	}
}

func TestStaleVersionConflicts(t *testing.T) {
	uc, tasks, _, _ := taskFixture(domain.RoleOwner)
	status := domain.TaskStatusInProgress
	version := int64(99)

	_, err := uc.Update(context.Background(), UpdateTaskCommand{
		UserID:          1,
		TaskID:          10,
		Status:          &status,
		ExpectedVersion: &version,
	})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("Update() error = %v, want ErrVersionConflict", err)
	}
	if tasks.updates != 0 {
		t.Fatalf("updates = %d, want 0", tasks.updates)
	}
}

func TestUpdateRequiresVersion(t *testing.T) {
	uc, tasks, _, _ := taskFixture(domain.RoleOwner)
	status := domain.TaskStatusInProgress

	_, err := uc.Update(context.Background(), UpdateTaskCommand{
		UserID: 1,
		TaskID: 10,
		Status: &status,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Update() error = %v, want ErrValidation", err)
	}
	if tasks.updates != 0 {
		t.Fatalf("updates = %d, want 0", tasks.updates)
	}
}

func TestNoopUpdateDoesNotBumpVersion(t *testing.T) {
	uc, tasks, history, cache := taskFixture(domain.RoleOwner)
	status := domain.TaskStatusTodo
	version := int64(1)

	task, err := uc.Update(context.Background(), UpdateTaskCommand{
		UserID:          1,
		TaskID:          10,
		Status:          &status,
		ExpectedVersion: &version,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if task.Version != 1 || tasks.updates != 0 {
		t.Fatalf("version = %d, updates = %d, want 1 and 0", task.Version, tasks.updates)
	}
	if len(history.entries) != 0 || cache.invalidated {
		t.Fatal("no-op update must not write history or invalidate cache")
	}
}
