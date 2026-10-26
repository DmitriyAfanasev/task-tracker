package httpapi

import (
	"context"

	"secunda/internal/delivery/http/openapi"
	"secunda/internal/domain"
	"secunda/internal/usecase"
)

// AuthUseCase — порт HTTP-слоя: хендлер не зависит от конкретной реализации.
type AuthUseCase interface {
	Register(ctx context.Context, cmd usecase.RegisterRequest) (*usecase.AuthResult, error)
	Login(ctx context.Context, cmd usecase.LoginRequest) (*usecase.AuthResult, error)
}

// TeamUseCase — порт HTTP-слоя для сценариев команд.
type TeamUseCase interface {
	Create(ctx context.Context, cmd usecase.CreateTeamRequest) (*domain.Team, error)
	List(ctx context.Context, userID int64) ([]domain.TeamWithRole, error)
	Invite(ctx context.Context, cmd usecase.InviteMemberRequest) (*usecase.TeamMember, error)
}

// TeamStatsUseCase — сценарий SQL-аналитики команды.
type TeamStatsUseCase interface {
	Get(ctx context.Context, userID, teamID int64) (*domain.TeamStats, error)
}

// TaskUseCase — порт HTTP-слоя для сценариев задач.
type TaskUseCase interface {
	Create(ctx context.Context, cmd usecase.CreateTaskRequest) (*domain.Task, error)
	List(ctx context.Context, query usecase.ListTasksQuery) (*usecase.TaskListResult, error)
	Update(ctx context.Context, cmd usecase.UpdateTaskRequest) (*domain.Task, error)
	History(ctx context.Context, userID, taskID int64) ([]domain.TaskHistoryEntry, error)
	AddComment(ctx context.Context, cmd usecase.AddCommentRequest) (*domain.TaskComment, error)
	Comments(ctx context.Context, query usecase.ListCommentsQuery) (*usecase.CommentListResult, error)
}

// Handler — тонкий адаптер между сгенерированным OpenAPI-контрактом и use case.
// Здесь нет бизнес-правил: только вызов сценария и маппинг результата.
type Handler struct {
	auth  AuthUseCase
	teams TeamUseCase
	stats TeamStatsUseCase
	tasks TaskUseCase
}

var _ openapi.StrictServerInterface = (*Handler)(nil)

func NewHandler(auth AuthUseCase, teams TeamUseCase, stats TeamStatsUseCase, tasks TaskUseCase) *Handler {
	return &Handler{auth: auth, teams: teams, stats: stats, tasks: tasks}
}

func (h *Handler) Register(ctx context.Context, request openapi.RegisterRequestObject) (openapi.RegisterResponseObject, error) {
	if request.Body == nil {
		return nil, domain.ErrBadRequest
	}

	result, err := h.auth.Register(ctx, usecase.RegisterRequest{
		Email:    string(request.Body.Email),
		Password: request.Body.Password,
		Name:     request.Body.Name,
	})
	if err != nil {
		return nil, err
	}

	return openapi.Register201JSONResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      toAPIUser(result.User),
	}, nil
}

func (h *Handler) Login(ctx context.Context, request openapi.LoginRequestObject) (openapi.LoginResponseObject, error) {
	if request.Body == nil {
		return nil, domain.ErrBadRequest
	}

	result, err := h.auth.Login(ctx, usecase.LoginRequest{
		Email:    string(request.Body.Email),
		Password: request.Body.Password,
	})
	if err != nil {
		return nil, err
	}

	return openapi.Login200JSONResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      toAPIUser(result.User),
	}, nil
}

func (h *Handler) CreateTeam(ctx context.Context, request openapi.CreateTeamRequestObject) (openapi.CreateTeamResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, domain.ErrBadRequest
	}

	team, err := h.teams.Create(ctx, usecase.CreateTeamRequest{
		OwnerID: userID,
		Name:    request.Body.Name,
	})
	if err != nil {
		return nil, err
	}

	return openapi.CreateTeam201JSONResponse(toAPITeam(*team)), nil
}

func (h *Handler) ListTeams(ctx context.Context, _ openapi.ListTeamsRequestObject) (openapi.ListTeamsResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	teams, err := h.teams.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]openapi.TeamWithRole, 0, len(teams))
	for _, team := range teams {
		items = append(items, toAPITeamWithRole(team))
	}

	return openapi.ListTeams200JSONResponse{Items: items}, nil
}

func (h *Handler) InviteMember(ctx context.Context, request openapi.InviteMemberRequestObject) (openapi.InviteMemberResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, domain.ErrBadRequest
	}

	member, err := h.teams.Invite(ctx, usecase.InviteMemberRequest{
		CallerID: userID,
		TeamID:   int64(request.Id),
		UserID:   request.Body.UserId,
		Role:     domain.Role(request.Body.Role),
	})
	if err != nil {
		return nil, err
	}

	return openapi.InviteMember201JSONResponse(toAPITeamMember(*member)), nil
}

func (h *Handler) GetTeamStats(ctx context.Context, request openapi.GetTeamStatsRequestObject) (openapi.GetTeamStatsResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := h.stats.Get(ctx, userID, int64(request.TeamId))
	if err != nil {
		return nil, err
	}
	return openapi.GetTeamStats200JSONResponse(toAPITeamStats(*stats)), nil
}

func (h *Handler) CreateTask(ctx context.Context, request openapi.CreateTaskRequestObject) (openapi.CreateTaskResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, domain.ErrBadRequest
	}

	task, err := h.tasks.Create(ctx, usecase.CreateTaskRequest{
		UserID:      userID,
		TeamID:      request.Body.TeamId,
		Title:       request.Body.Title,
		Description: request.Body.Description,
		Status:      toDomainStatusPtr(request.Body.Status),
	})
	if err != nil {
		return nil, err
	}

	return openapi.CreateTask201JSONResponse(toAPITask(*task)), nil
}

func (h *Handler) ListTasks(ctx context.Context, request openapi.ListTasksRequestObject) (openapi.ListTasksResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	result, err := h.tasks.List(ctx, usecase.ListTasksQuery{
		UserID:     userID,
		TeamID:     request.Params.TeamId,
		Status:     toDomainStatusPtr(request.Params.Status),
		AssigneeID: request.Params.AssigneeId,
		Limit:      intValue(request.Params.Limit),
		Offset:     intValue(request.Params.Offset),
	})
	if err != nil {
		return nil, err
	}

	items := make([]openapi.Task, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		items = append(items, toAPITask(task))
	}

	return openapi.ListTasks200JSONResponse{
		Items:  items,
		Total:  result.Total,
		Limit:  result.Limit,
		Offset: result.Offset,
	}, nil
}

func (h *Handler) UpdateTask(ctx context.Context, request openapi.UpdateTaskRequestObject) (openapi.UpdateTaskResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, domain.ErrBadRequest
	}

	task, err := h.tasks.Update(ctx, usecase.UpdateTaskRequest{
		UserID:          userID,
		TaskID:          int64(request.Id),
		Title:           request.Body.Title,
		Description:     request.Body.Description,
		Status:          toDomainStatusPtr(request.Body.Status),
		AssigneeID:      request.Body.AssigneeId,
		ExpectedVersion: &request.Body.Version,
	})
	if err != nil {
		return nil, err
	}

	return openapi.UpdateTask200JSONResponse(toAPITask(*task)), nil
}

func (h *Handler) ListTaskHistory(ctx context.Context, request openapi.ListTaskHistoryRequestObject) (openapi.ListTaskHistoryResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	entries, err := h.tasks.History(ctx, userID, int64(request.Id))
	if err != nil {
		return nil, err
	}

	items := make([]openapi.TaskHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toAPITaskHistoryEntry(entry))
	}

	return openapi.ListTaskHistory200JSONResponse{Items: items}, nil
}

func (h *Handler) AddTaskComment(ctx context.Context, request openapi.AddTaskCommentRequestObject) (openapi.AddTaskCommentResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, domain.ErrBadRequest
	}

	comment, err := h.tasks.AddComment(ctx, usecase.AddCommentRequest{
		UserID:  userID,
		TaskID:  int64(request.Id),
		Content: request.Body.Content,
	})
	if err != nil {
		return nil, err
	}

	return openapi.AddTaskComment201JSONResponse(toAPITaskComment(*comment)), nil
}

func (h *Handler) ListTaskComments(ctx context.Context, request openapi.ListTaskCommentsRequestObject) (openapi.ListTaskCommentsResponseObject, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	result, err := h.tasks.Comments(ctx, usecase.ListCommentsQuery{
		UserID: userID,
		TaskID: int64(request.Id),
		Limit:  intValue(request.Params.Limit),
		Offset: intValue(request.Params.Offset),
	})
	if err != nil {
		return nil, err
	}

	items := make([]openapi.TaskComment, 0, len(result.Comments))
	for _, comment := range result.Comments {
		items = append(items, toAPITaskComment(comment))
	}

	return openapi.ListTaskComments200JSONResponse{
		Items:  items,
		Total:  result.Total,
		Limit:  result.Limit,
		Offset: result.Offset,
	}, nil
}
