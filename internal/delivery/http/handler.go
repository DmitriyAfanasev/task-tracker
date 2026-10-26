package httpapi

import (
	"context"

	"secunda/internal/delivery/http/openapi"
	"secunda/internal/domain"
	"secunda/internal/usecase"
)

// Handler — тонкий адаптер между сгенерированным OpenAPI-контрактом и use case.
// Здесь нет бизнес-правил: только вызов сценария и маппинг результата.
type Handler struct {
	auth  *usecase.AuthUseCase
	teams *usecase.TeamUseCase
}

var _ openapi.StrictServerInterface = (*Handler)(nil)

func NewHandler(auth *usecase.AuthUseCase, teams *usecase.TeamUseCase) *Handler {
	return &Handler{auth: auth, teams: teams}
}

func (h *Handler) Register(ctx context.Context, request openapi.RegisterRequestObject) (openapi.RegisterResponseObject, error) {
	if request.Body == nil {
		return nil, domain.ErrBadRequest
	}

	result, err := h.auth.Register(ctx, usecase.RegisterCommand{
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

	result, err := h.auth.Login(ctx, usecase.LoginCommand{
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

	team, err := h.teams.Create(ctx, usecase.CreateTeamCommand{
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

	member, err := h.teams.Invite(ctx, usecase.InviteMemberCommand{
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
