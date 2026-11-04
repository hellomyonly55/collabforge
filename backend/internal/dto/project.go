package dto

import (
	"strconv"
	"taskflow/internal/model"
	"time"
)

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}
type Project struct {
	ID              string     `json:"id"`
	TeamID          string     `json:"team_id"`
	TeamName        string     `json:"team_name"`
	MyRole          string     `json:"my_role"`
	TeamArchivedAt  *time.Time `json:"team_archived_at"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	CreatedBy       string     `json:"created_by"`
	CreatorNickname string     `json:"creator_nickname"`
	ArchivedAt      *time.Time `json:"archived_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func ProjectFrom(p *model.Project, t *model.Team, role, creator string) Project {
	return Project{ID: strconv.FormatUint(p.ID, 10), TeamID: strconv.FormatUint(p.TeamID, 10), TeamName: t.Name, MyRole: role, TeamArchivedAt: t.ArchivedAt, Name: p.Name, Description: p.Description, CreatedBy: strconv.FormatUint(p.CreatedBy, 10), CreatorNickname: creator, ArchivedAt: p.ArchivedAt, CreatedAt: p.CreatedAt.UTC(), UpdatedAt: p.UpdatedAt.UTC()}
}
