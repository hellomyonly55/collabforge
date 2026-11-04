package dto

import "time"

type AdminFilter struct{ Keyword, Status, Role string }
type UserStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}
type AdminArchiveRequest struct {
	Reason string `json:"reason"`
}
type AdminUser struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	Nickname       string    `json:"nickname"`
	PlatformRole   string    `json:"platform_role"`
	Status         string    `json:"status"`
	OwnedTeamCount int64     `json:"owned_team_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type AdminTeam struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	OwnerID       string     `json:"owner_id"`
	OwnerNickname string     `json:"owner_nickname"`
	OwnerEmail    string     `json:"owner_email"`
	MemberCount   int64      `json:"member_count"`
	ProjectCount  int64      `json:"project_count"`
	TaskCount     int64      `json:"task_count"`
	ArchivedAt    *time.Time `json:"archived_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
type AdminStatistics struct {
	Users            int64     `json:"users"`
	ActiveUsers      int64     `json:"active_users"`
	DisabledUsers    int64     `json:"disabled_users"`
	Admins           int64     `json:"admins"`
	Teams            int64     `json:"teams"`
	ArchivedTeams    int64     `json:"archived_teams"`
	Projects         int64     `json:"projects"`
	ArchivedProjects int64     `json:"archived_projects"`
	Tasks            int64     `json:"tasks"`
	CompletedTasks   int64     `json:"completed_tasks"`
	OverdueTasks     int64     `json:"overdue_tasks"`
	Comments         int64     `json:"comments"`
	AsOf             time.Time `json:"as_of"`
	Today            string    `json:"today"`
	Timezone         string    `json:"timezone"`
}
