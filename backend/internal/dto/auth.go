package dto

import (
	"strconv"
	"taskflow/internal/model"
	"time"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type ProfileRequest struct {
	Nickname string `json:"nickname"`
}
type PasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Nickname     string    `json:"nickname"`
	PlatformRole string    `json:"platform_role"`
	CreatedAt    time.Time `json:"created_at"`
}

func UserFrom(u *model.User) User {
	return User{ID: strconv.FormatUint(u.ID, 10), Email: u.Email, Nickname: u.Nickname, PlatformRole: u.PlatformRole, CreatedAt: u.CreatedAt.UTC()}
}
