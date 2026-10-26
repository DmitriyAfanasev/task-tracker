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

func toAPITask(task domain.Task) openapi.Task {
	return openapi.Task{
		Id:          task.ID,
		TeamId:      task.TeamID,
		Title:       task.Title,
		Description: task.Description,
		Status:      openapi.TaskStatus(task.Status),
		CreatedBy:   task.CreatedBy,
		AssigneeId:  task.AssigneeID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		ClosedAt:    task.ClosedAt,
		Version:     task.Version,
	}
}

func toAPITaskHistoryEntry(entry domain.TaskHistoryEntry) openapi.TaskHistoryEntry {
	changes := make([]openapi.TaskChange, 0, len(entry.Changes))
	for _, change := range entry.Changes {
		changes = append(changes, openapi.TaskChange{
			Field: change.Field,
			Old:   change.Old,
			New:   change.New,
		})
	}
	return openapi.TaskHistoryEntry{
		Id:        entry.ID,
		TaskId:    entry.TaskID,
		ChangedBy: entry.ChangedBy,
		Changes:   changes,
		CreatedAt: entry.CreatedAt,
	}
}

func toAPITaskComment(comment domain.TaskComment) openapi.TaskComment {
	return openapi.TaskComment{
		Id:        comment.ID,
		TaskId:    comment.TaskID,
		UserId:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}
}

func toDomainStatusPtr(status *openapi.TaskStatus) *domain.TaskStatus {
	if status == nil {
		return nil
	}
	value := domain.TaskStatus(*status)
	return &value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
