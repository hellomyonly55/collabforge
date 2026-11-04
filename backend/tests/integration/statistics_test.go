package integration

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"taskflow/internal/dto"
	"taskflow/internal/model"
)

func getStatistics(t *testing.T, f *authFixture, a teamActor, path string) dto.Statistics {
	t.Helper()
	w := f.call("GET", path, "", a.Cookie)
	expect(t, w, 200)
	data := dataOf[dto.Statistics](t, w.Body.Bytes())
	assertStatisticsConsistent(t, data)
	return data
}
func assertStatisticsConsistent(t *testing.T, s dto.Statistics) {
	t.Helper()
	var statuses, priorities, assignees, completed, overdue int64
	for _, g := range s.ByStatus {
		statuses += g.Count
	}
	for _, g := range s.ByPriority {
		priorities += g.Count
	}
	for _, g := range s.ByAssignee {
		assignees += g.Total
		completed += g.Completed
		overdue += g.Overdue
	}
	if statuses != s.Total || priorities != s.Total || assignees != s.Total || completed != s.Completed || overdue != s.Overdue || s.Completed+s.Uncompleted != s.Total || s.Overdue > s.Uncompleted || len(s.ByStatus) != 4 || len(s.ByPriority) != 4 {
		t.Fatalf("inconsistent aggregates: %+v", s)
	}
}
func TestStatisticsAggregationAndTaskChanges(t *testing.T) {
	f := setupAuth(t)
	owner, member := actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, member, invite.Code)
	p := createProject(t, f, owner, team.ID)
	path := "/projects/" + p.ID + "/statistics"
	empty := getStatistics(t, f, member, path)
	if empty.Total != 0 || empty.CompletionRate != 0 || len(empty.ByAssignee) != 1 || empty.ByAssignee[0].UserID != nil || empty.ProjectID == nil || *empty.ProjectID != p.ID || empty.Scope != "project" || empty.Timezone != "Asia/Shanghai" {
		t.Fatal("empty statistics invalid")
	}
	today, err := time.Parse(time.DateOnly, empty.Today)
	if err != nil {
		t.Fatal(err)
	}
	past := today.AddDate(0, 0, -1).Format(time.DateOnly)
	first := createTask(t, f, owner, p.ID, map[string]any{"title": "逾期待办", "priority": "low", "assignee_id": strconv.FormatUint(member.ID, 10), "due_at": past})
	second := createTask(t, f, owner, p.ID, map[string]any{"title": "今天进行中", "priority": "medium", "assignee_id": strconv.FormatUint(member.ID, 10), "due_at": empty.Today})
	second = moveTask(t, f, owner, second, "in_progress", nil)
	third := createTask(t, f, owner, p.ID, map[string]any{"title": "无日期待验收", "priority": "high"})
	third = moveTask(t, f, owner, third, "in_review", nil)
	fourth := createTask(t, f, owner, p.ID, map[string]any{"title": "过去截止已完成", "priority": "urgent", "assignee_id": strconv.FormatUint(owner.ID, 10), "due_at": past})
	fourth = moveTask(t, f, owner, fourth, "done", nil)
	deleted := createTask(t, f, owner, p.ID, nil)
	expect(t, f.call("DELETE", "/tasks/"+deleted.ID+"?version=1", "", owner.Cookie), 204)
	s := getStatistics(t, f, member, path)
	if s.Total != 4 || s.Completed != 1 || s.Uncompleted != 3 || s.Overdue != 1 || s.CompletionRate != 25 || s.Name != p.Name || s.Archived {
		t.Fatalf("wrong aggregate: %+v", s)
	}
	for _, row := range s.ByStatus {
		if row.Count != 1 {
			t.Fatal("wrong status distribution")
		}
	}
	for _, row := range s.ByPriority {
		if row.Count != 1 {
			t.Fatal("wrong priority distribution")
		}
	}
	var assigned bool
	for _, row := range s.ByAssignee {
		if row.UserID != nil && *row.UserID == strconv.FormatUint(member.ID, 10) {
			assigned = true
			if row.Total != 2 || row.Completed != 0 || row.Overdue != 1 {
				t.Fatal("wrong assignee aggregate")
			}
		}
	}
	if !assigned {
		t.Fatal("assignee missing")
	}
	first = getTask(t, f, owner, first.ID)
	first = moveTask(t, f, owner, first, "done", nil)
	s = getStatistics(t, f, owner, path)
	if s.Completed != 2 || s.Overdue != 0 || s.CompletionRate != 50 {
		t.Fatal("move did not change statistics")
	}
	first = patchTask(t, f, owner, first, map[string]any{"priority": "urgent", "assignee_id": nil}, 200)
	s = getStatistics(t, f, owner, path)
	if s.ByPriority[0].Count != 0 || s.ByPriority[3].Count != 2 {
		t.Fatal("priority mutation not aggregated")
	}
	expect(t, f.call("DELETE", "/tasks/"+first.ID+"?version="+strconv.FormatUint(first.Version, 10), "", owner.Cookie), 204)
	s = getStatistics(t, f, owner, path)
	if s.Total != 3 || s.CompletionRate != 33.3 {
		t.Fatal("soft deletion or rounded rate wrong")
	}
	p2 := createProject(t, f, owner, team.ID)
	createTask(t, f, owner, p2.ID, nil)
	expect(t, f.call("POST", "/projects/"+p2.ID+"/archive", "", owner.Cookie), 204)
	combined := getStatistics(t, f, owner, "/teams/"+team.ID+"/statistics")
	if combined.Total != 4 || combined.ProjectID != nil || combined.Scope != "team" {
		t.Fatal("team should include archived project")
	}
}
func TestStatisticsAuthorizationArchiveAndMembership(t *testing.T) {
	f := setupAuth(t)
	owner, member, outside := actor(t, f), actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, member, invite.Code)
	p := createProject(t, f, owner, team.ID)
	createTask(t, f, owner, p.ID, map[string]any{"title": "分配", "assignee_id": strconv.FormatUint(member.ID, 10)})
	paths := []string{"/teams/" + team.ID + "/statistics", "/projects/" + p.ID + "/statistics"}
	f.Users.DB.Model(&model.User{}).Where("id = ?", outside.ID).Update("platform_role", "admin")
	for _, path := range paths {
		expect(t, f.call("GET", path, "", nil), 401)
		expect(t, f.call("GET", path, "", outside.Cookie), 404)
		if getStatistics(t, f, member, path).Total != 1 {
			t.Fatal("member cannot read scope")
		}
	}
	other := createTeam(t, f, outside)
	if getStatistics(t, f, outside, "/teams/"+other.ID+"/statistics").Total != 0 {
		t.Fatal("cross-team aggregate leak")
	}
	expect(t, f.call("POST", "/projects/"+p.ID+"/archive", "", owner.Cookie), 204)
	if !getStatistics(t, f, member, paths[1]).Archived {
		t.Fatal("project archive missing")
	}
	expect(t, f.call("POST", "/teams/"+team.ID+"/leave", "", member.Cookie), 204)
	for _, path := range paths {
		expect(t, f.call("GET", path, "", member.Cookie), 404)
		s := getStatistics(t, f, owner, path)
		if s.Total != 1 || len(s.ByAssignee) != 1 || s.ByAssignee[0].UserID != nil || s.ByAssignee[0].Total != 1 {
			t.Fatal("member cleanup not reflected")
		}
	}
	expect(t, f.call("POST", "/teams/"+team.ID+"/archive", "", owner.Cookie), 204)
	for _, path := range paths {
		if !getStatistics(t, f, owner, path).Archived {
			t.Fatal("team archive missing")
		}
	}
	expect(t, f.call("GET", "/teams/0/statistics", "", owner.Cookie), 400)
	expect(t, f.call("GET", "/projects/18446744073709551615/statistics", "", owner.Cookie), 404)
}
func TestStatisticsIncludesAllPages(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	team := createTeam(t, f, owner)
	p := createProject(t, f, owner, team.ID)
	for i := 0; i < 105; i++ {
		createTask(t, f, owner, p.ID, map[string]any{"title": fmt.Sprintf("完整统计 %d", i)})
	}
	for _, path := range []string{"/teams/" + team.ID + "/statistics", "/projects/" + p.ID + "/statistics"} {
		s := getStatistics(t, f, owner, path)
		if s.Total != 105 || s.ByStatus[0].Count != 105 || s.ByAssignee[0].Total != 105 {
			t.Fatal("statistics limited to list or board page")
		}
	}
}
func TestStatisticsSnapshotAndDatabaseFailure(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	team := createTeam(t, f, owner)
	p := createProject(t, f, owner, team.ID)
	path := "/projects/" + p.ID + "/statistics"
	var once sync.Once
	status := 0
	name := "test:statistics_interleaved_write"
	if err := f.Users.DB.Callback().Row().After("gorm:row").Register(name, func(tx *gorm.DB) {
		if strings.Contains(tx.Statement.SQL.String(), "COUNT(*) AS total") {
			once.Do(func() {
				status = f.call("POST", "/projects/"+p.ID+"/tasks", `{"title":"聚合之间的真实写入"}`, owner.Cookie).Code
			})
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Users.DB.Callback().Row().Remove(name) })
	first := getStatistics(t, f, owner, path)
	if status != 201 || first.Total != 0 {
		t.Fatalf("snapshot changed or writer failed: %d %+v", status, first)
	}
	f.Users.DB.Callback().Row().Remove(name)
	if getStatistics(t, f, owner, path).Total != 1 {
		t.Fatal("next snapshot did not see write")
	}
	failure := "test:statistics_database_failure"
	if err := f.Users.DB.Callback().Row().Before("gorm:row").Register(failure, func(tx *gorm.DB) {
		if tx.Statement.Table == "tasks" {
			tx.AddError(errors.New("private SQL connection details"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Users.DB.Callback().Row().Remove(failure) })
	w := f.call("GET", path, "", owner.Cookie)
	expect(t, w, 503)
	if strings.Contains(w.Body.String(), "private SQL") || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatal("database failure leaked or returned false data")
	}
}
