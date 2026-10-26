package domain

import (
	"fmt"
	"net/http"
)

// Error — доменная ошибка с кодом для API и HTTP-статусом.
type Error struct {
	Code       string
	Message    string
	HTTPStatus int
	Details    map[string]string
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return e.Code
}

func (e *Error) Unwrap() error { return e.Err }

// WithDetails возвращает копию ошибки с деталями валидации.
func (e *Error) WithDetails(details map[string]string) *Error {
	cp := *e
	cp.Details = details
	return &cp
}

// WithErr возвращает копию ошибки с исходной причиной (для логов).
func (e *Error) WithErr(err error) *Error {
	cp := *e
	cp.Err = err
	return &cp
}

func newError(status int, code, message string) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: status}
}

var (
	ErrBadRequest    = newError(http.StatusBadRequest, "bad_request", "некорректный запрос")
	ErrValidation    = newError(http.StatusBadRequest, "validation_failed", "ошибка валидации")
	ErrUnauthorized  = newError(http.StatusUnauthorized, "unauthorized", "требуется аутентификация")
	ErrInvalidToken  = newError(http.StatusUnauthorized, "invalid_token", "недействительный токен")
	ErrInvalidCreds  = newError(http.StatusUnauthorized, "invalid_credentials", "неверный email или пароль")
	ErrForbidden     = newError(http.StatusForbidden, "forbidden", "недостаточно прав")
	ErrNotFound      = newError(http.StatusNotFound, "not_found", "ресурс не найден")
	ErrUserNotFound  = newError(http.StatusNotFound, "user_not_found", "пользователь не найден")
	ErrTeamNotFound  = newError(http.StatusNotFound, "team_not_found", "команда не найдена")
	ErrEmailTaken    = newError(http.StatusConflict, "email_taken", "пользователь с таким email уже существует")
	ErrAlreadyMember = newError(http.StatusConflict, "already_member", "пользователь уже состоит в команде")
	ErrInvalidRole   = newError(http.StatusBadRequest, "invalid_role", "роль должна быть admin или member")
	ErrInternal      = newError(http.StatusInternalServerError, "internal_error", "внутренняя ошибка сервера")
)
