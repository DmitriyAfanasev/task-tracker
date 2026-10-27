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

// --- helpers ---

func taskFixture(t *testing.T, role domain.Role) (*TaskUseCase, *MockTaskRepository, *MockTaskHistoryRepository, *MockTaskCommentRepository, *MockTeamRepository, *MockTaskListCache, *gomock.Controller) {
	t.Helper()

	ctrl := gomock.NewController(t)
	tasks := NewMockTaskRepository(ctrl)
	history := NewMockTaskHistoryRepository(ctrl)
	comments := NewMockTaskCommentRepository(ctrl)
	teams := NewMockTeamRepository(ctrl)
	cache := NewMockTaskListCache(ctrl)

	task := &domain.Task{
		ID:         10,
		TeamID:     1,
		Title:      "Task",
		Status:     domain.TaskStatusTodo,
		CreatedBy:  1,
		AssigneeID: new(int64(2)),
		Version:    1,
	}

	tasks.EXPECT().GetByID(gomock.Any(), int64(10)).Return(task, nil).AnyTimes()
	tasks.EXPECT().GetForUpdate(gomock.Any(), int64(10)).Return(task, nil).AnyTimes()
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(role, nil).AnyTimes()

	uc := NewTaskUseCase(tasks, history, comments, teams, cache, immediateTx{})
	return uc, tasks, history, comments, teams, cache, ctrl
}

// --- tests ---

func TestUpdateTaskBumpsVersionAndWritesHistory(t *testing.T) {
	uc, tasks, history, _, _, cache, _ := taskFixture(t, domain.RoleMember)
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
	uc, _, _, _, _, _, _ := taskFixture(t, domain.RoleMember)
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
	uc, tasks, history, _, _, cache, _ := taskFixture(t, domain.RoleOwner)
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
	uc, _, _, _, _, _, _ := taskFixture(t, domain.RoleOwner)
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
	uc, _, _, _, _, _, _ := taskFixture(t, domain.RoleOwner)
	status := domain.TaskStatusInProgress

	_, err := uc.Update(context.Background(), UpdateTaskRequest{
		UserID: 1,
		TaskID: 10,
		Status: &status,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestNoopUpdateDoesNotBumpVersion(t *testing.T) {
	uc, _, _, _, _, _, _ := taskFixture(t, domain.RoleOwner)
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
