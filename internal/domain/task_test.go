package domain

import "testing"

func TestTaskStatusValid(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   bool
	}{
		{TaskStatusTodo, true},
		{TaskStatusInProgress, true},
		{TaskStatusDone, true},
		{TaskStatusCancelled, true},
		{TaskStatus("unknown"), false},
		{TaskStatus(""), false},
	}

	for _, tt := range tests {
		if got := tt.status.Valid(); got != tt.want {
			t.Errorf("%q.Valid() = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestTaskScope(t *testing.T) {
	assigneeID := int64(2)
	otherID := int64(3)

	tests := []struct {
		name          string
		role          Role
		actorID       int64
		taskCreatedBy int64
		assigneeID    *int64
		want          TaskEditScope
	}{
		{"owner edits any task", RoleOwner, 1, 9, &otherID, TaskEditFull},
		{"admin edits any task", RoleAdmin, 1, 9, &otherID, TaskEditFull},
		{"creator edits own task", RoleMember, 2, 2, nil, TaskEditFull},
		{"assignee edits status", RoleMember, 2, 9, &assigneeID, TaskEditStatusOnly},
		{"other member denied", RoleMember, 2, 9, &otherID, TaskEditDenied},
		{"member without assignee denied", RoleMember, 2, 9, nil, TaskEditDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TaskScope(tt.role, tt.actorID, tt.taskCreatedBy, tt.assigneeID)
			if got != tt.want {
				t.Fatalf("TaskScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskEditScopeAllows(t *testing.T) {
	tests := []struct {
		scope TaskEditScope
		field string
		want  bool
	}{
		{TaskEditFull, "title", true},
		{TaskEditFull, "assignee_id", true},
		{TaskEditStatusOnly, "status", true},
		{TaskEditStatusOnly, "closed_at", true},
		{TaskEditStatusOnly, "title", false},
		{TaskEditDenied, "status", false},
	}

	for _, tt := range tests {
		if got := tt.scope.Allows(tt.field); got != tt.want {
			t.Errorf("scope %v Allows(%q) = %v, want %v", tt.scope, tt.field, got, tt.want)
		}
	}
}
