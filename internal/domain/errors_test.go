package domain

import (
	"errors"
	"testing"
)

func TestErrorIsMatchesCopiesAndWrappedErrors(t *testing.T) {
	cause := errors.New("db is down")
	copied := ErrTeamNotFound.WithErr(cause)

	if !errors.Is(copied, ErrTeamNotFound) {
		t.Fatal("errors.Is(copied, ErrTeamNotFound) = false, want true")
	}
	if !errors.Is(errors.Join(errors.New("context"), copied), ErrTeamNotFound) {
		t.Fatal("errors.Is(joined, ErrTeamNotFound) = false, want true")
	}
	if errors.Is(copied, ErrTaskNotFound) {
		t.Fatal("errors.Is(copied, ErrTaskNotFound) = true, want false")
	}
	if !errors.Is(copied, cause) {
		t.Fatal("errors.Is(copied, cause) = false, want true")
	}
}
