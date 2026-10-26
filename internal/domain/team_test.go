package domain

import "testing"

func TestRoleCanManageTeam(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleMember, false},
		{Role("unknown"), false},
	}

	for _, tt := range tests {
		if got := tt.role.CanManageTeam(); got != tt.want {
			t.Errorf("%q.CanManageTeam() = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestRoleCanInvite(t *testing.T) {
	tests := []struct {
		caller Role
		target Role
		want   bool
	}{
		{RoleOwner, RoleAdmin, true},
		{RoleOwner, RoleMember, true},
		{RoleOwner, RoleOwner, false},
		{RoleAdmin, RoleAdmin, false},
		{RoleAdmin, RoleMember, true},
		{RoleAdmin, RoleOwner, false},
		{RoleMember, RoleMember, false},
		{RoleMember, RoleAdmin, false},
	}

	for _, tt := range tests {
		if got := tt.caller.CanInvite(tt.target); got != tt.want {
			t.Errorf("%q.CanInvite(%q) = %v, want %v", tt.caller, tt.target, got, tt.want)
		}
	}
}
