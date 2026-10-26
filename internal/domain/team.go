package domain

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// CanManageTeam — владелец и администратор могут управлять командой.
func (r Role) CanManageTeam() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanInvite — владелец может приглашать admin и member,
// администратор — только рядовых участников. Роль owner не выдаётся.
func (r Role) CanInvite(target Role) bool {
	switch r {
	case RoleOwner:
		return target == RoleAdmin || target == RoleMember
	case RoleAdmin:
		return target == RoleMember
	default:
		return false
	}
}

type Team struct {
	ID        int64     `db:"id" json:"id" example:"1"`
	Name      string    `db:"name" json:"name" example:"Платформа"`
	CreatedBy int64     `db:"created_by" json:"created_by" example:"1"`
	CreatedAt time.Time `db:"created_at" json:"created_at" example:"2026-01-01T12:00:00Z"`
}

// TeamWithRole — команда вместе с ролью текущего пользователя.
type TeamWithRole struct {
	ID        int64     `db:"id" json:"id" example:"1"`
	Name      string    `db:"name" json:"name" example:"Платформа"`
	CreatedBy int64     `db:"created_by" json:"created_by" example:"1"`
	CreatedAt time.Time `db:"created_at" json:"created_at" example:"2026-01-01T12:00:00Z"`
	Role      Role      `db:"role" json:"role" example:"owner"`
}

// TeamStats содержит агрегированный SQL-отчёт по одной команде.
// AverageCloseSeconds равен nil, когда в команде нет завершённых задач.
type TeamStats struct {
	TeamID              int64
	TasksByStatus       TasksByStatus
	TopAssignees        []TopAssignee
	AverageCloseSeconds *float64
	CommentsCount       int64
}

type TasksByStatus struct {
	Todo       int64
	InProgress int64
	Done       int64
	Cancelled  int64
}

type TopAssignee struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	ClosedTasks int64  `json:"closed_tasks"`
}
