package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	mysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"time"
	"unicode/utf8"
)

type TeamService struct{ Repo *repository.TeamRepository }

func teamError(err error) error {
	if err == nil {
		return nil
	}
	var app *AppError
	if errors.As(err, &app) {
		return app
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Fail(404, "NOT_FOUND", "团队或成员不存在，或你没有访问权限")
	}
	var dbErr *mysql.MySQLError
	if errors.As(err, &dbErr) && (dbErr.Number == 1213 || dbErr.Number == 1205) {
		return Fail(409, "CONFLICT", "操作冲突，请刷新后重试")
	}
	return unavailable()
}
func boundedText(value, label string, min, max int) (string, error) {
	value = strings.TrimSpace(value)
	length := utf8.RuneCountInString(value)
	if !utf8.ValidString(value) || length < min || length > max {
		return "", Fail(400, "VALIDATION_ERROR", label+"长度不符合要求")
	}
	return value, nil
}
func manager(role string) bool { return role == "owner" || role == "admin" }
func ownerOnly(role string) error {
	if role != "owner" {
		return Fail(403, "FORBIDDEN", "仅团队所有者可执行此操作")
	}
	return nil
}
func managerOnly(role string) error {
	if !manager(role) {
		return Fail(403, "FORBIDDEN", "需要团队管理员权限")
	}
	return nil
}

// All team writes lock the team first, then re-read membership. Permission changes cannot race past this boundary.
func (s *TeamService) inTeam(ctx context.Context, teamID, actorID uint64, write bool, fn func(*repository.TeamRepository, *model.Team, *model.TeamMember) error) error {
	return teamError(s.Repo.Transaction(ctx, func(r *repository.TeamRepository) error {
		t, err := r.Team(teamID, write)
		if err != nil {
			return err
		}
		m, err := r.Member(teamID, actorID)
		if err != nil {
			return err
		}
		if write && t.ArchivedAt != nil {
			return Fail(409, "TEAM_ARCHIVED", "团队已归档，当前仅可查看")
		}
		return fn(r, t, m)
	}))
}
func teamView(r *repository.TeamRepository, t *model.Team, role string) (dto.Team, error) {
	count, err := r.CountMembers(t.ID)
	return dto.TeamFrom(t, role, count), err
}
func (s *TeamService) Create(ctx context.Context, actorID uint64, req dto.CreateTeamRequest) (dto.Team, error) {
	var result dto.Team
	name, err := boundedText(req.Name, "团队名称", 1, 80)
	if err != nil {
		return result, err
	}
	description, err := boundedText(req.Description, "团队简介", 0, 2000)
	if err != nil {
		return result, err
	}
	err = s.Repo.Transaction(ctx, func(r *repository.TeamRepository) error {
		t := &model.Team{Name: name, Description: description, OwnerID: actorID}
		if err := r.CreateTeam(t); err != nil {
			return err
		}
		if err := r.AddMember(t.ID, actorID, "owner"); err != nil {
			return err
		}
		if err := r.Log(t.ID, actorID, "team.created", "team", t.ID, map[string]any{"name": name}); err != nil {
			return err
		}
		result = dto.TeamFrom(t, "owner", 1)
		return nil
	})
	return result, teamError(err)
}
func (s *TeamService) List(ctx context.Context, actorID uint64, status string, p dto.Pagination) (dto.Page[dto.Team], error) {
	result := dto.Page[dto.Team]{Items: []dto.Team{}, Page: p.Page, PageSize: p.PageSize}
	if status != "all" && status != "active" && status != "archived" {
		return result, Fail(400, "VALIDATION_ERROR", "status 应为 all、active 或 archived")
	}
	err := s.Repo.Transaction(ctx, func(r *repository.TeamRepository) error {
		rows, total, err := r.ListTeams(actorID, status, p)
		if err != nil {
			return err
		}
		result.Total = total
		for _, row := range rows {
			result.Items = append(result.Items, dto.TeamFrom(&row.Team, row.MyRole, row.MemberCount))
		}
		return nil
	})
	return result, teamError(err)
}
func (s *TeamService) Get(ctx context.Context, teamID, actorID uint64) (dto.Team, error) {
	var result dto.Team
	err := s.inTeam(ctx, teamID, actorID, false, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		var err error
		result, err = teamView(r, t, m.Role)
		return err
	})
	return result, err
}
func (s *TeamService) Update(ctx context.Context, teamID, actorID uint64, req dto.UpdateTeamRequest) (dto.Team, error) {
	var result dto.Team
	values := map[string]any{}
	if req.Name != nil {
		name, err := boundedText(*req.Name, "团队名称", 1, 80)
		if err != nil {
			return result, err
		}
		values["name"] = name
	}
	if req.Description != nil {
		description, err := boundedText(*req.Description, "团队简介", 0, 2000)
		if err != nil {
			return result, err
		}
		values["description"] = description
	}
	if len(values) == 0 {
		return result, Fail(400, "VALIDATION_ERROR", "请提供团队名称或简介")
	}
	err := s.inTeam(ctx, teamID, actorID, true, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := managerOnly(m.Role); err != nil {
			return err
		}
		if err := r.UpdateTeam(teamID, values); err != nil {
			return err
		}
		if err := r.Log(teamID, actorID, "team.updated", "team", teamID, values); err != nil {
			return err
		}
		fresh, err := r.Team(teamID, false)
		if err != nil {
			return err
		}
		result, err = teamView(r, fresh, m.Role)
		return err
	})
	return result, err
}
func (s *TeamService) Members(ctx context.Context, teamID, actorID uint64, p dto.Pagination) (dto.Page[dto.Member], error) {
	result := dto.Page[dto.Member]{Items: []dto.Member{}, Page: p.Page, PageSize: p.PageSize}
	err := s.inTeam(ctx, teamID, actorID, false, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		rows, total, err := r.ListMembers(teamID, p)
		if err != nil {
			return err
		}
		result.Total = total
		for _, row := range rows {
			result.Items = append(result.Items, dto.Member{UserID: strconv.FormatUint(row.UserID, 10), Nickname: row.Nickname, Email: row.Email, Role: row.Role, Status: row.Status, JoinedAt: row.JoinedAt.UTC()})
		}
		return nil
	})
	return result, err
}
func (s *TeamService) SetRole(ctx context.Context, teamID, actorID, targetID uint64, role string) error {
	if role != "admin" && role != "member" {
		return Fail(400, "VALIDATION_ERROR", "角色只能为 admin 或 member")
	}
	return s.inTeam(ctx, teamID, actorID, true, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := ownerOnly(m.Role); err != nil {
			return err
		}
		target, err := r.Member(teamID, targetID)
		if err != nil {
			return err
		}
		if target.Role == "owner" {
			return Fail(409, "OWNER_REQUIRED", "不能直接修改所有者角色，请转让所有权")
		}
		if target.Role == role {
			return nil
		}
		if err = r.SetRole(teamID, targetID, role); err != nil {
			return err
		}
		return r.Log(teamID, actorID, "member.role_changed", "user", targetID, map[string]any{"from": target.Role, "to": role})
	})
}
func (s *TeamService) Remove(ctx context.Context, teamID, actorID, targetID uint64) error {
	return s.inTeam(ctx, teamID, actorID, true, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := managerOnly(m.Role); err != nil {
			return err
		}
		target, err := r.Member(teamID, targetID)
		if err != nil {
			return err
		}
		if target.Role == "owner" {
			return Fail(409, "OWNER_REQUIRED", "不能移除团队所有者")
		}
		if actorID == targetID {
			return Fail(400, "USE_LEAVE", "请使用退出团队操作")
		}
		if m.Role == "admin" && target.Role != "member" {
			return Fail(403, "FORBIDDEN", "管理员不能移除其他管理员")
		}
		if err = r.RemoveMember(teamID, targetID, actorID, "member_removed"); err != nil {
			return err
		}
		return r.Log(teamID, actorID, "member.removed", "user", targetID, map[string]any{"role": target.Role})
	})
}
func (s *TeamService) Leave(ctx context.Context, teamID, actorID uint64) error {
	return s.inTeam(ctx, teamID, actorID, true, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if m.Role == "owner" {
			return Fail(409, "OWNER_REQUIRED", "请先转让所有权，再退出团队")
		}
		if err := r.RemoveMember(teamID, actorID, actorID, "member_left"); err != nil {
			return err
		}
		return r.Log(teamID, actorID, "member.left", "user", actorID, map[string]any{"role": m.Role})
	})
}
func (s *TeamService) Transfer(ctx context.Context, teamID, actorID, targetID uint64) (dto.Team, error) {
	var result dto.Team
	err := s.inTeam(ctx, teamID, actorID, true, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := ownerOnly(m.Role); err != nil {
			return err
		}
		if targetID == actorID {
			return Fail(400, "VALIDATION_ERROR", "请选择其他团队成员")
		}
		if _, err := r.Member(teamID, targetID); err != nil {
			return err
		}
		u, err := r.User(targetID)
		if err != nil {
			return err
		}
		if u.Status != "active" {
			return Fail(409, "USER_DISABLED", "不能转让给已禁用的用户")
		}
		if err = r.SetRole(teamID, actorID, "admin"); err != nil {
			return err
		}
		if err = r.SetRole(teamID, targetID, "owner"); err != nil {
			return err
		}
		if err = r.UpdateTeam(teamID, map[string]any{"owner_id": targetID}); err != nil {
			return err
		}
		if err = r.Log(teamID, actorID, "team.ownership_transferred", "team", teamID, map[string]any{"previous_owner_id": strconv.FormatUint(actorID, 10), "new_owner_id": strconv.FormatUint(targetID, 10)}); err != nil {
			return err
		}
		fresh, err := r.Team(teamID, false)
		if err != nil {
			return err
		}
		result, err = teamView(r, fresh, "admin")
		return err
	})
	return result, err
}
func (s *TeamService) Archive(ctx context.Context, teamID, actorID uint64) error {
	return s.inTeam(ctx, teamID, actorID, true, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := ownerOnly(m.Role); err != nil {
			return err
		}
		if err := r.UpdateTeam(teamID, map[string]any{"archived_at": time.Now().UTC()}); err != nil {
			return err
		}
		return r.Log(teamID, actorID, "team.archived", "team", teamID, map[string]any{"name": t.Name})
	})
}
func (s *TeamService) CreateInvite(ctx context.Context, teamID, actorID uint64, req dto.CreateInviteRequest) (dto.InviteCreated, error) {
	var result dto.InviteCreated
	days, uses := 7, 20
	if req.ExpiresInDays != nil {
		days = *req.ExpiresInDays
	}
	if req.MaxUses != nil {
		uses = *req.MaxUses
	}
	if days < 1 || days > 30 || uses < 1 || uses > 100 {
		return result, Fail(400, "VALIDATION_ERROR", "有效期为 1 至 30 天，使用次数为 1 至 100 次")
	}
	err := s.inTeam(ctx, teamID, actorID, true, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := managerOnly(m.Role); err != nil {
			return err
		}
		var raw [32]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return err
		}
		code := hex.EncodeToString(raw[:])
		hash := sha256.Sum256([]byte(code))
		invite := &model.TeamInvite{TeamID: teamID, TokenHash: hex.EncodeToString(hash[:]), ExpiresAt: time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour), MaxUses: uses, CreatedBy: actorID}
		if err := r.CreateInvite(invite); err != nil {
			return err
		}
		if err := r.Log(teamID, actorID, "invite.created", "invite", invite.ID, map[string]any{"max_uses": uses, "expires_at": invite.ExpiresAt}); err != nil {
			return err
		}
		result = dto.InviteCreated{Invite: dto.InviteFrom(invite), Code: code}
		return nil
	})
	return result, err
}
func (s *TeamService) Invites(ctx context.Context, teamID, actorID uint64, p dto.Pagination) (dto.Page[dto.Invite], error) {
	result := dto.Page[dto.Invite]{Items: []dto.Invite{}, Page: p.Page, PageSize: p.PageSize}
	err := s.inTeam(ctx, teamID, actorID, false, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := managerOnly(m.Role); err != nil {
			return err
		}
		rows, total, err := r.ListInvites(teamID, p)
		if err != nil {
			return err
		}
		result.Total = total
		for _, v := range rows {
			result.Items = append(result.Items, dto.InviteFrom(&v))
		}
		return nil
	})
	return result, err
}
func (s *TeamService) RevokeInvite(ctx context.Context, teamID, actorID, inviteID uint64) error {
	return s.inTeam(ctx, teamID, actorID, true, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		if err := managerOnly(m.Role); err != nil {
			return err
		}
		v, err := r.Invite(inviteID, teamID)
		if err != nil {
			return err
		}
		if v.RevokedAt != nil {
			return nil
		}
		if err = r.RevokeInvite(inviteID); err != nil {
			return err
		}
		return r.Log(teamID, actorID, "invite.revoked", "invite", inviteID, map[string]any{})
	})
}
func (s *TeamService) Join(ctx context.Context, actorID uint64, code string) (dto.JoinedTeam, error) {
	var result dto.JoinedTeam
	code = strings.TrimSpace(code)
	if len(code) != 64 {
		return result, Fail(400, "INVITE_INVALID", "邀请码无效、已过期或已用尽")
	}
	if _, err := hex.DecodeString(code); err != nil {
		return result, Fail(400, "INVITE_INVALID", "邀请码无效、已过期或已用尽")
	}
	hash := sha256.Sum256([]byte(code))
	hint, err := s.Repo.InviteByHash(ctx, hex.EncodeToString(hash[:]))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, Fail(400, "INVITE_INVALID", "邀请码无效、已过期或已用尽")
	}
	if err != nil {
		return result, teamError(err)
	}
	err = s.Repo.Transaction(ctx, func(r *repository.TeamRepository) error {
		t, err := r.Team(hint.TeamID, true)
		if err != nil {
			return err
		}
		if t.ArchivedAt != nil {
			return Fail(409, "TEAM_ARCHIVED", "团队已归档，不能加入")
		}
		if m, err := r.Member(t.ID, actorID); err == nil {
			result.Team, err = teamView(r, t, m.Role)
			result.AlreadyJoined = true
			return err
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		v, err := r.Invite(hint.ID, t.ID)
		if err != nil {
			return err
		}
		if v.RevokedAt != nil || !v.ExpiresAt.After(time.Now()) || v.UsedCount >= v.MaxUses {
			return Fail(400, "INVITE_INVALID", "邀请码无效、已过期或已用尽")
		}
		if err = r.AddMember(t.ID, actorID, "member"); err != nil {
			return err
		}
		if err = r.ConsumeInvite(v.ID); err != nil {
			return err
		}
		if err = r.Log(t.ID, actorID, "member.joined", "user", actorID, map[string]any{"invite_id": strconv.FormatUint(v.ID, 10)}); err != nil {
			return err
		}
		result.Team, err = teamView(r, t, "member")
		return err
	})
	return result, teamError(err)
}
func (s *TeamService) Activities(ctx context.Context, teamID, actorID uint64, p dto.Pagination, filters ...dto.ActivityFilter) (dto.Page[dto.Activity], error) {
	result := dto.Page[dto.Activity]{Items: []dto.Activity{}, Page: p.Page, PageSize: p.PageSize}
	var filter dto.ActivityFilter
	if len(filters) > 0 {
		filter = filters[0]
	}
	if err := ValidateActivityFilter(filter); err != nil {
		return result, err
	}
	err := s.inTeam(ctx, teamID, actorID, false, func(r *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		rows, total, err := (&repository.ActivityRepository{DB: r.DB}).List(teamID, nil, p, filter)
		if err != nil {
			return err
		}
		result.Total = total
		for _, v := range rows {
			result.Items = append(result.Items, dto.ActivityFrom(&v.ActivityLog, v.ActorNickname))
		}
		return nil
	})
	return result, err
}
