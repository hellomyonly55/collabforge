package model

import "time"

// User is a database entity. Return dto.User instead of serializing this entity.
type User struct {
	ID             uint64    `json:"-"`
	Email          string    `json:"-"`
	Nickname       string    `json:"-"`
	PasswordHash   string    `json:"-"`
	PlatformRole   string    `json:"-"`
	Status         string    `json:"-"`
	SessionVersion uint64    `json:"-"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}
