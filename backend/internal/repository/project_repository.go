package repository

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"taskflow/internal/dto"
	"taskflow/internal/model"
)

type ProjectRepository struct{ DB *gorm.DB }
type ProjectRecord struct {
	model.Project
	CreatorNickname string
}

func (r *ProjectRepository) Find(ctx context.Context, id uint64) (*model.Project, error) {
	var p model.Project
	err := r.DB.WithContext(ctx).First(&p, id).Error
	return &p, err
}
func (r *ProjectRepository) Project(id, teamID uint64, lock bool) (*model.Project, error) {
	var p model.Project
	q := r.DB.Where("id = ? AND team_id = ?", id, teamID)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&p).Error
	return &p, err
}
func (r *ProjectRepository) Create(p *model.Project) error { return r.DB.Create(p).Error }
func (r *ProjectRepository) Update(id uint64, values map[string]any) error {
	return r.DB.Model(&model.Project{}).Where("id = ?", id).Updates(values).Error
}
func (r *ProjectRepository) List(teamID uint64, status string, p dto.Pagination) ([]ProjectRecord, int64, error) {
	base := func() *gorm.DB {
		q := r.DB.Table("projects").Where("projects.team_id = ?", teamID)
		if status == "active" {
			q = q.Where("projects.archived_at IS NULL")
		}
		if status == "archived" {
			q = q.Where("projects.archived_at IS NOT NULL")
		}
		return q
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []ProjectRecord{}
	err := base().Select("projects.*, users.nickname AS creator_nickname").Joins("JOIN users ON users.id = projects.created_by").Order("projects.id DESC").Offset(p.Offset()).Limit(p.PageSize).Scan(&rows).Error
	return rows, total, err
}
