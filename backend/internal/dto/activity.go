package dto

import "time"

type ActivityFilter struct {
	Action     string
	TargetType string
	ActorID    *uint64
	ProjectID  *uint64
	From       *time.Time
	To         *time.Time
}
