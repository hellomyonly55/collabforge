package model

import "time"

// Comment is soft-deleted; public responses are always constructed explicitly as DTOs.
type Comment struct {
	ID        uint64
	TaskID    uint64
	AuthorID  uint64
	Content   string
	Version   uint64
	DeletedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
