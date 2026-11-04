package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"taskflow/internal/dto"
	"taskflow/internal/model"
)

type TaskRepository struct{ DB *gorm.DB }
type TaskRecord struct {
	model.Task
	ProjectName       string
	TeamID            uint64
	TeamName          string
	MyRole            string
	ProjectArchivedAt *time.Time
	TeamArchivedAt    *time.Time
	AssigneeNickname  *string
	CreatorNickname   string
}

func (r *TaskRepository) Find(ctx context.Context, id uint64) (*model.Task, error) {
	var task model.Task
	err := r.DB.WithContext(ctx).Where("deleted_at IS NULL").First(&task, id).Error
	return &task, err
}
func (r *TaskRepository) Task(id, projectID uint64, lock bool) (*model.Task, error) {
	var task model.Task
	q := r.DB.Where("id = ? AND project_id = ? AND deleted_at IS NULL", id, projectID)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&task).Error
	return &task, err
}
func (r *TaskRepository) Create(task *model.Task) error { return r.DB.Create(task).Error }
func (r *TaskRepository) Update(id, version uint64, values map[string]any) (bool, error) {
	values["version"] = gorm.Expr("version + 1")
	result := r.DB.Model(&model.Task{}).Where("id = ? AND version = ? AND deleted_at IS NULL", id, version).Updates(values)
	return result.RowsAffected == 1, result.Error
}
func (r *TaskRepository) BumpRevision(projectID uint64) error {
	return r.DB.Model(&model.Project{}).Where("id = ?", projectID).UpdateColumn("task_revision", gorm.Expr("task_revision + 1")).Error
}
func (r *TaskRepository) Ordered(projectID uint64, status string) ([]model.Task, error) {
	rows := []model.Task{}
	err := r.DB.Select("id, status, position, version").Where("project_id = ? AND status = ? AND deleted_at IS NULL", projectID, status).Order("position ASC, id ASC").Find(&rows).Error
	return rows, err
}

// Project lock serializes the entire reorder. Only rows whose order changes get a new version.
func (r *TaskRepository) Reorder(projectID uint64, status string, rows []model.Task) error {
	for i, row := range rows {
		if row.Status == status && row.Position == int64(i) {
			continue
		}
		if err := r.DB.Model(&model.Task{}).Where("id = ? AND project_id = ? AND deleted_at IS NULL", row.ID, projectID).Updates(map[string]any{"status": status, "position": i, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
	}
	return nil
}
func (r *TaskRepository) rowBase(actorID uint64) *gorm.DB {
	return r.DB.Table("tasks").Joins("JOIN projects ON projects.id = tasks.project_id").Joins("JOIN teams ON teams.id = projects.team_id").Joins("JOIN team_members task_member ON task_member.team_id = teams.id AND task_member.user_id = ? AND task_member.removed_at IS NULL", actorID).Joins("JOIN users creator ON creator.id = tasks.created_by").Joins("LEFT JOIN users assignee ON assignee.id = tasks.assignee_id").Where("tasks.deleted_at IS NULL")
}

const taskColumns = "tasks.*, projects.name AS project_name, projects.team_id, projects.archived_at AS project_archived_at, teams.name AS team_name, teams.archived_at AS team_archived_at, task_member.role AS my_role, creator.nickname AS creator_nickname, assignee.nickname AS assignee_nickname"

func (r *TaskRepository) View(id, actorID uint64) (TaskRecord, error) {
	var row TaskRecord
	err := r.rowBase(actorID).Select(taskColumns).Where("tasks.id = ?", id).Take(&row).Error
	return row, err
}
func taskConditions(q *gorm.DB, f dto.TaskFilter, today string) *gorm.DB {
	if f.Keyword != "" {
		pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(f.Keyword) + "%"
		q = q.Where("(tasks.title LIKE ? ESCAPE '!' OR tasks.description LIKE ? ESCAPE '!')", pattern, pattern)
	}
	if f.Status != "" {
		q = q.Where("tasks.status = ?", f.Status)
	}
	if f.Priority != "" {
		q = q.Where("tasks.priority = ?", f.Priority)
	}
	if f.Unassigned {
		q = q.Where("tasks.assignee_id IS NULL")
	} else if f.AssigneeID != nil {
		q = q.Where("tasks.assignee_id = ?", *f.AssigneeID)
	}
	if f.Overdue != nil {
		if *f.Overdue {
			q = q.Where("tasks.status <> 'done' AND tasks.due_at < ?", today)
		} else {
			q = q.Where("(tasks.status = 'done' OR tasks.due_at IS NULL OR tasks.due_at >= ?)", today)
		}
	}
	return q
}
func (r *TaskRepository) List(actorID, projectID uint64, mine bool, f dto.TaskFilter, p dto.Pagination, today string) ([]TaskRecord, int64, error) {
	base := func() *gorm.DB {
		q := r.rowBase(actorID)
		if projectID != 0 {
			q = q.Where("tasks.project_id = ?", projectID)
		}
		if mine {
			q = q.Where("tasks.assignee_id = ?", actorID)
		}
		return taskConditions(q, f, today)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := base().Select(taskColumns)
	if mine {
		q = q.Order("tasks.due_at IS NULL ASC, tasks.due_at ASC, tasks.updated_at DESC, tasks.id DESC")
	} else {
		q = q.Order("CASE tasks.status WHEN 'todo' THEN 0 WHEN 'in_progress' THEN 1 WHEN 'in_review' THEN 2 ELSE 3 END, tasks.position ASC, tasks.id ASC")
	}
	rows := []TaskRecord{}
	err := q.Offset(p.Offset()).Limit(p.PageSize).Scan(&rows).Error
	return rows, total, err
}

// Caller holds the team lock. This system cleanup also covers archived projects and increments every affected task version.
func (r *TaskRepository) ClearAssignments(teamID, userID, actorID uint64, reason string) error {
	rows := []model.Task{}
	if err := r.DB.Table("tasks").Select("tasks.*").Joins("JOIN projects ON projects.id = tasks.project_id").Where("projects.team_id = ? AND tasks.assignee_id = ? AND tasks.deleted_at IS NULL", teamID, userID).Order("tasks.project_id ASC, tasks.id ASC").Scan(&rows).Error; err != nil {
		return err
	}
	changedProjects := map[uint64]bool{}
	for _, task := range rows {
		ok, err := r.Update(task.ID, task.Version, map[string]any{"assignee_id": nil})
		if err != nil {
			return err
		}
		if !ok {
			return gorm.ErrInvalidTransaction
		}
		if err := (&ActivityRepository{DB: r.DB}).Create(teamID, &task.ProjectID, actorID, "task.unassigned", "task", task.ID, map[string]any{"previous_assignee_id": strconv.FormatUint(userID, 10), "reason": reason}); err != nil {
			return err
		}
		changedProjects[task.ProjectID] = true
	}
	for projectID := range changedProjects {
		if err := r.BumpRevision(projectID); err != nil {
			return err
		}
	}
	return nil
}
