package dto

import (
	"encoding/json"
	"strconv"
	"taskflow/internal/model"
	"time"
)

type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}
type Pagination struct{ Page, PageSize int }

func (p Pagination) Offset() int { return (p.Page - 1) * p.PageSize }

type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type UpdateTeamRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}
type RoleRequest struct {
	Role string `json:"role"`
}
type TransferRequest struct {
	UserID string `json:"user_id"`
}
type JoinRequest struct {
	Code string `json:"code"`
}
type CreateInviteRequest struct {
	ExpiresInDays *int `json:"expires_in_days"`
	MaxUses       *int `json:"max_uses"`
}
type Team struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	OwnerID     string     `json:"owner_id"`
	MyRole      string     `json:"my_role"`
	MemberCount int64      `json:"member_count"`
	ArchivedAt  *time.Time `json:"archived_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func TeamFrom(t *model.Team, role string, count int64) Team {
	return Team{strconv.FormatUint(t.ID, 10), t.Name, t.Description, strconv.FormatUint(t.OwnerID, 10), role, count, t.ArchivedAt, t.CreatedAt.UTC(), t.UpdatedAt.UTC()}
}

type Member struct {
	UserID   string    `json:"user_id"`
	Nickname string    `json:"nickname"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	JoinedAt time.Time `json:"joined_at"`
}
type Invite struct {
	ID        string     `json:"id"`
	TeamID    string     `json:"team_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	MaxUses   int        `json:"max_uses"`
	UsedCount int        `json:"used_count"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedBy string     `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	Status    string     `json:"status"`
}

func InviteFrom(v *model.TeamInvite) Invite {
	status := "active"
	if v.RevokedAt != nil {
		status = "revoked"
	} else if !v.ExpiresAt.After(time.Now()) {
		status = "expired"
	} else if v.UsedCount >= v.MaxUses {
		status = "exhausted"
	}
	return Invite{strconv.FormatUint(v.ID, 10), strconv.FormatUint(v.TeamID, 10), v.ExpiresAt.UTC(), v.MaxUses, v.UsedCount, v.RevokedAt, strconv.FormatUint(v.CreatedBy, 10), v.CreatedAt.UTC(), status}
}

type InviteCreated struct {
	Invite Invite `json:"invite"`
	Code   string `json:"code"`
}
type JoinedTeam struct {
	Team          Team `json:"team"`
	AlreadyJoined bool `json:"already_joined"`
}
type Activity struct {
	ProjectID     *string         `json:"project_id"`
	ID            string          `json:"id"`
	TeamID        string          `json:"team_id"`
	ActorID       string          `json:"actor_id"`
	ActorNickname string          `json:"actor_nickname"`
	Action        string          `json:"action"`
	TargetType    string          `json:"target_type"`
	TargetID      string          `json:"target_id"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
}

func ActivityFrom(a *model.ActivityLog, nickname string) Activity {
	var projectID *string
	if a.ProjectID != nil {
		value := strconv.FormatUint(*a.ProjectID, 10)
		projectID = &value
	}
	return Activity{ID: strconv.FormatUint(a.ID, 10), TeamID: strconv.FormatUint(a.TeamID, 10), ProjectID: projectID, ActorID: strconv.FormatUint(a.ActorID, 10), ActorNickname: nickname, Action: a.Action, TargetType: a.TargetType, TargetID: strconv.FormatUint(a.TargetID, 10), Metadata: json.RawMessage(a.Metadata), CreatedAt: a.CreatedAt.UTC()}
}
