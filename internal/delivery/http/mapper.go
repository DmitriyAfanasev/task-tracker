package httpapi

import (
	openapi_types "github.com/oapi-codegen/runtime/types"

	"secunda/internal/delivery/http/openapi"
	"secunda/internal/domain"
	"secunda/internal/usecase"
)

func toAPIUser(user domain.User) openapi.User {
	return openapi.User{
		Id:        user.ID,
		Email:     openapi_types.Email(user.Email),
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
	}
}

func toAPITeam(team domain.Team) openapi.Team {
	return openapi.Team{
		Id:        team.ID,
		Name:      team.Name,
		CreatedBy: team.CreatedBy,
		CreatedAt: team.CreatedAt,
	}
}

func toAPITeamWithRole(team domain.TeamWithRole) openapi.TeamWithRole {
	return openapi.TeamWithRole{
		Id:        team.ID,
		Name:      team.Name,
		CreatedBy: team.CreatedBy,
		CreatedAt: team.CreatedAt,
		Role:      openapi.Role(team.Role),
	}
}

func toAPITeamMember(member usecase.TeamMember) openapi.TeamMember {
	return openapi.TeamMember{
		TeamId: member.TeamID,
		UserId: member.UserID,
		Role:   openapi.Role(member.Role),
	}
}
