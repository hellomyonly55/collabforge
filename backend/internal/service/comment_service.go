package service

import (
	"context"
	"strconv"
	"time"

	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/internal/repository"
)

type CommentService struct {
	Tasks *TaskService
	Repo  *repository.CommentRepository
}

func commentDTO(row repository.CommentRecord, actorID uint64, role string, readonly bool) dto.Comment {
	return dto.Comment{
		ID:             strconv.FormatUint(row.ID, 10),
		TaskID:         strconv.FormatUint(row.TaskID, 10),
		AuthorID:       strconv.FormatUint(row.AuthorID, 10),
		AuthorNickname: row.AuthorNickname,
		Content:        row.Content,
		Version:        row.Version,
		CanEdit:        !readonly && row.AuthorID == actorID,
		CanDelete:      !readonly && (row.AuthorID == actorID || manager(role)),
		CreatedAt:      row.CreatedAt.UTC(),
		UpdatedAt:      row.UpdatedAt.UTC(),
	}
}
func commentConflict() error {
	return Fail(409, "COMMENT_VERSION_CONFLICT", "评论已被其他操作修改，请刷新后重试")
}
func commentError(err error) error {
	err = taskError(err)
	if app, ok := err.(*AppError); ok && app.Status == 404 {
		return Fail(404, "NOT_FOUND", "评论不存在或你没有访问权限")
	}
	return err
}

// inTask locks team -> project -> task, sharing the archive/member boundary with task mutations.
func (s *CommentService) inComment(ctx context.Context, id, actorID uint64, fn func(*repository.TeamRepository, *repository.CommentRepository, *model.Project, *model.TeamMember, *model.Task, *model.Comment) error) error {
	hint, err := s.Repo.Find(ctx, id)
	if err != nil {
		return commentError(err)
	}
	err = s.Tasks.inTask(ctx, hint.TaskID, actorID, true, func(tr *repository.TeamRepository, _ *repository.TaskRepository, p *model.Project, m *model.TeamMember, t *model.Task) error {
		r := &repository.CommentRepository{DB: tr.DB}
		c, err := r.Get(id, t.ID, true)
		if err != nil {
			return err
		}
		return fn(tr, r, p, m, t, c)
	})
	return commentError(err)
}
func commentLog(tr *repository.TeamRepository, p *model.Project, t *model.Task, actorID, id uint64, action string) error {
	return (&repository.ActivityRepository{DB: tr.DB}).Create(p.TeamID, &p.ID, actorID, action, "comment", id, map[string]any{"task_id": strconv.FormatUint(t.ID, 10), "task_title": t.Title})
}
func (s *CommentService) List(ctx context.Context, taskID, actorID uint64, pagination dto.Pagination) (dto.Page[dto.Comment], error) {
	result := dto.Page[dto.Comment]{Items: []dto.Comment{}, Page: pagination.Page, PageSize: pagination.PageSize}
	err := s.Tasks.inTask(ctx, taskID, actorID, false, func(tr *repository.TeamRepository, _ *repository.TaskRepository, p *model.Project, m *model.TeamMember, t *model.Task) error {
		rows, total, err := (&repository.CommentRepository{DB: tr.DB}).List(t.ID, pagination)
		if err != nil {
			return err
		}
		team, err := tr.Team(p.TeamID, false)
		if err != nil {
			return err
		}
		readonly := p.ArchivedAt != nil || team.ArchivedAt != nil
		result.Total = total
		for _, row := range rows {
			result.Items = append(result.Items, commentDTO(row, actorID, m.Role, readonly))
		}
		return nil
	})
	return result, err
}
func (s *CommentService) Create(ctx context.Context, taskID, actorID uint64, req dto.CreateCommentRequest) (dto.Comment, error) {
	var result dto.Comment
	content, err := boundedText(req.Content, "评论内容", 1, 2000)
	if err != nil {
		return result, err
	}
	err = s.Tasks.inTask(ctx, taskID, actorID, true, func(tr *repository.TeamRepository, _ *repository.TaskRepository, p *model.Project, m *model.TeamMember, t *model.Task) error {
		r := &repository.CommentRepository{DB: tr.DB}
		c := &model.Comment{TaskID: t.ID, AuthorID: actorID, Content: content, Version: 1}
		if err := r.Create(c); err != nil {
			return err
		}
		if err := commentLog(tr, p, t, actorID, c.ID, "comment.created"); err != nil {
			return err
		}
		row, err := r.View(c.ID)
		if err != nil {
			return err
		}
		result = commentDTO(row, actorID, m.Role, false)
		return nil
	})
	return result, err
}
func (s *CommentService) Update(ctx context.Context, id, actorID uint64, req dto.UpdateCommentRequest) (dto.Comment, error) {
	var result dto.Comment
	content, err := boundedText(req.Content, "评论内容", 1, 2000)
	if err != nil {
		return result, err
	}
	if req.Version == 0 {
		return result, Fail(400, "VALIDATION_ERROR", "必须提供评论版本")
	}
	err = s.inComment(ctx, id, actorID, func(tr *repository.TeamRepository, r *repository.CommentRepository, p *model.Project, m *model.TeamMember, t *model.Task, c *model.Comment) error {
		if c.AuthorID != actorID {
			return Fail(403, "FORBIDDEN", "仅作者可编辑评论")
		}
		if c.Version != req.Version {
			return commentConflict()
		}
		ok, err := r.Update(id, req.Version, map[string]any{"content": content})
		if err != nil {
			return err
		}
		if !ok {
			return commentConflict()
		}
		if err = commentLog(tr, p, t, actorID, id, "comment.updated"); err != nil {
			return err
		}
		row, err := r.View(id)
		if err != nil {
			return err
		}
		result = commentDTO(row, actorID, m.Role, false)
		return nil
	})
	return result, err
}
func (s *CommentService) Delete(ctx context.Context, id, actorID, version uint64) error {
	if version == 0 {
		return Fail(400, "VALIDATION_ERROR", "必须提供评论版本")
	}
	return s.inComment(ctx, id, actorID, func(tr *repository.TeamRepository, r *repository.CommentRepository, p *model.Project, m *model.TeamMember, t *model.Task, c *model.Comment) error {
		if c.AuthorID != actorID && !manager(m.Role) {
			return Fail(403, "FORBIDDEN", "仅作者或团队管理员可删除评论")
		}
		if c.Version != version {
			return commentConflict()
		}
		ok, err := r.Update(id, version, map[string]any{"deleted_at": time.Now().UTC()})
		if err != nil {
			return err
		}
		if !ok {
			return commentConflict()
		}
		return commentLog(tr, p, t, actorID, id, "comment.deleted")
	})
}
