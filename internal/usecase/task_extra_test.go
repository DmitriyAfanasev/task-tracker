package usecase

import (
	"context"
	"errors"
	"testing"

	"secunda/internal/domain"
)

// countingTeamRepo tracks membership call count and allows custom behavior.
type countingTeamRepo struct {
	role         domain.Role
	membershipFn func(ctx context.Context, teamID, userID int64) (domain.Role, error)
}

func (c *countingTeamRepo) Create(context.Context, *domain.Team) error { return nil }
func (c *countingTeamRepo) ListByUser(context.Context, int64) ([]domain.TeamWithRole, error) {
	return nil, nil
}
func (c *countingTeamRepo) ListMembers(context.Context, int64) ([]domain.TeamMemberInfo, error) {
	return nil, nil
}
func (c *countingTeamRepo) Membership(ctx context.Context, teamID, userID int64) (domain.Role, error) {
	if c.membershipFn != nil {
		return c.membershipFn(ctx, teamID, userID)
	}
	return c.role, nil
}
func (c *countingTeamRepo) AddMember(context.Context, int64, int64, domain.Role) error { return nil }

// --- Create ---

func TestCreateTaskRejectsZeroTeamID(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		Title:  "Task",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestCreateTaskRejectsEmptyTitle(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestCreateTaskRejectsLongTitle(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  string(make([]rune, 256)),
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestCreateTaskRejectsLongDescription(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	long := string(make([]rune, 5001))
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID:      1,
		TeamID:      1,
		Title:       "Task",
		Description: &long,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestCreateTaskRejectsInvalidStatus(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	status := domain.TaskStatus("invalid")
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "Task",
		Status: &status,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestCreateTaskChecksTeamMembership(t *testing.T) {
	repo := &fakeTeamRepository{role: domain.Role("none"), membershipErr: domain.ErrForbidden}
	uc := NewTaskUseCase(&fakeTaskRepository{}, &fakeHistoryRepository{}, &fakeCommentRepository{}, repo, &fakeTaskCache{}, immediateTx{})
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "Task",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestCreateTaskSetsDefaultStatus(t *testing.T) {
	uc, tasks, history, cache := taskFixture(domain.RoleMember)
	task, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "New Task",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.Status != domain.TaskStatusTodo {
		t.Fatalf("status = %v, want todo", task.Status)
	}
	if task.Version != 1 {
		t.Fatalf("version = %d, want 1", task.Version)
	}
	if tasks.task.Title != "New Task" {
		t.Fatalf("title = %q, want New Task", tasks.task.Title)
	}
	if len(history.entries) != 1 {
		t.Fatalf("history entries = %d, want 1", len(history.entries))
	}
	if !cache.invalidated {
		t.Fatal("cache should be invalidated")
	}
}

func TestCreateTaskSetsClosedAtForDoneStatus(t *testing.T) {
	uc, tasks, _, _ := taskFixture(domain.RoleMember)
	status := domain.TaskStatusDone
	task, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "Done Task",
		Status: &status,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.ClosedAt == nil {
		t.Fatal("closed_at should be set for done status")
	}
	_ = tasks
}

func TestCreateTaskTrimsTitle(t *testing.T) {
	uc, tasks, _, _ := taskFixture(domain.RoleMember)
	_, err := uc.Create(context.Background(), CreateTaskRequest{
		UserID: 1,
		TeamID: 1,
		Title:  "  Trimmed  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if tasks.task.Title != "Trimmed" {
		t.Fatalf("title = %q, want Trimmed", tasks.task.Title)
	}
}

// --- List ---

func TestListTasksRejectsZeroTeamID(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	_, err := uc.List(context.Background(), ListTasksQuery{
		UserID: 1,
		TeamID: 0,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestListTasksRejectsInvalidStatus(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	status := domain.TaskStatus("invalid")
	_, err := uc.List(context.Background(), ListTasksQuery{
		UserID: 1,
		TeamID: 1,
		Status: &status,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestListTasksRejectsZeroAssigneeID(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	assignee := int64(0)
	_, err := uc.List(context.Background(), ListTasksQuery{
		UserID:     1,
		TeamID:     1,
		AssigneeID: &assignee,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestListTasksChecksTeamMembership(t *testing.T) {
	repo := &fakeTeamRepository{role: domain.RoleMember, membershipErr: domain.ErrForbidden}
	uc := NewTaskUseCase(&fakeTaskRepository{}, &fakeHistoryRepository{}, &fakeCommentRepository{}, repo, &fakeTaskCache{}, immediateTx{})
	_, err := uc.List(context.Background(), ListTasksQuery{
		UserID: 1,
		TeamID: 1,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

// --- History ---

func TestHistoryChecksTeamMembership(t *testing.T) {
	repo := &fakeTeamRepository{role: domain.RoleMember, membershipErr: domain.ErrForbidden}
	uc := NewTaskUseCase(&fakeTaskRepository{}, &fakeHistoryRepository{}, &fakeCommentRepository{}, repo, &fakeTaskCache{}, immediateTx{})
	_, err := uc.History(context.Background(), 1, 10)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

// --- AddComment ---

func TestAddCommentRejectsEmptyContent(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	_, err := uc.AddComment(context.Background(), AddCommentRequest{
		UserID:  1,
		TaskID:  10,
		Content: "",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestAddCommentRejectsLongContent(t *testing.T) {
	uc, _, _, _ := taskFixture(domain.RoleMember)
	long := string(make([]rune, 5001))
	_, err := uc.AddComment(context.Background(), AddCommentRequest{
		UserID:  1,
		TaskID:  10,
		Content: long,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestAddCommentChecksTeamMembership(t *testing.T) {
	repo := &fakeTeamRepository{role: domain.RoleMember, membershipErr: domain.ErrForbidden}
	uc := NewTaskUseCase(&fakeTaskRepository{}, &fakeHistoryRepository{}, &fakeCommentRepository{}, repo, &fakeTaskCache{}, immediateTx{})
	_, err := uc.AddComment(context.Background(), AddCommentRequest{
		UserID:  1,
		TaskID:  10,
		Content: "comment",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

// --- Comments ---

func TestCommentsChecksTeamMembership(t *testing.T) {
	repo := &fakeTeamRepository{role: domain.RoleMember, membershipErr: domain.ErrForbidden}
	uc := NewTaskUseCase(&fakeTaskRepository{}, &fakeHistoryRepository{}, &fakeCommentRepository{}, repo, &fakeTaskCache{}, immediateTx{})
	_, err := uc.Comments(context.Background(), ListCommentsQuery{
		UserID: 1,
		TaskID: 10,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

// --- normalizePagination ---

func TestNormalizePaginationDefaultsLimit(t *testing.T) {
	limit, offset, err := normalizePagination(0, 0)
	if err != nil {
		t.Fatalf("normalizePagination(0, 0) error = %v", err)
	}
	if limit != defaultPageLimit {
		t.Fatalf("limit = %d, want %d", limit, defaultPageLimit)
	}
	if offset != 0 {
		t.Fatalf("offset = %d, want 0", offset)
	}
}

func TestNormalizePaginationAcceptsLimitAtMax(t *testing.T) {
	limit, offset, err := normalizePagination(100, 0)
	if err != nil {
		t.Fatalf("normalizePagination(100, 0) error = %v", err)
	}
	if limit != 100 {
		t.Fatalf("limit = %d, want 100", limit)
	}
	if offset != 0 {
		t.Fatalf("offset = %d, want 0", offset)
	}
}

func TestNormalizePaginationRejectsLimitTooHigh(t *testing.T) {
	_, _, err := normalizePagination(101, 0)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestNormalizePaginationRejectsNegativeLimit(t *testing.T) {
	_, _, err := normalizePagination(-1, 0)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestNormalizePaginationRejectsNegativeOffset(t *testing.T) {
	_, _, err := normalizePagination(10, -1)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestNormalizePaginationAcceptsValidValues(t *testing.T) {
	limit, offset, err := normalizePagination(50, 10)
	if err != nil {
		t.Fatalf("normalizePagination(50, 10) error = %v", err)
	}
	if limit != 50 || offset != 10 {
		t.Fatalf("limit = %d, offset = %d, want 50, 10", limit, offset)
	}
}

// --- taskFilterKey ---

func TestTaskFilterKeyWithAllParams(t *testing.T) {
	status := domain.TaskStatusTodo
	assignee := int64(42)
	key := taskFilterKey(&status, &assignee, 20, 10)
	want := "status=todo;assignee=42;limit=20;offset=10"
	if key != want {
		t.Fatalf("filterKey = %q, want %q", key, want)
	}
}

func TestTaskFilterKeyWithNilParams(t *testing.T) {
	key := taskFilterKey(nil, nil, 20, 0)
	want := "status=;assignee=;limit=20;offset=0"
	if key != want {
		t.Fatalf("filterKey = %q, want %q", key, want)
	}
}
