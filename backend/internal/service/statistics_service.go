package service

import (
	"context"
	"math"
	"strconv"
	"time"

	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/internal/repository"
)

type StatisticsService struct{ Projects *ProjectService }

func completeGroups(keys []string, rows []dto.CountGroup) []dto.CountGroup {
	counts := map[string]int64{}
	for _, row := range rows {
		counts[row.Key] = row.Count
	}
	result := make([]dto.CountGroup, 0, len(keys))
	for _, key := range keys {
		result = append(result, dto.CountGroup{Key: key, Count: counts[key]})
	}
	return result
}
func completionRate(completed, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(completed)/float64(total)*1000) / 10
}
func (s *StatisticsService) aggregate(tr *repository.TeamRepository, t *model.Team, p *model.Project) (dto.Statistics, error) {
	now := time.Now().UTC()
	result := dto.Statistics{Scope: "team", TeamID: strconv.FormatUint(t.ID, 10), Name: t.Name, Archived: t.ArchivedAt != nil, AsOf: now, Today: now.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format(time.DateOnly), Timezone: "Asia/Shanghai", ByAssignee: []dto.AssigneeStatistics{}}
	var projectID *uint64
	if p != nil {
		projectID = &p.ID
		id := strconv.FormatUint(p.ID, 10)
		result.ProjectID = &id
		result.Scope = "project"
		result.Name = p.Name
		result.Archived = result.Archived || p.ArchivedAt != nil
	}
	totals, statuses, priorities, assignees, err := (&repository.StatisticsRepository{DB: tr.DB}).Aggregate(t.ID, projectID, result.Today)
	if err != nil {
		return dto.Statistics{}, err
	}
	result.Total = totals.Total
	result.Completed = totals.Completed
	result.Uncompleted = totals.Total - totals.Completed
	result.Overdue = totals.Overdue
	result.CompletionRate = completionRate(totals.Completed, totals.Total)
	result.ByStatus = completeGroups(taskStatuses, statuses)
	result.ByPriority = completeGroups([]string{"low", "medium", "high", "urgent"}, priorities)
	unassigned := false
	for _, row := range assignees {
		var id *string
		if row.UserID != nil {
			v := strconv.FormatUint(*row.UserID, 10)
			id = &v
		} else {
			unassigned = true
		}
		result.ByAssignee = append(result.ByAssignee, dto.AssigneeStatistics{UserID: id, Nickname: row.Nickname, Total: row.Total, Completed: row.Completed, Overdue: row.Overdue})
	}
	if !unassigned {
		result.ByAssignee = append(result.ByAssignee, dto.AssigneeStatistics{Nickname: "未分配"})
	}
	return result, nil
}
func (s *StatisticsService) Team(ctx context.Context, id, actorID uint64) (dto.Statistics, error) {
	var result dto.Statistics
	err := s.Projects.Teams.inTeam(ctx, id, actorID, false, func(tr *repository.TeamRepository, t *model.Team, m *model.TeamMember) error {
		var err error
		result, err = s.aggregate(tr, t, nil)
		return err
	})
	return result, err
}
func (s *StatisticsService) Project(ctx context.Context, id, actorID uint64) (dto.Statistics, error) {
	var result dto.Statistics
	err := s.Projects.inProjectAccess(ctx, id, actorID, false, false, func(tr *repository.TeamRepository, pr *repository.ProjectRepository, t *model.Team, m *model.TeamMember, p *model.Project) error {
		var err error
		result, err = s.aggregate(tr, t, p)
		return err
	})
	return result, err
}
