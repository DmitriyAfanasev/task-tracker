package domain

import "time"

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          int64      `db:"id" json:"id"`
	TeamID      int64      `db:"team_id" json:"team_id"`
	Title       string     `db:"title" json:"title"`
	Description string     `db:"description" json:"description"`
	Status      TaskStatus `db:"status" json:"status"`
	CreatedBy   int64      `db:"created_by" json:"created_by"`
	AssigneeID  *int64     `db:"assignee_id" json:"assignee_id"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	ClosedAt    *time.Time `db:"closed_at" json:"closed_at"`
	Version     int64      `db:"version" json:"version"`
}

type TaskChange struct {
	Field string `json:"field"`
	Old   any    `json:"old"`
	New   any    `json:"new"`
}

type TaskHistoryEntry struct {
	ID        int64        `json:"id"`
	TaskID    int64        `json:"task_id"`
	ChangedBy int64        `json:"changed_by"`
	Changes   []TaskChange `json:"changes"`
	CreatedAt time.Time    `json:"created_at"`
}

type TaskComment struct {
	ID        int64     `db:"id" json:"id"`
	TaskID    int64     `db:"task_id" json:"task_id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Content   string    `db:"content" json:"content"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// TaskEditScope описывает, какие поля задачи может менять пользователь.
type TaskEditScope int

const (
	TaskEditDenied TaskEditScope = iota
	TaskEditStatusOnly
	TaskEditFull
)

// TaskScope вычисляет права на редактирование по роли в команде,
// автору задачи и исполнителю. Создатель команды и администратор
// редактируют любые задачи, автор задачи — любые её поля,
// исполнитель — только статус (и производный closed_at).
func TaskScope(teamRole Role, actorID, taskCreatedBy int64, assigneeID *int64) TaskEditScope {
	if teamRole.CanManageTeam() {
		return TaskEditFull
	}
	if actorID == taskCreatedBy {
		return TaskEditFull
	}
	if assigneeID != nil && *assigneeID == actorID {
		return TaskEditStatusOnly
	}
	return TaskEditDenied
}

func (s TaskEditScope) Allows(field string) bool {
	switch s {
	case TaskEditFull:
		return true
	case TaskEditStatusOnly:
		return field == "status" || field == "closed_at"
	default:
		return false
	}
}
