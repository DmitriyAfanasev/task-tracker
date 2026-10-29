package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"secunda/internal/domain"
)

// --- Register fixtures ---

func newRegisterUC(t *testing.T) (*AuthUseCase, *MockUserRepository, *MockTokenIssuer, *MockPasswordHasher) {
	t.Helper()

	ctrl := gomock.NewController(t)
	users := NewMockUserRepository(ctrl)
	tokens := NewMockTokenIssuer(ctrl)
	hasher := NewMockPasswordHasher(ctrl)

	hasher.EXPECT().Hash(gomock.Any()).Return("$2a$12$fakehash", nil).AnyTimes()
	tokens.EXPECT().Issue(gomock.Any()).Return("fake-jwt", time.Now().Add(24*time.Hour), nil).AnyTimes()

	uc := NewAuthUseCase(users, tokens, hasher)
	return uc, users, tokens, hasher
}

// --- Login fixtures ---

func newLoginUC(t *testing.T, user *domain.User) (*AuthUseCase, *MockUserRepository, *MockTokenIssuer, *MockPasswordHasher) {
	t.Helper()

	ctrl := gomock.NewController(t)
	users := NewMockUserRepository(ctrl)
	tokens := NewMockTokenIssuer(ctrl)
	hasher := NewMockPasswordHasher(ctrl)

	if user != nil {
		users.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).Return(user, nil).AnyTimes()
	} else {
		users.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).Return(nil, domain.ErrUserNotFound).AnyTimes()
	}

	hasher.EXPECT().Hash(gomock.Any()).Return("$2a$12$dummy", nil).AnyTimes()
	tokens.EXPECT().Issue(gomock.Any()).Return("fake-jwt", time.Now().Add(24*time.Hour), nil).AnyTimes()

	uc := NewAuthUseCase(users, tokens, hasher)
	return uc, users, tokens, hasher
}

// --- Register tests ---

func TestRegisterRejectsEmptyEmail(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Register(context.Background(), RegisterRequest{
		Password: "password123",
		Name:     "Иван",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestRegisterRejectsInvalidEmail(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "not-an-email",
		Password: "password123",
		Name:     "Иван",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestRegisterRejectsTooLongEmail(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    string(make([]byte, 256)) + "@a.co",
		Password: "password123",
		Name:     "Иван",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestRegisterRejectsEmptyName(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "ivan@example.com",
		Password: "password123",
		Name:     "",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestRegisterRejectsLongName(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "ivan@example.com",
		Password: "password123",
		Name:     string(make([]rune, 101)),
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "ivan@example.com",
		Password: "short",
		Name:     "Иван",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestRegisterRejectsLongPassword(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "ivan@example.com",
		Password: string(make([]byte, 73)),
		Name:     "Иван",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestRegisterRejectsMultipleValidationErrors(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "bad",
		Password: "123",
		Name:     "",
	})

	assert.ErrorIs(t, err, domain.ErrValidation)
	details, ok := err.(*domain.Error).Details["email"]
	assert.True(t, ok)
	assert.NotEmpty(t, details)
}

func TestRegisterNormalizesEmail(t *testing.T) {
	uc, users, _, _ := newRegisterUC(t)
	var capturedUser *domain.User

	users.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
		capturedUser = u
		return nil
	}).Times(1)

	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "  Ivan@Example.COM  ",
		Password: "password123",
		Name:     "Иван",
	})

	assert.NoError(t, err)
	assert.Equal(t, "ivan@example.com", capturedUser.Email)
}

func TestRegisterPersistsUserAndIssuesToken(t *testing.T) {
	uc, users, _, _ := newRegisterUC(t)

	users.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
		return nil
	}).Times(1)

	result, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "ivan@example.com",
		Password: "password123",
		Name:     "Иван",
	})

	assert.NoError(t, err)
	assert.Equal(t, "fake-jwt", result.Token)
	assert.Equal(t, "ivan@example.com", result.User.Email)
	assert.Equal(t, "Иван", result.User.Name)
	assert.Equal(t, "$2a$12$fakehash", result.User.PasswordHash)
}

func TestRegisterPropagatesUserCreateError(t *testing.T) {
	uc, users, _, _ := newRegisterUC(t)
	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrEmailTaken).Times(1)

	_, err := uc.Register(context.Background(), RegisterRequest{
		Email:    "ivan@example.com",
		Password: "password123",
		Name:     "Иван",
	})

	assert.ErrorIs(t, err, domain.ErrEmailTaken)
}

// --- Login tests ---

func TestLoginRejectsEmptyEmail(t *testing.T) {
	uc, _, _, _ := newLoginUC(t, nil)
	_, err := uc.Login(context.Background(), LoginRequest{
		Email:    "",
		Password: "password123",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCreds)
}

func TestLoginRejectsInvalidEmail(t *testing.T) {
	uc, _, _, _ := newLoginUC(t, nil)
	_, err := uc.Login(context.Background(), LoginRequest{
		Email:    "not-an-email",
		Password: "password123",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCreds)
}

func TestLoginRejectsEmptyPassword(t *testing.T) {
	uc, _, _, _ := newLoginUC(t, nil)
	_, err := uc.Login(context.Background(), LoginRequest{
		Email:    "ivan@example.com",
		Password: "",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCreds)
}

func TestLoginReturnsInvalidCredsForUnknownUser(t *testing.T) {
	uc, _, _, _ := newLoginUC(t, nil)

	_, err := uc.Login(context.Background(), LoginRequest{
		Email:    "unknown@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCreds)
}

func TestLoginReturnsInvalidCredsForWrongPassword(t *testing.T) {
	uc, _, _, hasher := newLoginUC(t, &domain.User{ID: 1, Email: "ivan@example.com", PasswordHash: "$2a$12$valid"})
	hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(false).Times(1)

	_, err := uc.Login(context.Background(), LoginRequest{
		Email:    "ivan@example.com",
		Password: "wrong",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCreds)
}

func TestLoginSucceedsWithCorrectCredentials(t *testing.T) {
	uc, _, _, hasher := newLoginUC(t, &domain.User{ID: 1, Email: "ivan@example.com", PasswordHash: "$2a$12$valid"})
	hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true).Times(1)

	result, err := uc.Login(context.Background(), LoginRequest{
		Email:    "ivan@example.com",
		Password: "correct",
	})

	assert.NoError(t, err)
	assert.Equal(t, "fake-jwt", result.Token)
	assert.Equal(t, int64(1), result.User.ID)
}

func TestLoginNormalizesEmail(t *testing.T) {
	uc, _, _, hasher := newLoginUC(t, &domain.User{ID: 1, Email: "ivan@example.com", PasswordHash: "$2a$12$valid"})
	hasher.EXPECT().Compare(gomock.Any(), gomock.Any()).Return(true).Times(1)

	result, err := uc.Login(context.Background(), LoginRequest{
		Email:    "  Ivan@EXAMPLE.COM  ",
		Password: "correct",
	})

	assert.NoError(t, err)
	assert.Equal(t, "ivan@example.com", result.User.Email)
}

// --- normalizeEmail tests ---

func TestNormalizeEmailReturnsErrorForEmpty(t *testing.T) {
	_, err := normalizeEmail("")
	assert.Error(t, err)
}

func TestNormalizeEmailReturnsErrorForInvalid(t *testing.T) {
	_, err := normalizeEmail("not-an-email")
	assert.Error(t, err)
}

func TestNormalizeEmailReturnsErrorForMissingName(t *testing.T) {
	_, err := normalizeEmail("no-at-sign")
	assert.Error(t, err)
}

func TestNormalizeEmailTrimSpacesAndLowercase(t *testing.T) {
	email, err := normalizeEmail("  USER@Example.COM  ")

	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", email)
}

func TestNormalizeEmailRejectsLongEmail(t *testing.T) {
	_, err := normalizeEmail(string(make([]byte, 256)) + "@a.co")
	assert.Error(t, err)
}

// --- Get tests ---

func newGetUC(t *testing.T, user *domain.User) (*AuthUseCase, *MockUserRepository, *MockTokenIssuer, *MockPasswordHasher) {
	t.Helper()

	ctrl := gomock.NewController(t)
	users := NewMockUserRepository(ctrl)
	tokens := NewMockTokenIssuer(ctrl)
	hasher := NewMockPasswordHasher(ctrl)

	if user != nil {
		users.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(user, nil).AnyTimes()
	} else {
		users.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrUserNotFound).AnyTimes()
	}

	uc := NewAuthUseCase(users, tokens, hasher)
	return uc, users, tokens, hasher
}

func TestGetReturnsUser(t *testing.T) {
	uc, _, _, _ := newGetUC(t, &domain.User{ID: 42, Email: "alice@example.com", Name: "Алиса"})

	user, err := uc.Get(context.Background(), 42)

	assert.NoError(t, err)
	assert.Equal(t, int64(42), user.ID)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "Алиса", user.Name)
}

func TestGetPropagatesUserNotFound(t *testing.T) {
	uc, _, _, _ := newGetUC(t, nil)

	_, err := uc.Get(context.Background(), 99)

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}
