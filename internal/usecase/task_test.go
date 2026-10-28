package usecase

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"secunda/internal/domain"
)

type immediateTx struct{}

func (immediateTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// --- Update fixtures ---

func newUpdateUC(t *testing.T, role domain.Role, task *domain.Task) (*TaskUseCase, *MockTaskRepository, *MockTaskHistoryRepository, *MockTaskListCache) {
	t.Helper()

	ctrl := gomock.NewController(t)
	tasks := NewMockTaskRepository(ctrl)
	history := NewMockTaskHistoryRepository(ctrl)
	teams := NewMockTeamRepository(ctrl)
	cache := NewMockTaskListCache(ctrl)

	if task == nil {
		task = &domain.Task{
			ID:         10,
			TeamID:     1,
			Title:      "Task",
			Status:     domain.TaskStatusTodo,
			CreatedBy:  1,
			AssigneeID: new(int64(2)),
			Version:    1,
		}
	}

	tasks.EXPECT().GetForUpdate(gomock.Any(), int64(10)).Return(task, nil).AnyTimes()
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(role, nil).AnyTimes()

	uc := NewTaskUseCase(tasks, history, NewMockTaskCommentRepository(ctrl), teams, cache, immediateTx{})
	return uc, tasks, history, cache
}

// --- Create fixtures ---

func newCreateUC(t *testing.T, role domain.Role) (*TaskUseCase, *MockTaskRepository, *MockTaskHistoryRepository, *MockTaskListCache) {
	t.Helper()

	ctrl := gomock.NewController(t)
	tasks := NewMockTaskRepository(ctrl)
	history := NewMockTaskHistoryRepository(ctrl)
	teams := NewMockTeamRepository(ctrl)
	cache := NewMockTaskListCache(ctrl)

	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(role, nil).AnyTimes()

	uc := NewTaskUseCase(tasks, history, NewMockTaskCommentRepository(ctrl), teams, cache, immediateTx{})
	return uc, tasks, history, cache
}

// --- List fixtures ---

func newListUC(t *testing.T, role domain.Role) (*TaskUseCase, *MockTeamRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	teams := NewMockTeamRepository(ctrl)

	// Membership не ставим — тесты валидации падают до вызова membership,
	// а TestListTasksChecksTeamMembership переопределяет её на ErrForbidden.

	uc := NewTaskUseCase(NewMockTaskRepository(ctrl), NewMockTaskHistoryRepository(ctrl), NewMockTaskCommentRepository(ctrl), teams, NewMockTaskListCache(ctrl), immediateTx{})
	return uc, teams
}

// --- History fixtures ---

func newHistoryUC(t *testing.T, role domain.Role, task *domain.Task) (*TaskUseCase, *MockTaskRepository, *MockTeamRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	tasks := NewMockTaskRepository(ctrl)
	teams := NewMockTeamRepository(ctrl)

	if task == nil {
		task = &domain.Task{ID: 10, TeamID: 1}
	}

	tasks.EXPECT().GetByID(gomock.Any(), int64(10)).Return(task, nil).AnyTimes()
	// Membership не ставим — тесты переопределяют её на ErrForbidden.

	uc := NewTaskUseCase(tasks, NewMockTaskHistoryRepository(ctrl), NewMockTaskCommentRepository(ctrl), teams, NewMockTaskListCache(ctrl), immediateTx{})
	return uc, tasks, teams
}

// --- AddComment fixtures ---

func newAddCommentUC(t *testing.T, role domain.Role, task *domain.Task) (*TaskUseCase, *MockTaskRepository, *MockTeamRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	tasks := NewMockTaskRepository(ctrl)
	teams := NewMockTeamRepository(ctrl)

	if task == nil {
		task = &domain.Task{ID: 10, TeamID: 1}
	}

	tasks.EXPECT().GetByID(gomock.Any(), int64(10)).Return(task, nil).AnyTimes()
	// Membership не ставим — тесты переопределяют её на ErrForbidden.

	uc := NewTaskUseCase(tasks, NewMockTaskHistoryRepository(ctrl), NewMockTaskCommentRepository(ctrl), teams, NewMockTaskListCache(ctrl), immediateTx{})
	return uc, tasks, teams
}

// --- Comments fixtures ---

func newCommentsUC(t *testing.T, role domain.Role, task *domain.Task) (*TaskUseCase, *MockTaskRepository, *MockTeamRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	tasks := NewMockTaskRepository(ctrl)
	teams := NewMockTeamRepository(ctrl)

	if task == nil {
		task = &domain.Task{ID: 10, TeamID: 1}
	}

	tasks.EXPECT().GetByID(gomock.Any(), int64(10)).Return(task, nil).AnyTimes()
	// Membership не ставим — тесты переопределяют её на ErrForbidden.

	uc := NewTaskUseCase(tasks, NewMockTaskHistoryRepository(ctrl), NewMockTaskCommentRepository(ctrl), teams, NewMockTaskListCache(ctrl), immediateTx{})
	return uc, tasks, teams
}

// --- Update tests ---

func TestUpdateTaskBumpsVersionAndWritesHistory(t *testing.T) {
	uc, tasks, history, cache := newUpdateUC(t, domain.RoleMember, nil)
	status := domain.TaskStatusInProgress
	version := int64(1)

	tasks.EXPECT().Update(gomock.Any(), gomock.Any(), int64(1)).Return(nil)
	history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
	cache.EXPECT().InvalidateTeam(gomock.Any(), int64(1)).Return()

	task, err := uc.Update(context.Background(), UpdateTaskRequest{
		UserID:          2,
		TaskID:          10,
		Status:          &status,
		ExpectedVersion: &version,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(2), task.Version)
}

func TestAssigneeCannotEditTitle(t *testing.T) {
	uc, _, _, _ := newUpdateUC(t, domain.RoleMember, nil)
	title := "Hacked"
	version := int64(1)

	_, err := uc.Update(context.Background(), UpdateTaskRequest{
		UserID:          2,
		TaskID:          10,
		Title:           &title,
		ExpectedVersion: &version,
	})

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestOwnerCanEditTask(t *testing.T) {
	uc, tasks, history, cache := newUpdateUC(t, domain.RoleOwner, nil)
	title := "Renamed"
	version := int64(1)

	tasks.EXPECT().Update(gomock.Any(), gomock.Any(), int64(1)).Return(nil)
	history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
	cache.EXPECT().InvalidateTeam(gomock.Any(), int64(1)).Return()

	task, err := uc.Update(context.Background(), UpdateTaskRequest{
		UserID:          1,
		TaskID:          10,
		Title:           &title,
		ExpectedVersion: &version,
	})

	assert.NoError(t, err)
	assert.Equal(t, "Renamed", task.Title)
}

func TestStaleVersionConflicts(t *testing.T) {
	uc, _, _, _ := newUpdateUC(t, domain.RoleOwner, nil)
	status := domain.TaskStatusInProgress
	version := int64(99)

	_, err := uc.Update(context.Background(), UpdateTaskRequest{
		UserID:          1,
		TaskID:          10,
		Status:          &status,
		ExpectedVersion: &version,
	})

	assert.ErrorIs(t, err, domain.ErrVersionConflict)
}

func TestUpdateRequiresVersion(t *testing.T) {
	uc, _, _, _ := newUpdateUC(t, domain.RoleOwner, nil)
	status := domain.TaskStatusInProgress

	_, err := uc.Update(context.Background(), UpdateTaskRequest{
		UserID: 1,
		TaskID: 10,
		Status: &status,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestNoopUpdateDoesNotBumpVersion(t *testing.T) {
	uc, _, _, _ := newUpdateUC(t, domain.RoleOwner, nil)
	status := domain.TaskStatusTodo
	version := int64(1)

	task, err := uc.Update(context.Background(), UpdateTaskRequest{
		UserID:          1,
		TaskID:          10,
		Status:          &status,
		ExpectedVersion: &version,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(1), task.Version)
}
