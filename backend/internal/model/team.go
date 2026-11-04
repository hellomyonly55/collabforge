package model

import "time"

type Team struct {
	ID          uint64
	Name        string
	Description string
	OwnerID     uint64
	ArchivedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
type TeamMember struct {
	ID        uint64
	TeamID    uint64
	UserID    uint64
	Role      string
	JoinedAt  time.Time
	RemovedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
type TeamInvite struct {
	ID        uint64
	TeamID    uint64
	TokenHash string `json:"-"`
	ExpiresAt time.Time
	MaxUses   int
	UsedCount int
	RevokedAt *time.Time
	CreatedBy uint64
	CreatedAt time.Time
	UpdatedAt time.Time
}
type ActivityLog struct {
	ID         uint64
	TeamID     uint64
	ProjectID  *uint64
	ActorID    uint64
	Action     string
	TargetType string
	TargetID   uint64
	Metadata   string `gorm:"type:json"`
	CreatedAt  time.Time
}
