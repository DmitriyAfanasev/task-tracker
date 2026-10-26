package usecase

import (
	"context"

	"secunda/internal/domain"
)

// TeamStatsUseCase выдаёт аналитику только владельцу или администратору команды.
type TeamStatsUseCase struct {
	teams TeamRepository
	stats TeamStatsRepository
}

func NewTeamStatsUseCase(teams TeamRepository, stats TeamStatsRepository) *TeamStatsUseCase {
	return &TeamStatsUseCase{teams: teams, stats: stats}
}

func (uc *TeamStatsUseCase) Get(ctx context.Context, userID, teamID int64) (*domain.TeamStats, error) {
	if teamID <= 0 {
		return nil, domain.ErrValidation.WithDetails(map[string]string{"team_id": "должен быть больше 0"})
	}

	role, err := uc.teams.Membership(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !role.CanManageTeam() {
		return nil, domain.ErrForbidden
	}

	return uc.stats.Stats(ctx, teamID)
}
