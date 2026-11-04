package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"taskflow/internal/dto"
	"taskflow/internal/model"
)

type AdminRepository struct{ DB *gorm.DB }
type AdminUserRecord struct {
	model.User
	OwnedTeamCount int64
}
type AdminTeamRecord struct {
	model.Team
	OwnerNickname, OwnerEmail            string
	MemberCount, ProjectCount, TaskCount int64
}

func (r *AdminRepository) Transaction(ctx context.Context, fn func(*AdminRepository) error) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&AdminRepository{DB: tx}) })
}
func (r *AdminRepository) Actors(ids ...uint64) ([]model.User, error) {
	rows := []model.User{}
	err := r.DB.Where("id IN ?", ids).Order("id ASC").Clauses(clause.Locking{Strength: "UPDATE"}).Find(&rows).Error
	return rows, err
}
func (r *AdminRepository) User(id uint64) (AdminUserRecord, error) {
	var row AdminUserRecord
	err := r.DB.Table("users").Select("users.*, (SELECT COUNT(*) FROM teams WHERE teams.owner_id=users.id) AS owned_team_count").Where("users.id = ?", id).Take(&row).Error
	return row, err
}
func (r *AdminRepository) Audit(actorID uint64, action, targetType string, targetID uint64, metadata map[string]any) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return r.DB.Create(&model.PlatformAuditLog{ActorID: actorID, Action: action, TargetType: targetType, TargetID: targetID, Metadata: string(raw)}).Error
}
func adminPattern(s string) string {
	return "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(s) + "%"
}
func (r *AdminRepository) Users(f dto.AdminFilter, p dto.Pagination) ([]AdminUserRecord, int64, error) {
	base := func() *gorm.DB {
		q := r.DB.Table("users")
		if f.Status != "all" {
			q = q.Where("users.status = ?", f.Status)
		}
		if f.Role != "all" {
			q = q.Where("users.platform_role = ?", f.Role)
		}
		if f.Keyword != "" {
			v := adminPattern(f.Keyword)
			q = q.Where("(users.email LIKE ? ESCAPE '!' OR users.nickname LIKE ? ESCAPE '!')", v, v)
		}
		return q
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []AdminUserRecord{}
	err := base().Select("users.*, (SELECT COUNT(*) FROM teams WHERE teams.owner_id=users.id) AS owned_team_count").Order("users.id DESC").Offset(p.Offset()).Limit(p.PageSize).Scan(&rows).Error
	return rows, total, err
}
func (r *AdminRepository) Teams(f dto.AdminFilter, p dto.Pagination) ([]AdminTeamRecord, int64, error) {
	base := func() *gorm.DB {
		q := r.DB.Table("teams")
		if f.Status == "active" {
			q = q.Where("teams.archived_at IS NULL")
		} else if f.Status == "archived" {
			q = q.Where("teams.archived_at IS NOT NULL")
		}
		if f.Keyword != "" {
			v := adminPattern(f.Keyword)
			q = q.Where("(teams.name LIKE ? ESCAPE '!' OR teams.description LIKE ? ESCAPE '!')", v, v)
		}
		return q
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []AdminTeamRecord{}
	err := base().Select("teams.*, users.nickname AS owner_nickname, users.email AS owner_email, (SELECT COUNT(*) FROM team_members WHERE team_members.team_id=teams.id AND removed_at IS NULL) AS member_count, (SELECT COUNT(*) FROM projects WHERE projects.team_id=teams.id) AS project_count, (SELECT COUNT(*) FROM tasks JOIN projects ON projects.id=tasks.project_id WHERE projects.team_id=teams.id AND tasks.deleted_at IS NULL) AS task_count").Joins("JOIN users ON users.id=teams.owner_id").Order("teams.id DESC").Offset(p.Offset()).Limit(p.PageSize).Scan(&rows).Error
	return rows, total, err
}
func (r *AdminRepository) Statistics(today string) (dto.AdminStatistics, error) {
	var s dto.AdminStatistics
	err := r.DB.Raw(`SELECT (SELECT COUNT(*) FROM users) AS users,(SELECT COUNT(*) FROM users WHERE status='active') AS active_users,(SELECT COUNT(*) FROM users WHERE status='disabled') AS disabled_users,(SELECT COUNT(*) FROM users WHERE platform_role='admin' AND status='active') AS admins,(SELECT COUNT(*) FROM teams) AS teams,(SELECT COUNT(*) FROM teams WHERE archived_at IS NOT NULL) AS archived_teams,(SELECT COUNT(*) FROM projects) AS projects,(SELECT COUNT(*) FROM projects WHERE archived_at IS NOT NULL) AS archived_projects,(SELECT COUNT(*) FROM tasks WHERE deleted_at IS NULL) AS tasks,(SELECT COUNT(*) FROM tasks WHERE deleted_at IS NULL AND status='done') AS completed_tasks,(SELECT COUNT(*) FROM tasks WHERE deleted_at IS NULL AND status<>'done' AND due_at < ?) AS overdue_tasks,(SELECT COUNT(*) FROM comments WHERE deleted_at IS NULL) AS comments`, today).Scan(&s).Error
	return s, err
}
