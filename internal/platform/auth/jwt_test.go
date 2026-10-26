package auth

import (
	"errors"
	"testing"
	"time"
)

func TestJWTIssueAndParse(t *testing.T) {
	issuer := NewJWTIssuer("0123456789abcdef0123456789abcdef", time.Hour)

	token, expiresAt, err := issuer.Issue(42)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if token == "" {
		t.Fatal("Issue() returned empty token")
	}
	if time.Until(expiresAt) <= 0 {
		t.Fatalf("expiresAt = %v, want in the future", expiresAt)
	}

	userID, err := issuer.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if userID != 42 {
		t.Fatalf("Parse() userID = %d, want 42", userID)
	}
}

func TestJWTParseRejectsInvalidToken(t *testing.T) {
	issuer := NewJWTIssuer("0123456789abcdef0123456789abcdef", time.Hour)
	other := NewJWTIssuer("fedcba9876543210fedcba9876543210", time.Hour)

	token, _, err := other.Issue(1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := issuer.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
	}
	if _, err := issuer.Parse("not-a-jwt"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

func TestJWTParseRejectsExpiredToken(t *testing.T) {
	issuer := NewJWTIssuer("0123456789abcdef0123456789abcdef", -time.Minute)

	token, _, err := issuer.Issue(1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := issuer.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
	}
}
