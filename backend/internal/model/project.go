package model

import "time"

type Project struct {
	TaskRevision uint64
	ID           uint64
	TeamID       uint64
	Name         string
	Description  string
	CreatedBy    uint64
	ArchivedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
