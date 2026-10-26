package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"secunda/internal/domain"
)

type fakeUserRepo struct {
	user      *domain.User
	byErr     error
	createErr error
}

func (f *fakeUserRepo) Create(_ context.Context, user *domain.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.user == nil {
		f.user = &domain.User{}
	}
	*f.user = *user
	return nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, _ int64) (*domain.User, error) {
	if f.byErr != nil {
		return nil, f.byErr
	}
	cp := *f.user
	return &cp, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	if f.byErr != nil {
		return nil, f.byErr
	}
	cp := *f.user
	return &cp, nil
}

type fakeTokenIssuer struct {
	token string
	expiresAt time.Time
	issueErr error
}

func (f *fakeTokenIssuer) Issue(_ int64) (string, time.Time, error) {
	if f.issueErr != nil {
		return "", time.Time{}, f.issueErr
	}
	return f.token, f.expiresAt, nil
}

type fakeHasher struct {
	hash    string
	compare bool
}

func (f *fakeHasher) Hash(_ string) (string, error) { return f.hash, nil }
func (f *fakeHasher) Compare(_, _ string) bool      { return f.compare }

func newFakeAuthUC() (*AuthUseCase, *fakeUserRepo, *fakeTokenIssuer, *fakeHasher) {
	users := &fakeUserRepo{}
	tokens := &fakeTokenIssuer{token: "fake-jwt", expiresAt: time.Now().Add(24 * time.Hour)}
	hasher := &fakeHasher{hash: "$2a$12$fakehash"}
	uc := NewAuthUseCase(users, tokens, hasher)
	return uc, users, tokens, hasher
}

// --- Register ---

func TestRegisterRejectsEmptyEmail(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Register(context.Background(), RegisterCommand{
		Password: "password123",
		Name:     "Иван",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestRegisterRejectsInvalidEmail(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "not-an-email",
		Password: "password123",
		Name:     "Иван",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestRegisterRejectsTooLongEmail(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    string(make([]byte, 256)) + "@a.co",
		Password: "password123",
		Name:     "Иван",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestRegisterRejectsEmptyName(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "ivan@example.com",
		Password: "password123",
		Name:     "",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestRegisterRejectsLongName(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "ivan@example.com",
		Password: "password123",
		Name:     string(make([]rune, 101)),
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "ivan@example.com",
		Password: "short",
		Name:     "Иван",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestRegisterRejectsLongPassword(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "ivan@example.com",
		Password: string(make([]byte, 73)),
		Name:     "Иван",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestRegisterRejectsMultipleValidationErrors(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "bad",
		Password: "123",
		Name:     "",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
	details, ok := err.(*domain.Error).Details["email"]
	if !ok || details == "" {
		t.Fatalf("expected email detail in validation error")
	}
}

func TestRegisterNormalizesEmail(t *testing.T) {
	uc, users, tokens, _ := newFakeAuthUC()

	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "  Ivan@Example.COM  ",
		Password: "password123",
		Name:     "Иван",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if users.user.Email != "ivan@example.com" {
		t.Fatalf("email = %q, want ivan@example.com", users.user.Email)
	}
	if tokens.token == "" {
		t.Fatal("token should be issued on success")
	}
}

func TestRegisterPersistsUserAndIssuesToken(t *testing.T) {
	uc, users, _, _ := newFakeAuthUC()
	users.createErr = nil

	result, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "ivan@example.com",
		Password: "password123",
		Name:     "Иван",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.Token != "fake-jwt" {
		t.Fatalf("token = %q, want fake-jwt", result.Token)
	}
	if result.User.Email != "ivan@example.com" {
		t.Fatalf("user email = %q, want ivan@example.com", result.User.Email)
	}
	if result.User.Name != "Иван" {
		t.Fatalf("user name = %q, want Иван", result.User.Name)
	}
	if result.User.PasswordHash != "$2a$12$fakehash" {
		t.Fatal("password hash should be stored")
	}
}

func TestRegisterPropagatesUserCreateError(t *testing.T) {
	uc, users, _, _ := newFakeAuthUC()
	users.createErr = domain.ErrEmailTaken

	_, err := uc.Register(context.Background(), RegisterCommand{
		Email:    "ivan@example.com",
		Password: "password123",
		Name:     "Иван",
	})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}

// --- Login ---

func TestLoginRejectsEmptyEmail(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Login(context.Background(), LoginCommand{
		Email:    "",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrInvalidCreds) {
		t.Fatalf("error = %v, want ErrInvalidCreds", err)
	}
}

func TestLoginRejectsInvalidEmail(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Login(context.Background(), LoginCommand{
		Email:    "not-an-email",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrInvalidCreds) {
		t.Fatalf("error = %v, want ErrInvalidCreds", err)
	}
}

func TestLoginRejectsEmptyPassword(t *testing.T) {
	uc, _, _, _ := newFakeAuthUC()
	_, err := uc.Login(context.Background(), LoginCommand{
		Email:    "ivan@example.com",
		Password: "",
	})
	if !errors.Is(err, domain.ErrInvalidCreds) {
		t.Fatalf("error = %v, want ErrInvalidCreds", err)
	}
}

func TestLoginReturnsInvalidCredsForUnknownUser(t *testing.T) {
	uc, users, _, _ := newFakeAuthUC()
	users.byErr = domain.ErrUserNotFound

	_, err := uc.Login(context.Background(), LoginCommand{
		Email:    "unknown@example.com",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrInvalidCreds) {
		t.Fatalf("error = %v, want ErrInvalidCreds", err)
	}
}

func TestLoginReturnsInvalidCredsForWrongPassword(t *testing.T) {
	uc, users, _, hasher := newFakeAuthUC()
	users.user = &domain.User{ID: 1, Email: "ivan@example.com", PasswordHash: "$2a$12$valid"}
	hasher.compare = false

	_, err := uc.Login(context.Background(), LoginCommand{
		Email:    "ivan@example.com",
		Password: "wrong",
	})
	if !errors.Is(err, domain.ErrInvalidCreds) {
		t.Fatalf("error = %v, want ErrInvalidCreds", err)
	}
}

func TestLoginSucceedsWithCorrectCredentials(t *testing.T) {
	uc, users, _, hasher := newFakeAuthUC()
	users.user = &domain.User{ID: 1, Email: "ivan@example.com", PasswordHash: "$2a$12$valid"}
	hasher.compare = true

	result, err := uc.Login(context.Background(), LoginCommand{
		Email:    "ivan@example.com",
		Password: "correct",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Token != "fake-jwt" {
		t.Fatalf("token = %q, want fake-jwt", result.Token)
	}
	if result.User.ID != 1 {
		t.Fatalf("user ID = %d, want 1", result.User.ID)
	}
}

func TestLoginNormalizesEmail(t *testing.T) {
	uc, users, _, hasher := newFakeAuthUC()
	users.user = &domain.User{ID: 1, Email: "ivan@example.com", PasswordHash: "$2a$12$valid"}
	hasher.compare = true

	result, err := uc.Login(context.Background(), LoginCommand{
		Email:    "  Ivan@EXAMPLE.COM  ",
		Password: "correct",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.Email != "ivan@example.com" {
		t.Fatalf("email = %q, want ivan@example.com", result.User.Email)
	}
}

// --- normalizeEmail ---

func TestNormalizeEmailReturnsErrorForEmpty(t *testing.T) {
	_, err := normalizeEmail("")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestNormalizeEmailReturnsErrorForInvalid(t *testing.T) {
	_, err := normalizeEmail("not-an-email")
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestNormalizeEmailReturnsErrorForMissingName(t *testing.T) {
	_, err := normalizeEmail("no-at-sign")
	if err == nil {
		t.Fatal("expected error for email without @")
	}
}

func TestNormalizeEmailTrimSpacesAndLowercase(t *testing.T) {
	email, err := normalizeEmail("  USER@Example.COM  ")
	if err != nil {
		t.Fatalf("normalizeEmail() error = %v", err)
	}
	if email != "user@example.com" {
		t.Fatalf("email = %q, want user@example.com", email)
	}
}

func TestNormalizeEmailRejectsLongEmail(t *testing.T) {
	_, err := normalizeEmail(string(make([]byte, 256)) + "@a.co")
	if err == nil {
		t.Fatal("expected error for too long email")
	}
}
