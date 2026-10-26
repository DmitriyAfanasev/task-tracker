package domain

import "time"

type User struct {
	ID           int64     `db:"id" json:"id" example:"1"`
	Email        string    `db:"email" json:"email" example:"user@example.com"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Name         string    `db:"name" json:"name" example:"Иван Петров"`
	CreatedAt    time.Time `db:"created_at" json:"created_at" example:"2026-01-01T12:00:00Z"`
}
