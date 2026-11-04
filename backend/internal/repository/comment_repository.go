package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"taskflow/internal/dto"
	"taskflow/internal/model"
)

type CommentRepository struct{ DB *gorm.DB }
type CommentRecord struct {
	model.Comment
	AuthorNickname string
}

func (r *CommentRepository) Find(ctx context.Context, id uint64) (*model.Comment, error) {
	var c model.Comment
	err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&c).Error
	return &c, err
}
func (r *CommentRepository) Get(id, taskID uint64, lock bool) (*model.Comment, error) {
	q := r.DB.Where("id = ? AND task_id = ? AND deleted_at IS NULL", id, taskID)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var c model.Comment
	err := q.First(&c).Error
	return &c, err
}
func (r *CommentRepository) Create(c *model.Comment) error { return r.DB.Create(c).Error }
func (r *CommentRepository) View(id uint64) (CommentRecord, error) {
	var row CommentRecord
	err := r.DB.Table("comments").Select("comments.*, users.nickname AS author_nickname").Joins("JOIN users ON users.id = comments.author_id").Where("comments.id = ? AND comments.deleted_at IS NULL", id).Take(&row).Error
	return row, err
}
func (r *CommentRepository) List(taskID uint64, p dto.Pagination) ([]CommentRecord, int64, error) {
	base := r.DB.Table("comments").Where("comments.task_id = ? AND comments.deleted_at IS NULL", taskID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []CommentRecord{}
	err := r.DB.Table("comments").Select("comments.*, users.nickname AS author_nickname").Joins("JOIN users ON users.id = comments.author_id").Where("comments.task_id = ? AND comments.deleted_at IS NULL", taskID).Order("comments.id DESC").Offset(p.Offset()).Limit(p.PageSize).Scan(&rows).Error
	return rows, total, err
}
func (r *CommentRepository) Update(id, version uint64, values map[string]any) (bool, error) {
	values["version"] = gorm.Expr("version + 1")
	values["updated_at"] = time.Now().UTC()
	tx := r.DB.Model(&model.Comment{}).Where("id = ? AND version = ? AND deleted_at IS NULL", id, version).Updates(values)
	return tx.RowsAffected == 1, tx.Error
}
