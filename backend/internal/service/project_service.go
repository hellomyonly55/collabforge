package service

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"time"
)

type ProjectService struct {
	Teams *TeamService
	Repo  *repository.ProjectRepository
}

func projectError(err error) error {
	if err == nil {
		return nil
	}
	var app *AppError
	if errors.Is(err, gorm.ErrRecordNotFound) || (errors.As(err, &app) && app.Status == 404) {
		return Fail(404, "NOT_FOUND", "项目不存在或你没有访问权限")
	}
	return teamError(err)
}

// Team IDs are immutable. Resolve the parent, then lock team -> project for every write.
func (s *ProjectService) inProject(ctx context.Context, id, actorID uint64, write bool, fn func(*repository.TeamRepository, *repository.ProjectRepository, *model.Team, *model.TeamMember, *model.Project) error) error {
	return s.inProjectAccess(ctx, id, actorID, write, write, fn)
}

// Task writes share the project archive and membership boundary without requiring project-manager privileges.
func (s *ProjectService) inProjectAccess(ctx context.Context, id, actorID uint64, write, managerRequired bool, fn func(*repository.TeamRepository, *repository.ProjectRepository, *model.Team, *model.TeamMember, *model.Project) error) error {
	hint, err := s.Repo.Find(ctx, id)
	if err != nil {
		return projectError(err)
	}
	err = s.Teams.inTeam(ctx, hint.TeamID, actorID, write, func(tr *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		pr := &repository.ProjectRepository{DB: tr.DB}
		p, err := pr.Project(id, t.ID, write)
		if err != nil {
			return err
		}
		if write {
			if managerRequired {
				if err = managerOnly(m.Role); err != nil {
					return err
				}
			}
			if p.ArchivedAt != nil {
				return Fail(409, "PROJECT_ARCHIVED", "项目已归档，当前仅可查看")
			}
		}
		return fn(tr, pr, t, m, p)
	})
	return projectError(err)
}
func projectView(tr *repository.TeamRepository, t *model.Team, m *model.TeamMember, p *model.Project) (dto.Project, error) {
	u, err := tr.User(p.CreatedBy)
	if err != nil {
		return dto.Project{}, err
	}
	return dto.ProjectFrom(p, t, m.Role, u.Nickname), nil
}
func (s *ProjectService) Create(ctx context.Context, teamID, actorID uint64, req dto.CreateProjectRequest) (dto.Project, error) {
	var result dto.Project
	name, err := boundedText(req.Name, "项目名称", 1, 100)
	if err != nil {
		return result, err
	}
	description, err := boundedText(req.Description, "项目简介", 0, 2000)
	if err != nil {
		return result, err
	}
	err = s.Teams.inTeam(ctx, teamID, actorID, true, func(tr *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := managerOnly(m.Role); err != nil {
			return err
		}
		pr := &repository.ProjectRepository{DB: tr.DB}
		p := &model.Project{TeamID: teamID, Name: name, Description: description, CreatedBy: actorID}
		if err := pr.Create(p); err != nil {
			return err
		}
		if err := (&repository.ActivityRepository{DB: tr.DB}).Create(teamID, &p.ID, actorID, "project.created", "project", p.ID, map[string]any{"name": name}); err != nil {
			return err
		}
		var err error
		result, err = projectView(tr, t, m, p)
		return err
	})
	return result, err
}
func (s *ProjectService) List(ctx context.Context, teamID, actorID uint64, status string, p dto.Pagination) (dto.Page[dto.Project], error) {
	result := dto.Page[dto.Project]{Items: []dto.Project{}, Page: p.Page, PageSize: p.PageSize}
	if status != "all" && status != "active" && status != "archived" {
		return result, Fail(400, "VALIDATION_ERROR", "status 应为 all、active 或 archived")
	}
	err := s.Teams.inTeam(ctx, teamID, actorID, false, func(tr *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		rows, total, err := (&repository.ProjectRepository{DB: tr.DB}).List(teamID, status, p)
		if err != nil {
			return err
		}
		result.Total = total
		for _, row := range rows {
			result.Items = append(result.Items, dto.ProjectFrom(&row.Project, t, m.Role, row.CreatorNickname))
		}
		return nil
	})
	return result, err
}
func (s *ProjectService) Get(ctx context.Context, id, actorID uint64) (dto.Project, error) {
	var result dto.Project
	err := s.inProject(ctx, id, actorID, false, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		var err error
		result, err = projectView(tr, t, m, p)
		return err
	})
	return result, err
}
func (s *ProjectService) Update(ctx context.Context, id, actorID uint64, req dto.UpdateProjectRequest) (dto.Project, error) {
	var result dto.Project
	values := map[string]any{}
	if req.Name != nil {
		name, err := boundedText(*req.Name, "项目名称", 1, 100)
		if err != nil {
			return result, err
		}
		values["name"] = name
	}
	if req.Description != nil {
		description, err := boundedText(*req.Description, "项目简介", 0, 2000)
		if err != nil {
			return result, err
		}
		values["description"] = description
	}
	if len(values) == 0 {
		return result, Fail(400, "VALIDATION_ERROR", "请提供项目名称或简介")
	}
	err := s.inProject(ctx, id, actorID, true, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		if err := pr.Update(id, values); err != nil {
			return err
		}
		if err := (&repository.ActivityRepository{DB: tr.DB}).Create(t.ID, &id, actorID, "project.updated", "project", id, values); err != nil {
			return err
		}
		fresh, err := pr.Project(id, t.ID, false)
		if err != nil {
			return err
		}
		result, err = projectView(tr, t, m, fresh)
		return err
	})
	return result, err
}
func (s *ProjectService) Archive(ctx context.Context, id, actorID uint64) error {
	return s.inProject(ctx, id, actorID, true, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		if err := pr.Update(id, map[string]any{"archived_at": time.Now().UTC()}); err != nil {
			return err
		}
		return (&repository.ActivityRepository{DB: tr.DB}).Create(t.ID, &id, actorID, "project.archived", "project", id, map[string]any{"name": p.Name})
	})
}
func (s *ProjectService) Activities(ctx context.Context, id, actorID uint64, pagination dto.Pagination, filters ...dto.ActivityFilter) (dto.Page[dto.Activity], error) {
	result := dto.Page[dto.Activity]{Items: []dto.Activity{}, Page: pagination.Page, PageSize: pagination.PageSize}
	var filter dto.ActivityFilter
	if len(filters) > 0 {
		filter = filters[0]
	}
	if err := ValidateActivityFilter(filter); err != nil {
		return result, err
	}
	err := s.inProject(ctx, id, actorID, false, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		rows, total, err := (&repository.ActivityRepository{DB: tr.DB}).List(t.ID, &id, pagination, filter)
		if err != nil {
			return err
		}
		result.Total = total
		for _, row := range rows {
			result.Items = append(result.Items, dto.ActivityFrom(&row.ActivityLog, row.ActorNickname))
		}
		return nil
	})
	return result, err
}
