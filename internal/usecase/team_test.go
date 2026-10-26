package usecase

import (
	"context"
	"errors"
	"testing"

	"secunda/internal/domain"
)

type fakeTeamRepo struct {
	team          *domain.Team
	membership    domain.Role
	membershipErr error
	createErr     error
	addMemberErr  error
	inviteCall    int
}

func (f *fakeTeamRepo) Create(_ context.Context, team *domain.Team) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.team != nil {
		*f.team = *team
	}
	return nil
}

func (f *fakeTeamRepo) ListByUser(_ context.Context, _ int64) ([]domain.TeamWithRole, error) {
	return nil, nil
}

func (f *fakeTeamRepo) Membership(_ context.Context, _, _ int64) (domain.Role, error) {
	return f.membership, f.membershipErr
}

func (f *fakeTeamRepo) AddMember(_ context.Context, _, _ int64, _ domain.Role) error {
	f.inviteCall++
	return f.addMemberErr
}

type fakeUserRepoForInvite struct {
	exists   bool
	notFound bool
}

func (f *fakeUserRepoForInvite) Create(_ context.Context, _ *domain.User) error {
	return nil
}

func (f *fakeUserRepoForInvite) GetByID(_ context.Context, _ int64) (*domain.User, error) {
	if f.notFound {
		return nil, domain.ErrUserNotFound
	}
	if !f.exists {
		return nil, domain.ErrUserNotFound
	}
	return &domain.User{ID: 1}, nil
}

func (f *fakeUserRepoForInvite) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func newFakeTeamUC(role domain.Role) (*TeamUseCase, *fakeTeamRepo, *fakeUserRepoForInvite) {
	teams := &fakeTeamRepo{membership: role}
	users := &fakeUserRepoForInvite{exists: true}
	uc := NewTeamUseCase(teams, users, immediateTx{})
	return uc, teams, users
}

// --- Create ---

func TestCreateTeamRejectsEmptyName(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestCreateTeamRejectsLongName(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    string(make([]rune, 101)),
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestCreateTeamTrimsWhitespace(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	team, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "  My Team  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if team.Name != "My Team" {
		t.Fatalf("name = %q, want My Team", team.Name)
	}
}

func TestCreateTeamSetsCreatedBy(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	team, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 42,
		Name:    "Test Team",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if team.CreatedBy != 42 {
		t.Fatalf("created_by = %d, want 42", team.CreatedBy)
	}
}

func TestCreateTeamAddsOwnerAsMember(t *testing.T) {
	uc, teams, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "Team",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if teams.inviteCall != 1 {
		t.Fatalf("AddMember calls = %d, want 1", teams.inviteCall)
	}
}

func TestCreateTeamPropagatesCreateError(t *testing.T) {
	uc, teams, _ := newFakeTeamUC(domain.RoleOwner)
	teams.createErr = domain.ErrInternal

	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "Team",
	})
	if !errors.Is(err, domain.ErrInternal) {
		t.Fatalf("error = %v, want ErrInternal", err)
	}
}

func TestCreateTeamPropagatesAddMemberError(t *testing.T) {
	uc, teams, _ := newFakeTeamUC(domain.RoleOwner)
	teams.addMemberErr = domain.ErrInternal

	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "Team",
	})
	if !errors.Is(err, domain.ErrInternal) {
		t.Fatalf("error = %v, want ErrInternal", err)
	}
}

// --- Invite ---

func TestInviteRejectsZeroUserID(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   0,
		Role:     domain.RoleMember,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestInviteRejectsNegativeUserID(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   -1,
		Role:     domain.RoleMember,
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestInviteRejectsOwnerRole(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleOwner,
	})
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("error = %v, want ErrInvalidRole", err)
	}
}

func TestInviteRejectsInvalidRole(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.Role("invalid"),
	})
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("error = %v, want ErrInvalidRole", err)
	}
}

func TestInviteRejectsMemberAsCaller(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleMember)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestInviteRejectsAdminInvitingAdmin(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleAdmin)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleAdmin,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestInviteRejectsAdminInvitingOwner(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleAdmin)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleOwner,
	})
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("error = %v, want ErrInvalidRole", err)
	}
}

func TestInviteSucceedsForOwnerInvitingAdmin(t *testing.T) {
	uc, teams, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if teams.inviteCall != 1 {
		t.Fatalf("AddMember calls = %d, want 1", teams.inviteCall)
	}
}

func TestInviteSucceedsForOwnerInvitingMember(t *testing.T) {
	uc, teams, _ := newFakeTeamUC(domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})
	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if teams.inviteCall != 1 {
		t.Fatalf("AddMember calls = %d, want 1", teams.inviteCall)
	}
}

func TestInviteSucceedsForAdminInvitingMember(t *testing.T) {
	uc, teams, _ := newFakeTeamUC(domain.RoleAdmin)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})
	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if teams.inviteCall != 1 {
		t.Fatalf("AddMember calls = %d, want 1", teams.inviteCall)
	}
}

func TestInvitePropagatesMembershipError(t *testing.T) {
	uc, teams, _ := newFakeTeamUC(domain.RoleOwner)
	teams.membershipErr = domain.ErrInternal

	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})
	if !errors.Is(err, domain.ErrInternal) {
		t.Fatalf("error = %v, want ErrInternal", err)
	}
}

func TestInvitePropagatesUserNotFoundError(t *testing.T) {
	uc, _, users := newFakeTeamUC(domain.RoleOwner)
	users.exists = false

	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   999,
		Role:     domain.RoleMember,
	})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("error = %v, want ErrUserNotFound", err)
	}
}

func TestInvitePropagatesAddMemberError(t *testing.T) {
	uc, teams, _ := newFakeTeamUC(domain.RoleOwner)
	teams.addMemberErr = domain.ErrAlreadyMember

	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})
	if !errors.Is(err, domain.ErrAlreadyMember) {
		t.Fatalf("error = %v, want ErrAlreadyMember", err)
	}
}

func TestInviteReturnsTeamMember(t *testing.T) {
	uc, _, _ := newFakeTeamUC(domain.RoleOwner)
	member, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   42,
		UserID:   99,
		Role:     domain.RoleMember,
	})
	if err != nil {
		t.Fatalf("Invite() error = %v", err)
	}
	if member.TeamID != 42 {
		t.Fatalf("team_id = %d, want 42", member.TeamID)
	}
	if member.UserID != 99 {
		t.Fatalf("user_id = %d, want 99", member.UserID)
	}
	if member.Role != domain.RoleMember {
		t.Fatalf("role = %v, want member", member.Role)
	}
}
