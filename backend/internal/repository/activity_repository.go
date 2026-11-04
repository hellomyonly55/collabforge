package repository

import (
	"encoding/json"
	"gorm.io/gorm"
	"taskflow/internal/dto"
	"taskflow/internal/model"
)

// ActivityRepository must use the caller's transaction when writing a business event.
type ActivityRepository struct{ DB *gorm.DB }
type ActivityRecord struct {
	model.ActivityLog
	ActorNickname string
}

func (r *ActivityRepository) Create(teamID uint64, projectID *uint64, actorID uint64, action, targetType string, targetID uint64, metadata map[string]any) error {
	value, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return r.DB.Create(&model.ActivityLog{TeamID: teamID, ProjectID: projectID, ActorID: actorID, Action: action, TargetType: targetType, TargetID: targetID, Metadata: string(value)}).Error
}
func (r *ActivityRepository) List(teamID uint64, projectID *uint64, p dto.Pagination, filters ...dto.ActivityFilter) ([]ActivityRecord, int64, error) {
	var f dto.ActivityFilter
	if len(filters) > 0 {
		f = filters[0]
	}
	base := func() *gorm.DB {
		q := r.DB.Table("activity_logs").Where("activity_logs.team_id = ?", teamID)
		if projectID != nil {
			q = q.Where("activity_logs.project_id = ?", *projectID)
		}
		if f.Action != "" {
			q = q.Where("activity_logs.action = ?", f.Action)
		}
		if f.TargetType != "" {
			q = q.Where("activity_logs.target_type = ?", f.TargetType)
		}
		if f.ActorID != nil {
			q = q.Where("activity_logs.actor_id = ?", *f.ActorID)
		}
		if f.ProjectID != nil {
			q = q.Where("activity_logs.project_id = ?", *f.ProjectID)
		}
		if f.From != nil {
			q = q.Where("activity_logs.created_at >= ?", *f.From)
		}
		if f.To != nil {
			q = q.Where("activity_logs.created_at <= ?", *f.To)
		}
		return q
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []ActivityRecord{}
	err := base().Select("activity_logs.*, users.nickname AS actor_nickname").Joins("JOIN users ON users.id = activity_logs.actor_id").Order("activity_logs.id DESC").Offset(p.Offset()).Limit(p.PageSize).Scan(&rows).Error
	return rows, total, err
}
