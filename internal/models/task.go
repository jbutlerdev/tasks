package models

import (
	"time"
)

type TaskState string

const (
	TaskStateTodo       TaskState = "todo"
	TaskStateInProgress TaskState = "in_progress"
	TaskStateDone       TaskState = "done"
	TaskStateBlocked    TaskState = "blocked"
)

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	ListID      string     `json:"list_id"`
	State       TaskState  `json:"state"`
	StateTime   time.Time  `json:"state_time"` // When this state was set
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Notes       []Note     `json:"notes,omitempty"`
	SubTasks    []Task     `json:"sub_tasks,omitempty"`
}

type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TaskList struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Time helper functions

// TimeInState returns the duration the task has been in the current state
func (t *Task) TimeInState() time.Duration {
	return time.Since(t.StateTime)
}

// SetState updates the task state and resets the state timer
func (t *Task) SetState(state TaskState) {
	t.State = state
	t.StateTime = time.Now()
	t.UpdatedAt = time.Now()
}