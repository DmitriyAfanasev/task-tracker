package httpapi

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

// BusinessMetrics holds business-level Prometheus metrics.
type BusinessMetrics struct {
	// TasksCreatedTotal is a counter of total tasks created by team.
	TasksCreatedTotal *prometheus.CounterVec
	// TasksCompletedTotal is a counter of total tasks completed by team/user.
	TasksCompletedTotal *prometheus.CounterVec
	// TeamsCreatedTotal is a counter of total teams created.
	TeamsCreatedTotal *prometheus.CounterVec
	// ActiveUsers tracks unique active users (gauges).
	ActiveUsers *prometheus.GaugeVec
}

// NewBusinessMetrics creates new business metrics and registers them.
func NewBusinessMetrics() *BusinessMetrics {
	tasksCreated := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "secunda",
		Name:      "tasks_created_total",
		Help:      "Total number of tasks created, labeled by team_id.",
	}, []string{"team_id"})

	tasksCompleted := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "secunda",
		Name:      "tasks_completed_total",
		Help:      "Total number of tasks completed, labeled by team_id and assignee_id.",
	}, []string{"team_id", "assignee_id"})

	teamsCreated := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "secunda",
		Name:      "teams_created_total",
		Help:      "Total number of teams created.",
	}, []string{})

	activeUsers := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "secunda",
		Name:      "active_users",
		Help:      "Number of active users observed.",
	}, []string{})

	prometheus.MustRegister(tasksCreated, tasksCompleted, teamsCreated, activeUsers)

	return &BusinessMetrics{
		TasksCreatedTotal: tasksCreated,
		TasksCompletedTotal: tasksCompleted,
		TeamsCreatedTotal: teamsCreated,
		ActiveUsers:        activeUsers,
	}
}

// IncrementTasksCreated increments the tasks created counter for a team.
func IncrementTasksCreated(metrics *BusinessMetrics, teamID int64) {
	metrics.TasksCreatedTotal.WithLabelValues(fmt.Sprintf("%d", teamID)).Add(1)
}

// IncrementTasksCompleted increments the tasks completed counter.
func IncrementTasksCompleted(metrics *BusinessMetrics, teamID int64, assigneeID int64) {
	metrics.TasksCompletedTotal.WithLabelValues(fmt.Sprintf("%d", teamID), fmt.Sprintf("%d", assigneeID)).Add(1)
}

// IncrementTeamsCreated increments the teams created counter.
func IncrementTeamsCreated(metrics *BusinessMetrics) {
	metrics.TeamsCreatedTotal.WithLabelValues("").Add(1)
}

// SetActiveUsers sets the active users gauge.
func SetActiveUsers(metrics *BusinessMetrics, count int64) {
	metrics.ActiveUsers.WithLabelValues("").Set(float64(count))
}