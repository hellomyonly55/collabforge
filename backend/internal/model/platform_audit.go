package model

import "time"

type PlatformAuditLog struct {
	ID         uint64
	ActorID    uint64
	Action     string
	TargetType string
	TargetID   uint64
	Metadata   string
	CreatedAt  time.Time
}
