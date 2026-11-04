package repository

import (
	"gorm.io/gorm"

	"taskflow/internal/dto"
)

// StatisticsRepository uses the caller's read transaction, never a cached board page.
type StatisticsRepository struct{ DB *gorm.DB }
type StatisticsTotals struct{ Total, Completed, Overdue int64 }
type AssigneeAggregate struct {
	UserID                    *uint64
	Nickname                  string
	Total, Completed, Overdue int64
}

func (r *StatisticsRepository) base(teamID uint64, projectID *uint64) *gorm.DB {
	q := r.DB.Table("tasks").Joins("JOIN projects ON projects.id = tasks.project_id").Where("projects.team_id = ? AND tasks.deleted_at IS NULL", teamID)
	if projectID != nil {
		q = q.Where("tasks.project_id = ?", *projectID)
	}
	return q
}
func (r *StatisticsRepository) Aggregate(teamID uint64, projectID *uint64, today string) (StatisticsTotals, []dto.CountGroup, []dto.CountGroup, []AssigneeAggregate, error) {
	var totals StatisticsTotals
	err := r.base(teamID, projectID).Select("COUNT(*) AS total, COALESCE(SUM(tasks.status = 'done'),0) AS completed, COALESCE(SUM(tasks.status <> 'done' AND tasks.due_at < ?),0) AS overdue", today).Scan(&totals).Error
	if err != nil {
		return totals, nil, nil, nil, err
	}
	statuses := []dto.CountGroup{}
	if err = r.base(teamID, projectID).Select("tasks.status AS `key`, COUNT(*) AS count").Group("tasks.status").Scan(&statuses).Error; err != nil {
		return totals, nil, nil, nil, err
	}
	priorities := []dto.CountGroup{}
	if err = r.base(teamID, projectID).Select("tasks.priority AS `key`, COUNT(*) AS count").Group("tasks.priority").Scan(&priorities).Error; err != nil {
		return totals, nil, nil, nil, err
	}
	assignees := []AssigneeAggregate{}
	err = r.base(teamID, projectID).Joins("LEFT JOIN users ON users.id = tasks.assignee_id").Select("tasks.assignee_id AS user_id, COALESCE(users.nickname, '未分配') AS nickname, COUNT(*) AS total, COALESCE(SUM(tasks.status = 'done'),0) AS completed, COALESCE(SUM(tasks.status <> 'done' AND tasks.due_at < ?),0) AS overdue", today).Group("tasks.assignee_id, users.nickname").Order("total DESC, tasks.assignee_id ASC").Scan(&assignees).Error
	return totals, statuses, priorities, assignees, err
}
