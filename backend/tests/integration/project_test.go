package integration

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"sync"
	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"taskflow/migrations"
	"testing"
	"time"
)

func createProject(t *testing.T, f *authFixture, a teamActor, teamID string) dto.Project {
	t.Helper()
	w := f.call("POST", "/teams/"+teamID+"/projects", fmt.Sprintf(`{"name":%q,"description":"真实项目介绍"}`, "项目-"+unique()), a.Cookie)
	expect(t, w, 201)
	return dataOf[dto.Project](t, w.Body.Bytes())
}
func promote(t *testing.T, f *authFixture, owner, target teamActor, teamID string) {
	t.Helper()
	expect(t, f.call("PATCH", "/teams/"+teamID+"/members/"+strconv.FormatUint(target.ID, 10), `{"role":"admin"}`, owner.Cookie), 204)
}
func TestProjectLifecycleAndIsolation(t *testing.T) {
	f := setupAuth(t)
	owner, admin, member, outsider := actor(t, f), actor(t, f), actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, admin, invite.Code)
	join(t, f, member, invite.Code)
	promote(t, f, owner, admin, team.ID)
	project := createProject(t, f, owner, team.ID)
	base := "/projects/" + project.ID
	list := "/teams/" + team.ID + "/projects"
	if project.TeamID != team.ID || project.MyRole != "owner" || project.CreatedBy != strconv.FormatUint(owner.ID, 10) || project.TeamArchivedAt != nil || project.ArchivedAt != nil {
		t.Fatal("project DTO invalid")
	}
	expect(t, f.call("GET", base, "", nil), 401)
	w := f.call("GET", base, "", member.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Project](t, w.Body.Bytes()).MyRole != "member" {
		t.Fatal("team membership not inherited")
	}
	w = f.call("GET", list, "", member.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Project]](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("member cannot see team project")
	}
	expect(t, f.call("POST", list, `{"name":"越权创建"}`, member.Cookie), 403)
	expect(t, f.call("PATCH", base, `{"name":"越权修改"}`, member.Cookie), 403)
	expect(t, f.call("POST", base+"/archive", "", member.Cookie), 403)
	second := createProject(t, f, admin, team.ID)
	expect(t, f.call("PATCH", base, `{"name":"管理员更新的项目"}`, admin.Cookie), 200)
	w = f.call("PATCH", base, `{"description":""}`, owner.Cookie)
	expect(t, w, 200)
	updated := dataOf[dto.Project](t, w.Body.Bytes())
	if updated.Name != "管理员更新的项目" || updated.Description != "" {
		t.Fatal("partial update did not preserve fields")
	}
	expect(t, f.call("PATCH", base, `{"team_id":"999","name":"换团队"}`, owner.Cookie), 400)
	expect(t, f.call("PATCH", base, `{"created_by":"999"}`, owner.Cookie), 400)
	expect(t, f.call("PATCH", base, `{}`, owner.Cookie), 400)
	expect(t, f.call("POST", list, fmt.Sprintf(`{"name":%q}`, strings.Repeat("汉", 101)), owner.Cookie), 400)
	expect(t, f.call("POST", list, `{"name":"  "}`, owner.Cookie), 400)
	other := createTeam(t, f, outsider)
	if err := f.Users.DB.Model(&model.User{}).Where("id = ?", outsider.ID).Update("platform_role", "admin").Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, path, body string }{{"GET", list, ""}, {"GET", base, ""}, {"PATCH", base, `{"name":"跨团队修改"}`}, {"POST", base + "/archive", ""}, {"GET", base + "/activities", ""}} {
		expect(t, f.call(tc.method, tc.path, tc.body, outsider.Cookie), 404)
	}
	w = f.call("GET", "/teams/"+other.ID+"/projects", "", outsider.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Project]](t, w.Body.Bytes()).Total != 0 {
		t.Fatal("other team list leaked project")
	}
	w = f.call("GET", base+"/activities", "", member.Cookie)
	expect(t, w, 200)
	logs := dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes())
	if logs.Total != 3 {
		t.Fatalf("project log total=%d", logs.Total)
	}
	for _, event := range logs.Items {
		if event.ProjectID == nil || *event.ProjectID != project.ID || event.TeamID != team.ID {
			t.Fatal("project log has wrong scope")
		}
	}
	w = f.call("GET", "/teams/"+team.ID+"/activities", "", member.Cookie)
	expect(t, w, 200)
	events := dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes())
	foundProject, foundTeam := false, false
	for _, event := range events.Items {
		if event.ProjectID != nil {
			foundProject = true
		}
		if event.Action == "team.created" && event.ProjectID == nil {
			foundTeam = true
		}
	}
	if !foundProject || !foundTeam {
		t.Fatal("team activity mapping lost project or team scope")
	}
	expect(t, f.call("POST", base+"/archive", "", admin.Cookie), 204)
	expect(t, f.call("GET", base, "", member.Cookie), 200)
	expect(t, f.call("PATCH", base, `{"name":"归档后修改"}`, owner.Cookie), 409)
	expect(t, f.call("POST", base+"/archive", "", owner.Cookie), 409)
	w = f.call("GET", list+"?status=archived", "", member.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Project]](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("archived project filter invalid")
	}
	w = f.call("GET", list+"?status=active", "", member.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Project]](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("active project filter invalid")
	}
	expect(t, f.call("POST", "/teams/"+team.ID+"/archive", "", owner.Cookie), 204)
	w = f.call("GET", "/projects/"+second.ID, "", member.Cookie)
	expect(t, w, 200)
	p := dataOf[dto.Project](t, w.Body.Bytes())
	if p.TeamArchivedAt == nil || p.ArchivedAt != nil {
		t.Fatal("parent archive state missing or project archive mutated")
	}
	expect(t, f.call("POST", list, `{"name":"归档团队新项目"}`, owner.Cookie), 409)
	expect(t, f.call("PATCH", "/projects/"+second.ID, `{"name":"归档团队修改"}`, admin.Cookie), 409)
	expect(t, f.call("POST", "/projects/"+second.ID+"/archive", "", owner.Cookie), 409)
	expect(t, f.call("GET", "/projects/"+second.ID+"/activities", "", member.Cookie), 200)
}

func TestProjectMembershipChangesApplyImmediately(t *testing.T) {
	f := setupAuth(t)
	owner, admin := actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, admin, invite.Code)
	promote(t, f, owner, admin, team.ID)
	project := createProject(t, f, admin, team.ID)
	base := "/projects/" + project.ID
	memberPath := "/teams/" + team.ID + "/members/" + strconv.FormatUint(admin.ID, 10)
	expect(t, f.call("PATCH", memberPath, `{"role":"member"}`, owner.Cookie), 204)
	expect(t, f.call("GET", base, "", admin.Cookie), 200)
	expect(t, f.call("PATCH", base, `{"name":"创建者不具特权"}`, admin.Cookie), 403)
	expect(t, f.call("DELETE", memberPath, "", owner.Cookie), 204)
	expect(t, f.call("GET", base, "", admin.Cookie), 404)
	expect(t, f.call("GET", base+"/activities", "", admin.Cookie), 404)
	join(t, f, admin, invite.Code)
	expect(t, f.call("GET", base, "", admin.Cookie), 200)
	expect(t, f.call("POST", base+"/archive", "", admin.Cookie), 403)
	promote(t, f, owner, admin, team.ID)
	expect(t, f.call("PATCH", base, `{"name":"再次授权后更新"}`, admin.Cookie), 200)
	expect(t, f.call("POST", "/teams/"+team.ID+"/leave", "", admin.Cookie), 204)
	expect(t, f.call("GET", base, "", admin.Cookie), 404)
}

func TestProjectWritesRollbackWithActivityFailure(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	team := createTeam(t, f, owner)
	failLogs := func() {
		if err := f.Users.DB.Callback().Create().Before("gorm:create").Register("test:project_audit_failure", func(tx *gorm.DB) {
			if tx.Statement.Table == "activity_logs" {
				tx.AddError(errors.New("injected project audit error"))
			}
		}); err != nil {
			t.Fatal(err)
		}
	}
	restore := func() { f.Users.DB.Callback().Create().Remove("test:project_audit_failure") }
	t.Cleanup(restore)
	failLogs()
	name := "rollback-project-" + unique()
	expect(t, f.call("POST", "/teams/"+team.ID+"/projects", fmt.Sprintf(`{"name":%q}`, name), owner.Cookie), 503)
	var count int64
	if err := f.Users.DB.Model(&model.Project{}).Where("name = ?", name).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("project committed despite log failure")
	}
	restore()
	project := createProject(t, f, owner, team.ID)
	before := logCount(t, f, team.ID)
	failLogs()
	expect(t, f.call("PATCH", "/projects/"+project.ID, `{"name":"不应保存"}`, owner.Cookie), 503)
	expect(t, f.call("POST", "/projects/"+project.ID+"/archive", "", owner.Cookie), 503)
	var stored model.Project
	if err := f.Users.DB.First(&stored, project.ID).Error; err != nil || stored.Name != project.Name || stored.ArchivedAt != nil {
		t.Fatal("project update or archive partially committed")
	}
	if logCount(t, f, team.ID) != before {
		t.Fatal("failed writes added success events")
	}
	restore()
	other := createTeam(t, f, owner)
	id, _ := strconv.ParseUint(project.ID, 10, 64)
	otherID, _ := strconv.ParseUint(other.ID, 10, 64)
	err := (&repository.ActivityRepository{DB: f.Users.DB}).Create(otherID, &id, owner.ID, "project.updated", "project", id, map[string]any{})
	if err == nil {
		t.Fatal("cross-team project activity accepted by database")
	}
}

func TestProjectConcurrentAuthorizationAndArchive(t *testing.T) {
	f := setupAuth(t)
	owner, admin := actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, admin, invite.Code)
	promote(t, f, owner, admin, team.ID)
	runRace := func(left, right func() int) (int, int) {
		var wg sync.WaitGroup
		var a, b int
		wg.Add(2)
		go func() { defer wg.Done(); a = left() }()
		go func() { defer wg.Done(); b = right() }()
		wg.Wait()
		return a, b
	}
	p := createProject(t, f, owner, team.ID)
	edit, remove := runRace(func() int { return f.call("PATCH", "/projects/"+p.ID, `{"name":"移除前可写"}`, admin.Cookie).Code }, func() int {
		return f.call("DELETE", "/teams/"+team.ID+"/members/"+strconv.FormatUint(admin.ID, 10), "", owner.Cookie).Code
	})
	if remove != 204 || (edit != 200 && edit != 404) {
		t.Fatalf("remove/edit race invalid: %d/%d", remove, edit)
	}
	expect(t, f.call("PATCH", "/projects/"+p.ID, `{"name":"移除后不可写"}`, admin.Cookie), 404)
	join(t, f, admin, invite.Code)
	promote(t, f, owner, admin, team.ID)
	second := createProject(t, f, owner, team.ID)
	edit, archived := runRace(func() int {
		return f.call("PATCH", "/projects/"+second.ID, `{"name":"归档前更新"}`, admin.Cookie).Code
	}, func() int { return f.call("POST", "/projects/"+second.ID+"/archive", "", owner.Cookie).Code })
	if archived != 204 || (edit != 200 && edit != 409) {
		t.Fatalf("project archive race invalid: %d/%d", archived, edit)
	}
	expect(t, f.call("PATCH", "/projects/"+second.ID, `{"name":"归档后不可写"}`, owner.Cookie), 409)
	var logs []model.ActivityLog
	if err := f.Users.DB.Where("project_id = ?", second.ID).Order("id ASC").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	seenArchive := false
	for _, event := range logs {
		if event.Action == "project.archived" {
			seenArchive = true
		}
		if seenArchive && event.Action == "project.updated" {
			t.Fatal("write committed after archive")
		}
	}
	third := createProject(t, f, owner, team.ID)
	edit, archived = runRace(func() int {
		return f.call("PATCH", "/projects/"+third.ID, `{"name":"团队归档前更新"}`, admin.Cookie).Code
	}, func() int { return f.call("POST", "/teams/"+team.ID+"/archive", "", owner.Cookie).Code })
	if archived != 204 || (edit != 200 && edit != 409) {
		t.Fatalf("team archive race invalid: %d/%d", archived, edit)
	}
	expect(t, f.call("PATCH", "/projects/"+third.ID, `{"name":"归档团队不可写"}`, owner.Cookie), 409)
}

func TestProjectPaginationAndValidation(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	team := createTeam(t, f, owner)
	list := "/teams/" + team.ID + "/projects"
	for i := 0; i < 21; i++ {
		createProject(t, f, owner, team.ID)
	}
	w := f.call("GET", list, "", owner.Cookie)
	expect(t, w, 200)
	first := dataOf[dto.Page[dto.Project]](t, w.Body.Bytes())
	if first.Total != 21 || len(first.Items) != 20 || first.PageSize != 20 {
		t.Fatal("project pagination defaults invalid")
	}
	w = f.call("GET", list+"?page=2", "", owner.Cookie)
	expect(t, w, 200)
	second := dataOf[dto.Page[dto.Project]](t, w.Body.Bytes())
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatal("project second page invalid")
	}
	expect(t, f.call("GET", list+"?page_size=101", "", owner.Cookie), 400)
	expect(t, f.call("GET", list+"?status=oops", "", owner.Cookie), 400)
	expect(t, f.call("GET", "/projects/0", "", owner.Cookie), 400)
	base := "/projects/" + first.Items[0].ID
	expect(t, f.call("PATCH", base, `{"name":"分页日志更新"}`, owner.Cookie), 200)
	w = f.call("GET", base+"/activities?page_size=1", "", owner.Cookie)
	expect(t, w, 200)
	a := dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes())
	w = f.call("GET", base+"/activities?page_size=1&page=2", "", owner.Cookie)
	expect(t, w, 200)
	b := dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes())
	if a.Total != 2 || len(a.Items) != 1 || len(b.Items) != 1 || a.Items[0].ID == b.Items[0].ID {
		t.Fatal("project activities pagination invalid")
	}
}

func TestProjectMigrationRecoversRecordedDDLCrash(t *testing.T) {
	f := setupAuth(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Simulate DDL committed but migration version not recorded, on the isolated test schema only.
	if err := f.Users.DB.Exec("DELETE FROM schema_migrations WHERE version = ?", 8).Error; err != nil {
		t.Fatal(err)
	}
	pool, err := f.Users.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err = migrations.Run(ctx, pool); err != nil {
		t.Fatal("cannot resume existing project foreign key:", err)
	}
	var count int64
	if err = f.Users.DB.Table("schema_migrations").Where("version = ?", 8).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("migration version not restored")
	}
}
