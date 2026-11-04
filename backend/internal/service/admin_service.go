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
	"taskflow/internal/infrastructure/password"
	"taskflow/internal/model"
	"taskflow/internal/repository"
)

type AdminService struct {
	Repo *repository.AdminRepository
	Auth *AuthService
}

func RequirePlatformAdmin(u *model.User) error {
	if u.PlatformRole != "admin" {
		return Fail(403, "FORBIDDEN", "需要平台管理员权限")
	}
	return nil
}
func adminUser(row repository.AdminUserRecord) dto.AdminUser {
	return dto.AdminUser{ID: strconv.FormatUint(row.ID, 10), Email: row.Email, Nickname: row.Nickname, PlatformRole: row.PlatformRole, Status: row.Status, OwnedTeamCount: row.OwnedTeamCount, CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC()}
}
func validateAdminFilter(f dto.AdminFilter, teams bool) error {
	if utf8.RuneCountInString(f.Keyword) > 100 {
		return Fail(400, "VALIDATION_ERROR", "关键词最多 100 字符")
	}
	if f.Status != "all" && f.Status != "active" && f.Status != "disabled" && (!teams || f.Status != "archived") {
		return Fail(400, "VALIDATION_ERROR", "状态筛选无效")
	}
	if teams && f.Status == "disabled" {
		return Fail(400, "VALIDATION_ERROR", "团队状态筛选无效")
	}
	if f.Role != "all" && f.Role != "admin" && f.Role != "user" {
		return Fail(400, "VALIDATION_ERROR", "角色筛选无效")
	}
	return nil
}
func (s *AdminService) read(ctx context.Context, actor *model.User, fn func(*repository.AdminRepository) error) error {
	return teamError(s.Repo.Transaction(ctx, func(r *repository.AdminRepository) error {
		u, err := (&repository.UserRepository{DB: r.DB}).ByID(ctx, actor.ID)
		if err != nil {
			return err
		}
		if u.Status != "active" || u.SessionVersion != actor.SessionVersion {
			return unauthorized()
		}
		if err = RequirePlatformAdmin(u); err != nil {
			return err
		}
		return fn(r)
	}))
}
func (s *AdminService) Users(ctx context.Context, actor *model.User, f dto.AdminFilter, p dto.Pagination) (dto.Page[dto.AdminUser], error) {
	result := dto.Page[dto.AdminUser]{Items: []dto.AdminUser{}, Page: p.Page, PageSize: p.PageSize}
	if err := validateAdminFilter(f, false); err != nil {
		return result, err
	}
	err := s.read(ctx, actor, func(r *repository.AdminRepository) error {
		rows, total, err := r.Users(f, p)
		if err != nil {
			return err
		}
		result.Total = total
		for _, row := range rows {
			result.Items = append(result.Items, adminUser(row))
		}
		return nil
	})
	return result, err
}
func (s *AdminService) Teams(ctx context.Context, actor *model.User, f dto.AdminFilter, p dto.Pagination) (dto.Page[dto.AdminTeam], error) {
	result := dto.Page[dto.AdminTeam]{Items: []dto.AdminTeam{}, Page: p.Page, PageSize: p.PageSize}
	if err := validateAdminFilter(f, true); err != nil {
		return result, err
	}
	err := s.read(ctx, actor, func(r *repository.AdminRepository) error {
		rows, total, err := r.Teams(f, p)
		if err != nil {
			return err
		}
		result.Total = total
		for _, v := range rows {
			result.Items = append(result.Items, dto.AdminTeam{ID: strconv.FormatUint(v.ID, 10), Name: v.Name, Description: v.Description, OwnerID: strconv.FormatUint(v.OwnerID, 10), OwnerNickname: v.OwnerNickname, OwnerEmail: v.OwnerEmail, MemberCount: v.MemberCount, ProjectCount: v.ProjectCount, TaskCount: v.TaskCount, ArchivedAt: v.ArchivedAt, CreatedAt: v.CreatedAt.UTC()})
		}
		return nil
	})
	return result, err
}
func (s *AdminService) Statistics(ctx context.Context, actor *model.User) (dto.AdminStatistics, error) {
	var result dto.AdminStatistics
	err := s.read(ctx, actor, func(r *repository.AdminRepository) error {
		now := time.Now().UTC()
		today := now.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format(time.DateOnly)
		var err error
		result, err = r.Statistics(today)
		result.AsOf = now
		result.Today = today
		result.Timezone = "Asia/Shanghai"
		return err
	})
	return result, err
}
func lockedAdmin(rows []model.User, actor *model.User) error {
	for _, u := range rows {
		if u.ID == actor.ID {
			if u.Status != "active" || u.SessionVersion != actor.SessionVersion {
				return unauthorized()
			}
			return RequirePlatformAdmin(&u)
		}
	}
	return unauthorized()
}
func (s *AdminService) Status(ctx context.Context, actor *model.User, targetID uint64, req dto.UserStatusRequest) (dto.AdminUser, error) {
	var result dto.AdminUser
	if req.Status != "active" && req.Status != "disabled" {
		return result, Fail(400, "VALIDATION_ERROR", "用户状态应为 active 或 disabled")
	}
	reason, err := boundedText(req.Reason, "操作原因", 1, 500)
	if err != nil {
		return result, err
	}
	if actor.ID == targetID && req.Status == "disabled" {
		return result, Fail(400, "CANNOT_DISABLE_SELF", "不能禁用自己")
	}
	changed := false
	err = s.Repo.Transaction(ctx, func(r *repository.AdminRepository) error {
		rows, err := r.Actors(actor.ID, targetID)
		if err != nil {
			return err
		}
		if err = lockedAdmin(rows, actor); err != nil {
			return err
		}
		var target *model.User
		for i := range rows {
			if rows[i].ID == targetID {
				target = &rows[i]
			}
		}
		if target == nil {
			return gorm.ErrRecordNotFound
		}
		if target.Status != req.Status {
			if err = r.DB.Model(&model.User{}).Where("id = ?", targetID).Updates(map[string]any{"status": req.Status, "session_version": gorm.Expr("session_version + 1")}).Error; err != nil {
				return err
			}
			if err = r.Audit(actor.ID, "user.status_changed", "user", targetID, map[string]any{"from": target.Status, "to": req.Status, "reason": reason}); err != nil {
				return err
			}
			changed = true
		}
		row, err := r.User(targetID)
		if err != nil {
			return err
		}
		result = adminUser(row)
		return nil
	})
	if err != nil {
		return result, teamError(err)
	}
	if changed {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err = s.Auth.Sessions.RevokeAll(cleanup, targetID); err != nil {
			s.Auth.Log.Warn("user status changed; session cleanup deferred", "user_id", targetID)
		}
	}
	return result, nil
}
func (s *AdminService) Archive(ctx context.Context, actor *model.User, teamID uint64, req dto.AdminArchiveRequest) error {
	reason, err := boundedText(req.Reason, "归档原因", 1, 500)
	if err != nil {
		return err
	}
	return teamError(s.Repo.Transaction(ctx, func(r *repository.AdminRepository) error {
		tr := &repository.TeamRepository{DB: r.DB}
		t, err := tr.Team(teamID, true)
		if err != nil {
			return err
		}
		rows, err := r.Actors(actor.ID)
		if err != nil {
			return err
		}
		if err = lockedAdmin(rows, actor); err != nil {
			return err
		}
		if t.ArchivedAt != nil {
			return Fail(409, "TEAM_ARCHIVED", "团队已归档")
		}
		if err = tr.UpdateTeam(teamID, map[string]any{"archived_at": time.Now().UTC()}); err != nil {
			return err
		}
		if err = tr.Log(teamID, actor.ID, "team.archived", "team", teamID, map[string]any{"name": t.Name, "source": "platform_admin", "reason": reason}); err != nil {
			return err
		}
		return r.Audit(actor.ID, "team.archived", "team", teamID, map[string]any{"name": t.Name, "reason": reason})
	}))
}

// BootstrapAdmin only creates a new account; existing email ownership is never changed.
func BootstrapAdmin(ctx context.Context, repo *repository.AdminRepository, req dto.RegisterRequest) (dto.User, error) {
	var result dto.User
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return result, err
	}
	name, err := nickname(req.Nickname)
	if err != nil {
		return result, err
	}
	if err = password.Validate(req.Password); err != nil {
		return result, Fail(400, "VALIDATION_ERROR", err.Error())
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		return result, Fail(500, "INTERNAL_ERROR", "密码处理失败")
	}
	err = repo.Transaction(ctx, func(r *repository.AdminRepository) error {
		u := &model.User{Email: email, Nickname: name, PasswordHash: hash, PlatformRole: "admin", Status: "active", SessionVersion: 1}
		if err := r.DB.Create(u).Error; err != nil {
			return err
		}
		if err := r.Audit(u.ID, "admin.created", "user", u.ID, map[string]any{"source": "cli"}); err != nil {
			return err
		}
		result = dto.UserFrom(u)
		return nil
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return result, Fail(409, "EMAIL_EXISTS", "该邮箱已存在，不会更改原账号权限")
	}
	return result, teamError(err)
}
func AdminKeyword(value string) string { return strings.TrimSpace(value) }
