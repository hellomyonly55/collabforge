package dto

import "time"

type CreateCommentRequest struct {
	Content string `json:"content"`
}
type UpdateCommentRequest struct {
	Content string `json:"content"`
	Version uint64 `json:"version"`
}
type Comment struct {
	ID             string    `json:"id"`
	TaskID         string    `json:"task_id"`
	AuthorID       string    `json:"author_id"`
	AuthorNickname string    `json:"author_nickname"`
	Content        string    `json:"content"`
	Version        uint64    `json:"version"`
	CanEdit        bool      `json:"can_edit"`
	CanDelete      bool      `json:"can_delete"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
