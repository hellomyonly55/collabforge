package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/internal/repository"
)

var taskStatuses = []string{"todo", "in_progress", "in_review", "done"}

func validStatus(value string) bool {
	for _, status := range taskStatuses {
		if status == value {
			return true
		}
	}
	return false
}
func validPriority(value string) bool {
	return value == "low" || value == "medium" || value == "high" || value == "urgent"
}
func taskToday() string {
	return time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format(time.DateOnly)
}
func taskDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	date, err := time.Parse(time.DateOnly, *value)
	if err != nil || date.Year() < 1000 || date.Year() > 9999 {
		return nil, Fail(400, "VALIDATION_ERROR", "截止日期必须为 YYYY-MM-DD 格式的有效日期")
	}
	return &date, nil
}
func taskAssignee(r *repository.TeamRepository, teamID uint64, value *string) (*uint64, error) {
	if value == nil {
		return nil, nil
	}
	id, err := strconv.ParseUint(*value, 10, 64)
	if err != nil || id == 0 {
		return nil, Fail(400, "VALIDATION_ERROR", "负责人 ID 无效")
	}
	if _, err = r.Member(teamID, id); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, Fail(400, "INVALID_ASSIGNEE", "负责人必须是当前团队成员")
	} else if err != nil {
		return nil, err
	}
	user, err := r.User(id)
	if err != nil {
		return nil, err
	}
	if user.Status != "active" {
		return nil, Fail(400, "INVALID_ASSIGNEE", "不能分配给已禁用的用户")
	}
	return &id, nil
}
func validateTaskFilter(f dto.TaskFilter) error {
	if utf8.RuneCountInString(f.Keyword) > 100 {
		return Fail(400, "VALIDATION_ERROR", "关键词最多 100 个字符")
	}
	if f.Status != "" && !validStatus(f.Status) {
		return Fail(400, "VALIDATION_ERROR", "任务状态无效")
	}
	if f.Priority != "" && !validPriority(f.Priority) {
		return Fail(400, "VALIDATION_ERROR", "优先级无效")
	}
	return nil
}
func versionConflict() error {
	return Fail(409, "VERSION_CONFLICT", "任务已被其他操作修改，请刷新后重试")
}
func taskError(err error) error {
	if err == nil {
		return nil
	}
	var app *AppError
	if errors.Is(err, gorm.ErrRecordNotFound) || (errors.As(err, &app) && app.Status == 404) {
		return Fail(404, "NOT_FOUND", "任务不存在或你没有访问权限")
	}
	return teamError(err)
}
func taskFrom(row repository.TaskRecord, actorID uint64, today string) dto.Task {
	var assignee, due *string
	if row.AssigneeID != nil {
		value := strconv.FormatUint(*row.AssigneeID, 10)
		assignee = &value
	}
	if row.DueAt != nil {
		value := row.DueAt.Format(time.DateOnly)
		due = &value
	}
	readonly := row.ProjectArchivedAt != nil || row.TeamArchivedAt != nil
	return dto.Task{
		ID:                strconv.FormatUint(row.ID, 10),
		ProjectID:         strconv.FormatUint(row.ProjectID, 10),
		ProjectName:       row.ProjectName,
		TeamID:            strconv.FormatUint(row.TeamID, 10),
		TeamName:          row.TeamName,
		MyRole:            row.MyRole,
		ProjectArchivedAt: row.ProjectArchivedAt,
		TeamArchivedAt:    row.TeamArchivedAt,
		Title:             row.Title,
		Description:       row.Description,
		Status:            row.Status,
		Priority:          row.Priority,
		AssigneeID:        assignee,
		AssigneeNickname:  row.AssigneeNickname,
		CreatedBy:         strconv.FormatUint(row.CreatedBy, 10),
		CreatorNickname:   row.CreatorNickname,
		DueAt:             due,
		IsOverdue:         due != nil && *due < today && row.Status != "done",
		Position:          row.Position,
		Version:           row.Version,
		CanDelete:         !readonly && (row.CreatedBy == actorID || manager(row.MyRole)),
		CreatedAt:         row.CreatedAt.UTC(),
		UpdatedAt:         row.UpdatedAt.UTC(),
	}
}

type TaskService struct {
	Projects *ProjectService
	Repo     *repository.TaskRepository
}

func (s *TaskService) inTask(ctx context.Context, id, actorID uint64, write bool, fn func(*repository.TeamRepository, *repository.TaskRepository, *model.Project, *model.TeamMember, *model.Task) error) error {
	hint, err := s.Repo.Find(ctx, id)
	if err != nil {
		return taskError(err)
	}
	err = s.Projects.inProjectAccess(ctx, hint.ProjectID, actorID, write, false, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		r := &repository.TaskRepository{DB: tr.DB}
		task, err := r.Task(id, p.ID, write)
		if err != nil {
			return err
		}
		return fn(tr, r, p, m, task)
	})
	return taskError(err)
}
func taskView(r *repository.TaskRepository, id, actorID uint64) (dto.Task, error) {
	row, err := r.View(id, actorID)
	if err != nil {
		return dto.Task{}, err
	}
	return taskFrom(row, actorID, taskToday()), nil
}
func logTask(tr *repository.TeamRepository, p *model.Project, actorID uint64, action string, id uint64, metadata map[string]any) error {
	return (&repository.ActivityRepository{DB: tr.DB}).Create(p.TeamID, &p.ID, actorID, action, "task", id, metadata)
}
func (s *TaskService) Create(ctx context.Context, projectID, actorID uint64, req dto.CreateTaskRequest) (dto.Task, error) {
	var result dto.Task
	title, err := boundedText(req.Title, "任务标题", 1, 200)
	if err != nil {
		return result, err
	}
	description, err := boundedText(req.Description, "任务描述", 0, 2000)
	if err != nil {
		return result, err
	}
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	if !validPriority(priority) {
		return result, Fail(400, "VALIDATION_ERROR", "优先级无效")
	}
	due, err := taskDate(req.DueAt)
	if err != nil {
		return result, err
	}
	err = s.Projects.inProjectAccess(ctx, projectID, actorID, true, false, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		assignee, err := taskAssignee(tr, t.ID, req.AssigneeID)
		if err != nil {
			return err
		}
		r := &repository.TaskRepository{DB: tr.DB}
		rows, err := r.Ordered(p.ID, "todo")
		if err != nil {
			return err
		}
		task := &model.Task{ProjectID: p.ID, Title: title, Description: description, Status: "todo", Priority: priority, AssigneeID: assignee, CreatedBy: actorID, DueAt: due, Position: int64(len(rows)), Version: 1}
		if err = r.Create(task); err != nil {
			return err
		}
		if err = r.BumpRevision(p.ID); err != nil {
			return err
		}
		if err = logTask(tr, p, actorID, "task.created", task.ID, map[string]any{"title": title}); err != nil {
			return err
		}
		result, err = taskView(r, task.ID, actorID)
		return err
	})
	return result, taskError(err)
}
func (s *TaskService) Get(ctx context.Context, id, actorID uint64) (dto.Task, error) {
	var result dto.Task
	err := s.inTask(ctx, id, actorID, false, func(tr *repository.TeamRepository, r *repository.TaskRepository, p *model.Project, m *model.TeamMember, task *model.Task) error {
		var err error
		result, err = taskView(r, id, actorID)
		return err
	})
	return result, err
}
func (s *TaskService) Update(ctx context.Context, id, actorID uint64, req dto.UpdateTaskRequest) (dto.Task, error) {
	var result dto.Task
	if req.Version == 0 {
		return result, Fail(400, "VALIDATION_ERROR", "必须提供任务版本")
	}
	values := map[string]any{}
	if req.Title != nil {
		value, err := boundedText(*req.Title, "任务标题", 1, 200)
		if err != nil {
			return result, err
		}
		values["title"] = value
	}
	if req.Description != nil {
		value, err := boundedText(*req.Description, "任务描述", 0, 2000)
		if err != nil {
			return result, err
		}
		values["description"] = value
	}
	if req.Priority != nil {
		if !validPriority(*req.Priority) {
			return result, Fail(400, "VALIDATION_ERROR", "优先级无效")
		}
		values["priority"] = *req.Priority
	}
	if req.DueAt.Set {
		value, err := taskDate(req.DueAt.Value)
		if err != nil {
			return result, err
		}
		values["due_at"] = value
	}
	if len(values) == 0 && !req.AssigneeID.Set {
		return result, Fail(400, "VALIDATION_ERROR", "请提供要更新的任务资料")
	}
	err := s.inTask(ctx, id, actorID, true, func(tr *repository.TeamRepository, r *repository.TaskRepository, p *model.Project, m *model.TeamMember, task *model.Task) error {
		if task.Version != req.Version {
			return versionConflict()
		}
		if req.AssigneeID.Set {
			value, err := taskAssignee(tr, p.TeamID, req.AssigneeID.Value)
			if err != nil {
				return err
			}
			values["assignee_id"] = value
		}
		ok, err := r.Update(id, req.Version, values)
		if err != nil {
			return err
		}
		if !ok {
			return versionConflict()
		}
		if err = r.BumpRevision(p.ID); err != nil {
			return err
		}
		// Log business fields only, not SQL expressions or internal version mechanics.
		metadata := map[string]any{"title": task.Title}
		if req.Title != nil {
			metadata["title"] = *req.Title
		}
		if req.AssigneeID.Set {
			metadata["assignee_id"] = req.AssigneeID.Value
		}
		if req.Priority != nil {
			metadata["priority"] = *req.Priority
		}
		if req.DueAt.Set {
			metadata["due_at"] = req.DueAt.Value
		}
		if req.Description != nil {
			metadata["description_changed"] = true
		}
		if err = logTask(tr, p, actorID, "task.updated", id, metadata); err != nil {
			return err
		}
		result, err = taskView(r, id, actorID)
		return err
	})
	return result, err
}
func withoutTask(rows []model.Task, id uint64) []model.Task {
	result := make([]model.Task, 0, len(rows))
	for _, row := range rows {
		if row.ID != id {
			result = append(result, row)
		}
	}
	return result
}
func (s *TaskService) Move(ctx context.Context, id, actorID uint64, req dto.MoveTaskRequest) (dto.Task, error) {
	var result dto.Task
	if req.Version == 0 || !validStatus(req.Status) {
		return result, Fail(400, "VALIDATION_ERROR", "任务版本或状态无效")
	}
	var before uint64
	if req.BeforeTaskID != nil {
		value, err := strconv.ParseUint(*req.BeforeTaskID, 10, 64)
		if err != nil || value == 0 {
			return result, Fail(400, "VALIDATION_ERROR", "目标任务 ID 无效")
		}
		before = value
	}
	if before == id {
		return result, Fail(400, "VALIDATION_ERROR", "不能放到任务自身前面")
	}
	err := s.inTask(ctx, id, actorID, true, func(tr *repository.TeamRepository, r *repository.TaskRepository, p *model.Project, m *model.TeamMember, task *model.Task) error {
		if task.Version != req.Version {
			return versionConflict()
		}
		source, err := r.Ordered(p.ID, task.Status)
		if err != nil {
			return err
		}
		target := source
		if task.Status != req.Status {
			target, err = r.Ordered(p.ID, req.Status)
			if err != nil {
				return err
			}
		}
		target = withoutTask(target, id)
		index := len(target)
		if before != 0 {
			index = -1
			for i, row := range target {
				if row.ID == before {
					index = i
					break
				}
			}
			if index < 0 {
				return Fail(409, "MOVE_TARGET_CHANGED", "目标任务已变化，请刷新后重试")
			}
		}
		if task.Status == req.Status && task.Position == int64(index) {
			result, err = taskView(r, id, actorID)
			return err
		}
		ordered := make([]model.Task, 0, len(target)+1)
		ordered = append(ordered, target[:index]...)
		ordered = append(ordered, *task)
		ordered = append(ordered, target[index:]...)
		if task.Status != req.Status {
			if err = r.Reorder(p.ID, task.Status, withoutTask(source, id)); err != nil {
				return err
			}
		}
		if err = r.Reorder(p.ID, req.Status, ordered); err != nil {
			return err
		}
		if err = r.BumpRevision(p.ID); err != nil {
			return err
		}
		if err = logTask(tr, p, actorID, "task.moved", id, map[string]any{"from": task.Status, "to": req.Status, "before_task_id": req.BeforeTaskID}); err != nil {
			return err
		}
		result, err = taskView(r, id, actorID)
		return err
	})
	return result, err
}
func (s *TaskService) Delete(ctx context.Context, id, actorID, version uint64) error {
	if version == 0 {
		return Fail(400, "VALIDATION_ERROR", "必须提供任务版本")
	}
	return s.inTask(ctx, id, actorID, true, func(tr *repository.TeamRepository, r *repository.TaskRepository, p *model.Project, m *model.TeamMember, task *model.Task) error {
		if task.CreatedBy != actorID && !manager(m.Role) {
			return Fail(403, "FORBIDDEN", "仅创建者或团队管理员可删除任务")
		}
		if task.Version != version {
			return versionConflict()
		}
		rows, err := r.Ordered(p.ID, task.Status)
		if err != nil {
			return err
		}
		ok, err := r.Update(id, version, map[string]any{"deleted_at": time.Now().UTC()})
		if err != nil {
			return err
		}
		if !ok {
			return versionConflict()
		}
		if err = r.Reorder(p.ID, task.Status, withoutTask(rows, id)); err != nil {
			return err
		}
		if err = r.BumpRevision(p.ID); err != nil {
			return err
		}
		return logTask(tr, p, actorID, "task.deleted", id, map[string]any{"title": task.Title})
	})
}
func taskPage(rows []repository.TaskRecord, total int64, p dto.Pagination, actorID uint64, revision *uint64) dto.TaskPage {
	items := []dto.Task{}
	today := taskToday()
	for _, row := range rows {
		items = append(items, taskFrom(row, actorID, today))
	}
	return dto.TaskPage{Page: dto.Page[dto.Task]{Items: items, Total: total, Page: p.Page, PageSize: p.PageSize}, Revision: revision}
}
func (s *TaskService) List(ctx context.Context, projectID, actorID uint64, f dto.TaskFilter, pagination dto.Pagination) (dto.TaskPage, error) {
	var result dto.TaskPage
	if err := validateTaskFilter(f); err != nil {
		return result, err
	}
	err := s.Projects.inProjectAccess(ctx, projectID, actorID, false, false, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		if f.Revision != nil && *f.Revision != p.TaskRevision {
			return Fail(409, "BOARD_CHANGED", "看板已变化，请重新加载")
		}
		rows, total, err := (&repository.TaskRepository{DB: tr.DB}).List(actorID, p.ID, false, f, pagination, taskToday())
		if err != nil {
			return err
		}
		result = taskPage(rows, total, pagination, actorID, &p.TaskRevision)
		return nil
	})
	return result, taskError(err)
}
func (s *TaskService) Mine(ctx context.Context, actorID uint64, f dto.TaskFilter, pagination dto.Pagination) (dto.TaskPage, error) {
	var result dto.TaskPage
	if err := validateTaskFilter(f); err != nil {
		return result, err
	}
	err := s.Projects.Teams.Repo.Transaction(ctx, func(tr *repository.TeamRepository) error {
		rows, total, err := (&repository.TaskRepository{DB: tr.DB}).List(actorID, 0, true, f, pagination, taskToday())
		if err != nil {
			return err
		}
		result = taskPage(rows, total, pagination, actorID, nil)
		return nil
	})
	return result, taskError(err)
}
func (s *TaskService) Board(ctx context.Context, projectID, actorID uint64, f dto.TaskFilter) (dto.Board, error) {
	result := dto.Board{Columns: []dto.BoardColumn{}}
	if err := validateTaskFilter(f); err != nil {
		return result, err
	}
	err := s.Projects.inProjectAccess(ctx, projectID, actorID, false, false, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		var err error
		result.Project, err = projectView(tr, t, m, p)
		if err != nil {
			return err
		}
		result.Revision = p.TaskRevision
		r := &repository.TaskRepository{DB: tr.DB}
		for _, status := range taskStatuses {
			column := dto.BoardColumn{Status: status, Items: []dto.Task{}, PageSize: 50}
			if f.Status == "" || f.Status == status {
				filter := f
				filter.Status = status
				page := dto.Pagination{Page: 1, PageSize: 50}
				rows, total, err := r.List(actorID, p.ID, false, filter, page, taskToday())
				if err != nil {
					return err
				}
				column.Items = taskPage(rows, total, page, actorID, nil).Items
				column.Total = total
				column.HasMore = int64(len(column.Items)) < total
			}
			result.Columns = append(result.Columns, column)
		}
		return nil
	})
	return result, taskError(err)
}

// ParseTaskAssigneeFilter is shared by HTTP parsing and validates numeric IDs before SQL queries.
func ParseTaskAssigneeFilter(raw string) (*uint64, bool, error) {
	if raw == "" {
		return nil, false, nil
	}
	if raw == "unassigned" {
		return nil, true, nil
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return nil, false, Fail(400, "VALIDATION_ERROR", "负责人筛选无效")
	}
	return &id, false, nil
}
func TrimTaskKeyword(raw string) string { return strings.TrimSpace(raw) }
