package usecase

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"secunda/internal/domain"
)

// --- Create ---

func TestCreateTaskRejectsZeroTeamID(t *testing.T) {
	uc, _, _, _ := newCreateUC(t, domain.RoleMember)
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		Title:  "Task",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateTaskRejectsEmptyTitle(t *testing.T) {
	uc, _, _, _ := newCreateUC(t, domain.RoleMember)
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateTaskRejectsLongTitle(t *testing.T) {
	uc, _, _, _ := newCreateUC(t, domain.RoleMember)
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  string(make([]rune, 256)),
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateTaskRejectsLongDescription(t *testing.T) {
	uc, _, _, _ := newCreateUC(t, domain.RoleMember)
	long := string(make([]rune, 5001))
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID:      1,
		TeamID:      1,
		Title:       "Task",
		Description: &long,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateTaskRejectsInvalidStatus(t *testing.T) {
	uc, _, _, _ := newCreateUC(t, domain.RoleMember)
	status := domain.TaskStatus("invalid")
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "Task",
		Status: &status,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateTaskChecksTeamMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	teams := NewMockTeamRepository(ctrl)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleMember, domain.ErrForbidden)
	uc := NewTaskUseCase(NewMockTaskRepository(ctrl), NewMockTaskHistoryRepository(ctrl), NewMockTaskCommentRepository(ctrl), teams, NewMockTaskListCache(ctrl), immediateTx{})

	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "Task",
	})

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestCreateTaskWritesHistoryEntry(t *testing.T) {
	uc, tasks, history, cache := newCreateUC(t, domain.RoleMember)
	tasks.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, t *domain.Task) error {
		t.ID = 1
		return nil
	}).Times(1)
	history.EXPECT().Add(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, entry *domain.TaskHistoryEntry) error {
		assert.NotEmpty(t, entry.Changes)
		assert.Equal(t, "title", entry.Changes[0].Field)
		return nil
	}).Times(1)
	cache.EXPECT().InvalidateTeam(gomock.Any(), int64(1)).Return().Times(1)

	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "New Task",
	})

	assert.NoError(t, err)
}

func TestCreateTaskSetsDefaultStatus(t *testing.T) {
	uc, tasks, history, cache := newCreateUC(t, domain.RoleMember)
	tasks.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, t *domain.Task) error {
		t.ID = 1
		t.Status = domain.TaskStatusTodo
		return nil
	}).Times(1)
	history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	cache.EXPECT().InvalidateTeam(gomock.Any(), int64(1)).Return().Times(1)

	task, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "New Task",
	})

	assert.NoError(t, err)
	assert.Equal(t, domain.TaskStatusTodo, task.Status)
	assert.Equal(t, int64(1), task.Version)
	assert.Equal(t, "New Task", task.Title)
}

func TestCreateTaskSetsClosedAtForDoneStatus(t *testing.T) {
	uc, tasks, history, cache := newCreateUC(t, domain.RoleMember)
	status := domain.TaskStatusDone
	tasks.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, t *domain.Task) error {
		t.ID = 1
		t.Status = status
		return nil
	}).Times(1)
	history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	cache.EXPECT().InvalidateTeam(gomock.Any(), int64(1)).Return().Times(1)

	task, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "Done Task",
		Status: &status,
	})

	assert.NoError(t, err)
	assert.NotNil(t, task.ClosedAt)
}

func TestCreateTaskTrimsTitle(t *testing.T) {
	uc, tasks, history, cache := newCreateUC(t, domain.RoleMember)
	var capturedTitle string
	tasks.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, t *domain.Task) error {
		capturedTitle = t.Title
		t.ID = 1
		return nil
	}).Times(1)
	history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	cache.EXPECT().InvalidateTeam(gomock.Any(), int64(1)).Return().Times(1)

	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "  Trimmed  ",
	})

	assert.NoError(t, err)
	assert.Equal(t, "Trimmed", capturedTitle)
}

// --- List ---

func TestListTasksRejectsZeroTeamID(t *testing.T) {
	uc, _ := newListUC(t, domain.RoleMember)
	_, err := uc.List(context.Background(), ListTasksQuery{
		UserID: 1,
		TeamID: 0,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestListTasksRejectsInvalidStatus(t *testing.T) {
	uc, _ := newListUC(t, domain.RoleMember)
	status := domain.TaskStatus("invalid")
	_, err := uc.List(context.Background(), ListTasksQuery{
		UserID: 1,
		TeamID: 1,
		Status: &status,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestListTasksRejectsZeroAssigneeID(t *testing.T) {
	uc, _ := newListUC(t, domain.RoleMember)
	assignee := int64(0)
	_, err := uc.List(context.Background(), ListTasksQuery{
		UserID:     1,
		TeamID:     1,
		AssigneeID: &assignee,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestListTasksChecksTeamMembership(t *testing.T) {
	uc, teams := newListUC(t, domain.RoleMember)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleMember, domain.ErrForbidden)

	_, err := uc.List(context.Background(), ListTasksQuery{
		UserID: 1,
		TeamID: 1,
	})

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- History ---

func TestHistoryChecksTeamMembership(t *testing.T) {
	uc, _, teams := newHistoryUC(t, domain.RoleMember, nil)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleMember, domain.ErrForbidden)

	_, err := uc.History(context.Background(), 1, 10)

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- AddComment ---

func TestAddCommentRejectsEmptyContent(t *testing.T) {
	uc, _, _ := newAddCommentUC(t, domain.RoleMember, nil)
	_, err := uc.AddComment(context.Background(), AddCommentRequest{
		UserID:  1,
		TaskID:  10,
		Content: "",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestAddCommentRejectsLongContent(t *testing.T) {
	uc, _, _ := newAddCommentUC(t, domain.RoleMember, nil)
	long := string(make([]rune, 5001))
	_, err := uc.AddComment(context.Background(), AddCommentRequest{
		UserID:  1,
		TaskID:  10,
		Content: long,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestAddCommentChecksTeamMembership(t *testing.T) {
	uc, _, teams := newAddCommentUC(t, domain.RoleMember, nil)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleMember, domain.ErrForbidden)

	_, err := uc.AddComment(context.Background(), AddCommentRequest{
		UserID:  1,
		TaskID:  10,
		Content: "comment",
	})

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- Comments ---

func TestCommentsChecksTeamMembership(t *testing.T) {
	uc, _, teams := newCommentsUC(t, domain.RoleMember, nil)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleMember, domain.ErrForbidden)

	_, err := uc.Comments(context.Background(), ListCommentsQuery{
		UserID: 1,
		TaskID: 10,
	})

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- normalizePagination ---

func TestNormalizePaginationDefaultsLimit(t *testing.T) {
	limit, offset, err := normalizePagination(0, 0)

	assert.NoError(t, err)
	assert.Equal(t, defaultPageLimit, limit)
	assert.Equal(t, 0, offset)
}

func TestNormalizePaginationAcceptsLimitAtMax(t *testing.T) {
	limit, offset, err := normalizePagination(100, 0)

	assert.NoError(t, err)
	assert.Equal(t, 100, limit)
	assert.Equal(t, 0, offset)
}

func TestNormalizePaginationRejectsLimitTooHigh(t *testing.T) {
	_, _, err := normalizePagination(101, 0)

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestNormalizePaginationRejectsNegativeLimit(t *testing.T) {
	_, _, err := normalizePagination(-1, 0)

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestNormalizePaginationRejectsNegativeOffset(t *testing.T) {
	_, _, err := normalizePagination(10, -1)

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestNormalizePaginationAcceptsValidValues(t *testing.T) {
	limit, offset, err := normalizePagination(50, 10)

	assert.NoError(t, err)
	assert.Equal(t, 50, limit)
	assert.Equal(t, 10, offset)
}

// --- taskFilterKey ---

func TestTaskFilterKeyWithAllParams(t *testing.T) {
	status := domain.TaskStatusTodo
	assignee := int64(42)
	key := taskFilterKey(&status, &assignee, 20, 10)

	assert.Equal(t, "status=todo;assignee=42;limit=20;offset=10", key)
}

func TestTaskFilterKeyWithNilParams(t *testing.T) {
	key := taskFilterKey(nil, nil, 20, 0)

	assert.Equal(t, "status=;assignee=;limit=20;offset=0", key)
}
