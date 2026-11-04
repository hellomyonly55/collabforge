package dto

import (
	"encoding/json"
	"time"
)

// NullableString distinguishes an omitted PATCH field from explicit JSON null (clear it).
type NullableString struct {
	Set   bool
	Value *string
}

func (v *NullableString) UnmarshalJSON(data []byte) error {
	v.Set = true
	return json.Unmarshal(data, &v.Value)
}

type CreateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	AssigneeID  *string `json:"assignee_id"`
	DueAt       *string `json:"due_at"`
}
type UpdateTaskRequest struct {
	Version     uint64         `json:"version"`
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Priority    *string        `json:"priority"`
	AssigneeID  NullableString `json:"assignee_id"`
	DueAt       NullableString `json:"due_at"`
}
type MoveTaskRequest struct {
	Version      uint64  `json:"version"`
	Status       string  `json:"status"`
	BeforeTaskID *string `json:"before_task_id"`
}
type TaskFilter struct {
	Keyword, Status, Priority string
	AssigneeID                *uint64
	Unassigned                bool
	Overdue                   *bool
	Revision                  *uint64
}
type Task struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"project_id"`
	ProjectName       string     `json:"project_name"`
	TeamID            string     `json:"team_id"`
	TeamName          string     `json:"team_name"`
	MyRole            string     `json:"my_role"`
	ProjectArchivedAt *time.Time `json:"project_archived_at"`
	TeamArchivedAt    *time.Time `json:"team_archived_at"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	Status            string     `json:"status"`
	Priority          string     `json:"priority"`
	AssigneeID        *string    `json:"assignee_id"`
	AssigneeNickname  *string    `json:"assignee_nickname"`
	CreatedBy         string     `json:"created_by"`
	CreatorNickname   string     `json:"creator_nickname"`
	DueAt             *string    `json:"due_at"`
	IsOverdue         bool       `json:"is_overdue"`
	Position          int64      `json:"position"`
	Version           uint64     `json:"version"`
	CanDelete         bool       `json:"can_delete"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
type TaskPage struct {
	Page[Task]
	Revision *uint64 `json:"revision,omitempty"`
}
type BoardColumn struct {
	Status   string `json:"status"`
	Items    []Task `json:"items"`
	Total    int64  `json:"total"`
	PageSize int    `json:"page_size"`
	HasMore  bool   `json:"has_more"`
}
type Board struct {
	Project  Project       `json:"project"`
	Revision uint64        `json:"revision"`
	Columns  []BoardColumn `json:"columns"`
}
