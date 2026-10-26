package usecase

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"secunda/internal/domain"
)

type RegisterCommand struct {
	Email    string
	Password string
	Name     string
}

type LoginCommand struct {
	Email    string
	Password string
}

type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	User      domain.User
}

type AuthUseCase struct {
	users     UserRepository
	tokens    TokenIssuer
	hasher    PasswordHasher
	dummyHash string
}

func NewAuthUseCase(users UserRepository, tokens TokenIssuer, hasher PasswordHasher) *AuthUseCase {
	// Хеш-заглушка выравнивает время ответа для несуществующего email,
	// чтобы нельзя было перебором определить зарегистрированных пользователей.
	dummyHash, _ := hasher.Hash("timing-equalization-dummy-password")

	return &AuthUseCase{
		users:     users,
		tokens:    tokens,
		hasher:    hasher,
		dummyHash: dummyHash,
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, cmd RegisterCommand) (*AuthResult, error) {
	email, emailErr := normalizeEmail(cmd.Email)
	name := strings.TrimSpace(cmd.Name)

	details := make(map[string]string)
	if emailErr != nil {
		details["email"] = emailErr.Error()
	}
	switch {
	case name == "":
		details["name"] = "поле обязательно"
	case utf8.RuneCountInString(name) > 100:
		details["name"] = "не более 100 символов"
	}
	if len(cmd.Password) < 8 {
		details["password"] = "не менее 8 символов"
	} else if len(cmd.Password) > 72 {
		details["password"] = "не более 72 байт"
	}
	if len(details) > 0 {
		return nil, domain.ErrValidation.WithDetails(details)
	}

	hash, err := uc.hasher.Hash(cmd.Password)
	if err != nil {
		return nil, domain.ErrInternal.WithErr(err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		CreatedAt:    time.Now().UTC(),
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return uc.issueToken(user)
}

func (uc *AuthUseCase) Login(ctx context.Context, cmd LoginCommand) (*AuthResult, error) {
	email, err := normalizeEmail(cmd.Email)
	if err != nil || cmd.Password == "" {
		return nil, domain.ErrInvalidCreds
	}

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			uc.hasher.Compare(uc.dummyHash, cmd.Password)
			return nil, domain.ErrInvalidCreds
		}
		return nil, err
	}

	if !uc.hasher.Compare(user.PasswordHash, cmd.Password) {
		return nil, domain.ErrInvalidCreds
	}

	return uc.issueToken(user)
}

func (uc *AuthUseCase) issueToken(user *domain.User) (*AuthResult, error) {
	token, expiresAt, err := uc.tokens.Issue(user.ID)
	if err != nil {
		return nil, domain.ErrInternal.WithErr(err)
	}
	return &AuthResult{Token: token, ExpiresAt: expiresAt, User: *user}, nil
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", errors.New("поле обязательно")
	}
	if len(email) > 255 {
		return "", errors.New("не более 255 символов")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", errors.New("некорректный email")
	}
	return email, nil
}
