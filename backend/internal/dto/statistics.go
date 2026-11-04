package dto

import "time"

type CountGroup struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}
type AssigneeStatistics struct {
	UserID    *string `json:"user_id"`
	Nickname  string  `json:"nickname"`
	Total     int64   `json:"total"`
	Completed int64   `json:"completed"`
	Overdue   int64   `json:"overdue"`
}
type Statistics struct {
	Scope          string               `json:"scope"`
	TeamID         string               `json:"team_id"`
	ProjectID      *string              `json:"project_id"`
	Name           string               `json:"name"`
	Archived       bool                 `json:"archived"`
	AsOf           time.Time            `json:"as_of"`
	Today          string               `json:"today"`
	Timezone       string               `json:"timezone"`
	Total          int64                `json:"total"`
	Completed      int64                `json:"completed"`
	Uncompleted    int64                `json:"uncompleted"`
	Overdue        int64                `json:"overdue"`
	CompletionRate float64              `json:"completion_rate"`
	ByStatus       []CountGroup         `json:"by_status"`
	ByPriority     []CountGroup         `json:"by_priority"`
	ByAssignee     []AssigneeStatistics `json:"by_assignee"`
}
