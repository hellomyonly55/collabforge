package repository

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"taskflow/internal/dto"
	"taskflow/internal/model"
	"time"
)

type TeamRepository struct{ DB *gorm.DB }
type TeamRecord struct {
	model.Team
	MyRole      string
	MemberCount int64
}
type MemberRecord struct {
	model.TeamMember
	Nickname string
	Email    string
	Status   string
}

func (r *TeamRepository) Transaction(ctx context.Context, fn func(*TeamRepository) error) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&TeamRepository{DB: tx}) })
}
func (r *TeamRepository) CreateTeam(t *model.Team) error { return r.DB.Create(t).Error }
func (r *TeamRepository) Team(id uint64, lock bool) (*model.Team, error) {
	var t model.Team
	q := r.DB
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&t, id).Error
	return &t, err
}
func (r *TeamRepository) Member(teamID, userID uint64) (*model.TeamMember, error) {
	var m model.TeamMember
	err := r.DB.Where("team_id = ? AND user_id = ? AND removed_at IS NULL", teamID, userID).First(&m).Error
	return &m, err
}
func (r *TeamRepository) AddMember(teamID, userID uint64, role string) error {
	now := time.Now().UTC()
	m := model.TeamMember{TeamID: teamID, UserID: userID, Role: role, JoinedAt: now}
	return r.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "team_id"}, {Name: "user_id"}}, DoUpdates: clause.Assignments(map[string]any{"role": role, "removed_at": nil, "joined_at": now, "updated_at": now})}).Create(&m).Error
}
func (r *TeamRepository) CountMembers(teamID uint64) (int64, error) {
	var count int64
	err := r.DB.Model(&model.TeamMember{}).Where("team_id = ? AND removed_at IS NULL", teamID).Count(&count).Error
	return count, err
}
func (r *TeamRepository) User(id uint64) (*model.User, error) {
	var u model.User
	err := r.DB.First(&u, id).Error
	return &u, err
}
func (r *TeamRepository) ListTeams(userID uint64, status string, p dto.Pagination) ([]TeamRecord, int64, error) {
	base := func() *gorm.DB {
		q := r.DB.Table("teams").Joins("JOIN team_members mine ON mine.team_id = teams.id AND mine.user_id = ? AND mine.removed_at IS NULL", userID)
		if status == "active" {
			q = q.Where("teams.archived_at IS NULL")
		}
		if status == "archived" {
			q = q.Where("teams.archived_at IS NOT NULL")
		}
		return q
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []TeamRecord{}
	err := base().Select("teams.*, mine.role AS my_role, (SELECT COUNT(*) FROM team_members members WHERE members.team_id = teams.id AND members.removed_at IS NULL) AS member_count").Order("teams.created_at DESC, teams.id DESC").Offset(p.Offset()).Limit(p.PageSize).Scan(&rows).Error
	return rows, total, err
}
func (r *TeamRepository) UpdateTeam(id uint64, values map[string]any) error {
	return r.DB.Model(&model.Team{}).Where("id = ?", id).Updates(values).Error
}
func (r *TeamRepository) SetRole(teamID, userID uint64, role string) error {
	return r.DB.Model(&model.TeamMember{}).Where("team_id = ? AND user_id = ? AND removed_at IS NULL", teamID, userID).Update("role", role).Error
}
func (r *TeamRepository) RemoveMember(teamID, userID, actorID uint64, reason string) error {
	if err := (&TaskRepository{DB: r.DB}).ClearAssignments(teamID, userID, actorID, reason); err != nil {
		return err
	}
	return r.DB.Model(&model.TeamMember{}).Where("team_id = ? AND user_id = ? AND removed_at IS NULL", teamID, userID).Update("removed_at", time.Now().UTC()).Error
}
func (r *TeamRepository) ListMembers(teamID uint64, p dto.Pagination) ([]MemberRecord, int64, error) {
	total, err := r.CountMembers(teamID)
	if err != nil {
		return nil, 0, err
	}
	rows := []MemberRecord{}
	err = r.DB.Table("team_members m").Select("m.*, users.nickname, users.email, users.status").Joins("JOIN users ON users.id = m.user_id").Where("m.team_id = ? AND m.removed_at IS NULL", teamID).Order("CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, m.id ASC").Offset(p.Offset()).Limit(p.PageSize).Scan(&rows).Error
	return rows, total, err
}
func (r *TeamRepository) CreateInvite(v *model.TeamInvite) error { return r.DB.Create(v).Error }
func (r *TeamRepository) Invite(id, teamID uint64) (*model.TeamInvite, error) {
	var v model.TeamInvite
	err := r.DB.Where("id = ? AND team_id = ?", id, teamID).First(&v).Error
	return &v, err
}
func (r *TeamRepository) InviteByHash(ctx context.Context, hash string) (*model.TeamInvite, error) {
	var v model.TeamInvite
	err := r.DB.WithContext(ctx).Where("token_hash = ?", hash).First(&v).Error
	return &v, err
}
func (r *TeamRepository) ConsumeInvite(id uint64) error {
	return r.DB.Model(&model.TeamInvite{}).Where("id = ?", id).Update("used_count", gorm.Expr("used_count + 1")).Error
}
func (r *TeamRepository) RevokeInvite(id uint64) error {
	return r.DB.Model(&model.TeamInvite{}).Where("id = ?", id).Update("revoked_at", time.Now().UTC()).Error
}
func (r *TeamRepository) ListInvites(teamID uint64, p dto.Pagination) ([]model.TeamInvite, int64, error) {
	var total int64
	if err := r.DB.Model(&model.TeamInvite{}).Where("team_id = ?", teamID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []model.TeamInvite{}
	err := r.DB.Where("team_id = ?", teamID).Order("id DESC").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error
	return rows, total, err
}
func (r *TeamRepository) Log(teamID, actorID uint64, action, targetType string, targetID uint64, metadata map[string]any) error {
	return (&ActivityRepository{DB: r.DB}).Create(teamID, nil, actorID, action, targetType, targetID, metadata)
}
func (r *TeamRepository) ListActivities(teamID uint64, p dto.Pagination) ([]ActivityRecord, int64, error) {
	return (&ActivityRepository{DB: r.DB}).List(teamID, nil, p)
}
