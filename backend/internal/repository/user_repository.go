package repository

import (
	"context"
	"gorm.io/gorm"
	"taskflow/internal/model"
)

type UserRepository struct{ DB *gorm.DB }

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	return r.DB.WithContext(ctx).Create(u).Error
}
func (r *UserRepository) ByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.DB.WithContext(ctx).Where("email = ?", email).First(&u).Error
	return &u, err
}
func (r *UserRepository) ByID(ctx context.Context, id uint64) (*model.User, error) {
	var u model.User
	err := r.DB.WithContext(ctx).First(&u, id).Error
	return &u, err
}
func (r *UserRepository) UpdateNickname(ctx context.Context, u *model.User, name string) (bool, error) {
	result := r.DB.WithContext(ctx).Model(&model.User{}).Where("id = ? AND status = ? AND session_version = ?", u.ID, "active", u.SessionVersion).Update("nickname", name)
	return result.RowsAffected == 1, result.Error
}

// Compare-and-swap prevents two concurrent password changes from both succeeding.
func (r *UserRepository) UpdatePassword(ctx context.Context, u *model.User, hash string) (bool, error) {
	result := r.DB.WithContext(ctx).Model(&model.User{}).Where("id = ? AND status = ? AND session_version = ?", u.ID, "active", u.SessionVersion).Updates(map[string]any{"password_hash": hash, "session_version": gorm.Expr("session_version + 1")})
	return result.RowsAffected == 1, result.Error
}
