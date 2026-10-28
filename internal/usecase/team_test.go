package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"secunda/internal/domain"
)

// --- Create fixtures ---

func newCreateTeamUC(t *testing.T, role domain.Role) (*TeamUseCase, *MockTeamRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	teams := NewMockTeamRepository(ctrl)

	// Create не требует membership, но AddMember вызывается внутри транзакции.
	// membership не ставим AnyTimes — Create не проверяет роль.

	uc := NewTeamUseCase(teams, NewMockUserRepository(ctrl), immediateTx{})
	return uc, teams
}

// --- Invite fixtures ---

func newInviteUC(t *testing.T, role domain.Role) (*TeamUseCase, *MockTeamRepository, *MockUserRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	teams := NewMockTeamRepository(ctrl)
	users := NewMockUserRepository(ctrl)

	// Membership и GetByID не ставим — каждый тест настраивает свои ожидания.

	uc := NewTeamUseCase(teams, users, immediateTx{})
	return uc, teams, users
}

// --- Create tests ---

func TestCreateTeamRejectsEmptyName(t *testing.T) {
	uc, _ := newCreateTeamUC(t, domain.RoleOwner)
	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateTeamRejectsLongName(t *testing.T) {
	uc, _ := newCreateTeamUC(t, domain.RoleOwner)
	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    string(make([]rune, 101)),
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateTeamTrimsWhitespace(t *testing.T) {
	uc, teams := newCreateTeamUC(t, domain.RoleOwner)
	var capturedName string

	teams.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, t *domain.Team) error {
		capturedName = t.Name
		return nil
	}).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	team, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "  My Team  ",
	})

	assert.NoError(t, err)
	assert.Equal(t, "My Team", team.Name)
	assert.Equal(t, "My Team", capturedName)
}

func TestCreateTeamSetsCreatedBy(t *testing.T) {
	uc, teams := newCreateTeamUC(t, domain.RoleOwner)
	var capturedOwner int64

	teams.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, t *domain.Team) error {
		capturedOwner = t.CreatedBy
		return nil
	}).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	team, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 42,
		Name:    "Test Team",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(42), team.CreatedBy)
	assert.Equal(t, int64(42), capturedOwner)
}

func TestCreateTeamAddsOwnerAsMember(t *testing.T) {
	uc, teams := newCreateTeamUC(t, domain.RoleOwner)

	teams.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "Team",
	})

	assert.NoError(t, err)
}

func TestCreateTeamPropagatesCreateError(t *testing.T) {
	uc, teams := newCreateTeamUC(t, domain.RoleOwner)

	teams.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrInternal).Times(1)

	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "Team",
	})

	assert.ErrorIs(t, err, domain.ErrInternal)
}

func TestCreateTeamPropagatesAddMemberError(t *testing.T) {
	uc, teams := newCreateTeamUC(t, domain.RoleOwner)

	teams.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.ErrInternal).Times(1)

	_, err := uc.Create(context.Background(), CreateTeamRequest{
		OwnerID: 1,
		Name:    "Team",
	})

	assert.ErrorIs(t, err, domain.ErrInternal)
}

// --- Invite tests ---

func TestInviteRejectsZeroUserID(t *testing.T) {
	uc, _, _ := newInviteUC(t, domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   0,
		Role:     domain.RoleMember,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestInviteRejectsNegativeUserID(t *testing.T) {
	uc, _, _ := newInviteUC(t, domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   -1,
		Role:     domain.RoleMember,
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestInviteRejectsOwnerRole(t *testing.T) {
	uc, _, _ := newInviteUC(t, domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleOwner,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidRole)
}

func TestInviteRejectsInvalidRole(t *testing.T) {
	uc, _, _ := newInviteUC(t, domain.RoleOwner)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.Role("invalid"),
	})

	assert.ErrorIs(t, err, domain.ErrInvalidRole)
}

func TestInviteRejectsMemberAsCaller(t *testing.T) {
	uc, teams, _ := newInviteUC(t, domain.RoleMember)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleMember, nil).Times(1)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestInviteRejectsAdminInvitingAdmin(t *testing.T) {
	uc, teams, _ := newInviteUC(t, domain.RoleAdmin)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleAdmin, nil).Times(1)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleAdmin,
	})

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestInviteRejectsAdminInvitingOwner(t *testing.T) {
	uc, _, _ := newInviteUC(t, domain.RoleAdmin)
	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleOwner,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidRole)
}

func TestInviteSucceedsForOwnerInvitingAdmin(t *testing.T) {
	uc, teams, users := newInviteUC(t, domain.RoleOwner)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleOwner, nil).Times(1)
	users.EXPECT().GetByID(gomock.Any(), int64(2)).Return(&domain.User{ID: 2}, nil).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), int64(1), int64(2), domain.RoleAdmin).Return(nil).Times(1)

	member, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleAdmin,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(1), member.TeamID)
	assert.Equal(t, int64(2), member.UserID)
	assert.Equal(t, domain.RoleAdmin, member.Role)
}

func TestInviteSucceedsForOwnerInvitingMember(t *testing.T) {
	uc, teams, users := newInviteUC(t, domain.RoleOwner)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleOwner, nil).Times(1)
	users.EXPECT().GetByID(gomock.Any(), int64(2)).Return(&domain.User{ID: 2}, nil).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), int64(1), int64(2), domain.RoleMember).Return(nil).Times(1)

	member, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})

	assert.NoError(t, err)
	assert.Equal(t, domain.RoleMember, member.Role)
}

func TestInviteSucceedsForAdminInvitingMember(t *testing.T) {
	uc, teams, users := newInviteUC(t, domain.RoleAdmin)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleAdmin, nil).Times(1)
	users.EXPECT().GetByID(gomock.Any(), int64(2)).Return(&domain.User{ID: 2}, nil).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), int64(1), int64(2), domain.RoleMember).Return(nil).Times(1)

	member, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})

	assert.NoError(t, err)
	assert.Equal(t, domain.RoleMember, member.Role)
}

func TestInvitePropagatesMembershipError(t *testing.T) {
	uc, teams, _ := newInviteUC(t, domain.RoleOwner)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleOwner, domain.ErrInternal).Times(1)

	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})

	assert.ErrorIs(t, err, domain.ErrInternal)
}

func TestInvitePropagatesUserNotFoundError(t *testing.T) {
	uc, teams, users := newInviteUC(t, domain.RoleOwner)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleOwner, nil).Times(1)
	users.EXPECT().GetByID(gomock.Any(), int64(999)).Return(nil, domain.ErrUserNotFound).Times(1)

	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   999,
		Role:     domain.RoleMember,
	})

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestInvitePropagatesAddMemberError(t *testing.T) {
	uc, teams, users := newInviteUC(t, domain.RoleOwner)
	teams.EXPECT().Membership(gomock.Any(), int64(1), gomock.Any()).Return(domain.RoleOwner, nil).Times(1)
	users.EXPECT().GetByID(gomock.Any(), int64(2)).Return(&domain.User{ID: 2}, nil).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), int64(1), int64(2), domain.RoleMember).Return(domain.ErrAlreadyMember).Times(1)

	_, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   1,
		UserID:   2,
		Role:     domain.RoleMember,
	})

	assert.ErrorIs(t, err, domain.ErrAlreadyMember)
}

func TestInviteReturnsTeamMember(t *testing.T) {
	uc, teams, users := newInviteUC(t, domain.RoleOwner)
	teams.EXPECT().Membership(gomock.Any(), int64(42), gomock.Any()).Return(domain.RoleOwner, nil).Times(1)
	users.EXPECT().GetByID(gomock.Any(), int64(99)).Return(&domain.User{ID: 99}, nil).Times(1)
	teams.EXPECT().AddMember(gomock.Any(), int64(42), int64(99), domain.RoleMember).Return(nil).Times(1)

	member, err := uc.Invite(context.Background(), InviteMemberRequest{
		CallerID: 1,
		TeamID:   42,
		UserID:   99,
		Role:     domain.RoleMember,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(42), member.TeamID)
	assert.Equal(t, int64(99), member.UserID)
	assert.Equal(t, domain.RoleMember, member.Role)
}

func TestListMembersRequiresMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	teams := NewMockTeamRepository(ctrl)
	users := NewMockUserRepository(ctrl)
	teams.EXPECT().Membership(gomock.Any(), int64(10), gomock.Any()).Return(domain.RoleMember, domain.ErrTeamNotFound)

	uc := NewTeamUseCase(teams, users, immediateTx{})
	_, err := uc.ListMembers(context.Background(), 1, 10)
	if !errors.Is(err, domain.ErrTeamNotFound) {
		t.Fatalf("error = %v, want ErrTeamNotFound", err)
	}
}

func TestListMembersReturnsMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	teams := NewMockTeamRepository(ctrl)
	users := NewMockUserRepository(ctrl)
	teams.EXPECT().Membership(gomock.Any(), int64(10), gomock.Any()).Return(domain.RoleMember, nil)
	teams.EXPECT().ListMembers(gomock.Any(), int64(10)).Return([]domain.TeamMemberInfo{
		{UserID: 1, Name: "Иван", Email: "ivan@example.com", Role: domain.RoleOwner},
		{UserID: 2, Name: "Пётр", Email: "petr@example.com", Role: domain.RoleMember},
	}, nil)

	uc := NewTeamUseCase(teams, users, immediateTx{})
	got, err := uc.ListMembers(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("ListMembers() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "Иван" || got[1].UserID != 2 {
		t.Fatalf("unexpected members: %+v", got)
	}
}
