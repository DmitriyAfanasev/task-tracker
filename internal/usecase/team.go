package usecase

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"secunda/internal/domain"
)

type CreateTeamRequest struct {
	OwnerID int64
	Name    string
}

type InviteMemberRequest struct {
	CallerID int64
	TeamID   int64
	UserID   int64
	Role     domain.Role
}

type TeamMember struct {
	TeamID int64
	UserID int64
	Role   domain.Role
}

type TeamUseCase struct {
	teams TeamRepository
	users UserRepository
	tx    TransactionManager
}

func NewTeamUseCase(teams TeamRepository, users UserRepository, tx TransactionManager) *TeamUseCase {
	return &TeamUseCase{teams: teams, users: users, tx: tx}
}

func (uc *TeamUseCase) Create(ctx context.Context, cmd CreateTeamRequest) (*domain.Team, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, domain.ErrValidation.WithDetails(map[string]string{"name": "поле обязательно"})
	}
	if utf8.RuneCountInString(name) > 100 {
		return nil, domain.ErrValidation.WithDetails(map[string]string{"name": "не более 100 символов"})
	}

	team := &domain.Team{
		Name:      name,
		CreatedBy: cmd.OwnerID,
		CreatedAt: time.Now().UTC(),
	}

	// Создание команды и добавление создателя как owner — атомарно.
	err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.teams.Create(txCtx, team); err != nil {
			return err
		}
		return uc.teams.AddMember(txCtx, team.ID, cmd.OwnerID, domain.RoleOwner)
	})
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (uc *TeamUseCase) List(ctx context.Context, userID int64) ([]domain.TeamWithRole, error) {
	return uc.teams.ListByUser(ctx, userID)
}

func (uc *TeamUseCase) Invite(ctx context.Context, cmd InviteMemberRequest) (*TeamMember, error) {
	if cmd.UserID <= 0 {
		return nil, domain.ErrValidation.WithDetails(map[string]string{"user_id": "должен быть больше 0"})
	}
	if !cmd.Role.Valid() || cmd.Role == domain.RoleOwner {
		return nil, domain.ErrInvalidRole
	}

	callerRole, err := uc.teams.Membership(ctx, cmd.TeamID, cmd.CallerID)
	if err != nil {
		return nil, err
	}
	if !callerRole.CanManageTeam() || !callerRole.CanInvite(cmd.Role) {
		return nil, domain.ErrForbidden
	}

	if _, err := uc.users.GetByID(ctx, cmd.UserID); err != nil {
		return nil, err
	}

	if err := uc.teams.AddMember(ctx, cmd.TeamID, cmd.UserID, cmd.Role); err != nil {
		return nil, err
	}

	return &TeamMember{TeamID: cmd.TeamID, UserID: cmd.UserID, Role: cmd.Role}, nil
}
