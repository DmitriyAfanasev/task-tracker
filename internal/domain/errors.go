package domain

import "fmt"

// Kind — категория доменной ошибки. Транспорт сам решает,
// какой HTTP-статус соответствует категории: домен не знает про HTTP.
type Kind int

const (
	KindBadRequest Kind = iota + 1
	KindPayloadTooLarge
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindInternal
	KindRateLimit
)

// Error — доменная ошибка с машинным кодом и сообщением для пользователя.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Details map[string]string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return e.Code
}

func (e *Error) Unwrap() error { return e.Err }

// Is позволяет сравнивать копии ошибок (WithErr/WithDetails) с сентинелами
// через errors.Is: копия сохраняет Kind и Code исходной ошибки.
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Kind == other.Kind && e.Code == other.Code
}

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

func newError(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

var (
	ErrBadRequest      = newError(KindBadRequest, "bad_request", "некорректный запрос")
	ErrPayloadTooLarge = newError(KindPayloadTooLarge, "payload_too_large", "тело запроса слишком большое")
	ErrValidation      = newError(KindBadRequest, "validation_failed", "ошибка валидации")
	ErrUnauthorized    = newError(KindUnauthorized, "unauthorized", "требуется аутентификация")
	ErrInvalidToken    = newError(KindUnauthorized, "invalid_token", "недействительный токен")
	ErrInvalidCreds    = newError(KindUnauthorized, "invalid_credentials", "неверный email или пароль")
	ErrForbidden       = newError(KindForbidden, "forbidden", "недостаточно прав")
	ErrNotFound        = newError(KindNotFound, "not_found", "ресурс не найден")
	ErrUserNotFound    = newError(KindNotFound, "user_not_found", "пользователь не найден")
	ErrTeamNotFound    = newError(KindNotFound, "team_not_found", "команда не найдена")
	ErrEmailTaken      = newError(KindConflict, "email_taken", "пользователь с таким email уже существует")
	ErrAlreadyMember   = newError(KindConflict, "already_member", "пользователь уже состоит в команде")
	ErrInvalidRole     = newError(KindBadRequest, "invalid_role", "роль должна быть admin или member")
	ErrTaskNotFound    = newError(KindNotFound, "task_not_found", "задача не найдена")
	ErrVersionConflict = newError(KindConflict, "version_conflict", "задача была изменена другим пользователем")
	ErrInternal        = newError(KindInternal, "internal_error", "внутренняя ошибка сервера")
	ErrRateLimit       = newError(KindRateLimit, "rate_limit", "слишком много запросов, повторите позже")
)
