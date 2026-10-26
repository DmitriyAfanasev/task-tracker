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
