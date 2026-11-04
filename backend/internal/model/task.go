package model

import "time"

type Task struct {
	ID          uint64
	ProjectID   uint64
	Title       string
	Description string
	Status      string
	Priority    string
	AssigneeID  *uint64
	CreatedBy   uint64
	DueAt       *time.Time
	Position    int64
	Version     uint64
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
