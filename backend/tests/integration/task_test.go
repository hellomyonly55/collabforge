package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/migrations"
	"testing"
	"time"
)

func createTask(t *testing.T, f *authFixture, a teamActor, projectID string, fields map[string]any) dto.Task {
	t.Helper()
	if fields == nil {
		fields = map[string]any{"title": "任务-" + unique()}
	}
	body, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	w := f.call("POST", "/projects/"+projectID+"/tasks", string(body), a.Cookie)
	expect(t, w, 201)
	return dataOf[dto.Task](t, w.Body.Bytes())
}
func getTask(t *testing.T, f *authFixture, a teamActor, id string) dto.Task {
	t.Helper()
	w := f.call("GET", "/tasks/"+id, "", a.Cookie)
	expect(t, w, 200)
	return dataOf[dto.Task](t, w.Body.Bytes())
}
func patchTask(t *testing.T, f *authFixture, a teamActor, task dto.Task, fields map[string]any, status int) dto.Task {
	t.Helper()
	fields["version"] = task.Version
	body, _ := json.Marshal(fields)
	w := f.call("PATCH", "/tasks/"+task.ID, string(body), a.Cookie)
	expect(t, w, status)
	if status == 200 {
		return dataOf[dto.Task](t, w.Body.Bytes())
	}
	return task
}
func moveTask(t *testing.T, f *authFixture, a teamActor, task dto.Task, status string, before *string) dto.Task {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"version": task.Version, "status": status, "before_task_id": before})
	w := f.call("POST", "/tasks/"+task.ID+"/move", string(body), a.Cookie)
	expect(t, w, 200)
	return dataOf[dto.Task](t, w.Body.Bytes())
}
func assertDense(t *testing.T, f *authFixture, projectID string) {
	t.Helper()
	for _, status := range []string{"todo", "in_progress", "in_review", "done"} {
		var tasks []model.Task
		if err := f.Users.DB.Where("project_id = ? AND status = ? AND deleted_at IS NULL", projectID, status).Order("position ASC,id ASC").Find(&tasks).Error; err != nil {
			t.Fatal(err)
		}
		for i, task := range tasks {
			if task.Position != int64(i) {
				t.Fatalf("non-dense order: %s at %d=%d", status, i, task.Position)
			}
		}
	}
}
func TestTaskLifecyclePermissionsAndFilters(t *testing.T) {
	f := setupAuth(t)
	owner, creator, member, outside := actor(t, f), actor(t, f), actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, creator, invite.Code)
	join(t, f, member, invite.Code)
	project := createProject(t, f, owner, team.ID)
	past := time.Now().Add(-72 * time.Hour).Format(time.DateOnly)
	task := createTask(t, f, creator, project.ID, map[string]any{"title": "真实任务", "description": "处理错误与权限", "priority": "urgent", "assignee_id": strconv.FormatUint(member.ID, 10), "due_at": past})
	base := "/tasks/" + task.ID
	list := "/projects/" + project.ID + "/tasks"
	if task.Status != "todo" || task.Version != 1 || task.Position != 0 || task.DueAt == nil || *task.DueAt != past || !task.IsOverdue || !task.CanDelete {
		t.Fatal("task defaults or date DTO invalid")
	}
	read := getTask(t, f, member, task.ID)
	if read.CanDelete || read.AssigneeID == nil || *read.AssigneeID != strconv.FormatUint(member.ID, 10) {
		t.Fatal("task deletion/assignment permission invalid")
	}
	expect(t, f.call("GET", base, "", nil), 401)
	for _, path := range []string{base, list, "/projects/" + project.ID + "/board"} {
		expect(t, f.call("GET", path, "", outside.Cookie), 404)
	}
	expect(t, f.call("DELETE", base+"?version=1", "", member.Cookie), 403)
	updated := patchTask(t, f, member, task, map[string]any{"title": "团队成员可编辑"}, 200)
	if updated.AssigneeID == nil || updated.DueAt == nil || updated.Version != 2 {
		t.Fatal("omitted PATCH fields were cleared")
	}
	for _, query := range []string{"status=todo", "priority=urgent", "overdue=true", "assignee_id=" + strconv.FormatUint(member.ID, 10), "keyword=" + url.QueryEscape("成员可编辑")} {
		w := f.call("GET", list+"?"+query, "", creator.Cookie)
		expect(t, w, 200)
		if dataOf[dto.TaskPage](t, w.Body.Bytes()).Total != 1 {
			t.Fatal("filter failed: " + query)
		}
	}
	w := f.call("GET", "/me/tasks", "", member.Cookie)
	expect(t, w, 200)
	if dataOf[dto.TaskPage](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("assigned task missing from my tasks")
	}
	w = f.call("GET", "/me/tasks", "", creator.Cookie)
	expect(t, w, 200)
	if dataOf[dto.TaskPage](t, w.Body.Bytes()).Total != 0 {
		t.Fatal("created task leaked into my assignments")
	}
	updated = patchTask(t, f, creator, updated, map[string]any{"assignee_id": nil, "due_at": nil}, 200)
	if updated.AssigneeID != nil || updated.DueAt != nil || updated.IsOverdue {
		t.Fatal("explicit null did not clear fields")
	}
	w = f.call("GET", list+"?assignee_id=unassigned&overdue=false", "", creator.Cookie)
	expect(t, w, 200)
	if dataOf[dto.TaskPage](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("unassigned/non-overdue filter failed")
	}
	patchTask(t, f, creator, task, map[string]any{"title": "旧版本不能覆盖"}, 409)
	for _, fields := range []map[string]any{{"assignee_id": strconv.FormatUint(outside.ID, 10)}, {"due_at": "2026-02-30"}, {"priority": "critical"}, {"title": strings.Repeat("汉", 201)}} {
		patchTask(t, f, creator, updated, fields, 400)
	}
	expect(t, f.call("PATCH", base, fmt.Sprintf(`{"version":%d,"status":"done"}`, updated.Version), member.Cookie), 400)
	expect(t, f.call("POST", list, `{"title":"不可注入状态","status":"done"}`, creator.Cookie), 400)
	if err := f.Users.DB.Model(&model.User{}).Where("id = ?", member.ID).Update("status", "disabled").Error; err != nil {
		t.Fatal(err)
	}
	patchTask(t, f, creator, updated, map[string]any{"assignee_id": strconv.FormatUint(member.ID, 10)}, 400)
	if err := f.Users.DB.Model(&model.User{}).Where("id = ?", member.ID).Update("status", "active").Error; err != nil {
		t.Fatal(err)
	}
	updated = patchTask(t, f, creator, updated, map[string]any{"assignee_id": strconv.FormatUint(member.ID, 10), "due_at": past}, 200)
	updated = moveTask(t, f, creator, updated, "done", nil)
	if updated.IsOverdue {
		t.Fatal("completed task marked overdue")
	}
	w = f.call("GET", list+"?overdue=true", "", creator.Cookie)
	expect(t, w, 200)
	if dataOf[dto.TaskPage](t, w.Body.Bytes()).Total != 0 {
		t.Fatal("done included in overdue filter")
	}
	wild := createTask(t, f, creator, project.ID, map[string]any{"title": "100%_验收"})
	createTask(t, f, creator, project.ID, map[string]any{"title": "100AB验收"})
	w = f.call("GET", list+"?keyword="+url.QueryEscape("100%_"), "", creator.Cookie)
	expect(t, w, 200)
	if dataOf[dto.TaskPage](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("LIKE wildcards not treated literally")
	}
	expect(t, f.call("DELETE", "/tasks/"+wild.ID+"?version="+strconv.FormatUint(wild.Version, 10), "", owner.Cookie), 204)
	updated = getTask(t, f, creator, task.ID)
	expect(t, f.call("DELETE", base+"?version="+strconv.FormatUint(updated.Version, 10), "", creator.Cookie), 204)
	expect(t, f.call("GET", base, "", owner.Cookie), 404)
	var stored model.Task
	if err := f.Users.DB.First(&stored, task.ID).Error; err != nil || stored.DeletedAt == nil {
		t.Fatal("task was not soft deleted")
	}
	w = f.call("GET", "/me/tasks", "", member.Cookie)
	expect(t, w, 200)
	if dataOf[dto.TaskPage](t, w.Body.Bytes()).Total != 0 {
		t.Fatal("deleted task remained in my tasks")
	}
	assertDense(t, f, project.ID)
}
func TestTaskOrderingAndVersionConflicts(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	team := createTeam(t, f, owner)
	project := createProject(t, f, owner, team.ID)
	a, b, c := createTask(t, f, owner, project.ID, nil), createTask(t, f, owner, project.ID, nil), createTask(t, f, owner, project.ID, nil)
	c = moveTask(t, f, owner, c, "todo", &a.ID)
	if c.Position != 0 {
		t.Fatal("before-task reorder failed")
	}
	patchTask(t, f, owner, b, map[string]any{"title": "邻居旧版本"}, 409)
	a = getTask(t, f, owner, a.ID)
	a = moveTask(t, f, owner, a, "in_progress", nil)
	c = getTask(t, f, owner, c.ID)
	c = moveTask(t, f, owner, c, "in_progress", &a.ID)
	assertDense(t, f, project.ID)
	before := logCount(t, f, team.ID)
	same := moveTask(t, f, owner, c, "in_progress", &a.ID)
	if same.Version != c.Version || logCount(t, f, team.ID) != before {
		t.Fatal("no-op move changed version or audit")
	}
	badBody := fmt.Sprintf(`{"version":%d,"status":"in_progress","before_task_id":%q}`, c.Version, c.ID)
	expect(t, f.call("POST", "/tasks/"+c.ID+"/move", badBody, owner.Cookie), 400)
	other := createProject(t, f, owner, team.ID)
	target := createTask(t, f, owner, other.ID, nil)
	badBody = fmt.Sprintf(`{"version":%d,"status":"todo","before_task_id":%q}`, c.Version, target.ID)
	expect(t, f.call("POST", "/tasks/"+c.ID+"/move", badBody, owner.Cookie), 409)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for _, title := range []string{"并发编辑一", "并发编辑二"} {
		wg.Add(1)
		go func(title string) {
			defer wg.Done()
			results <- f.call("PATCH", "/tasks/"+c.ID, fmt.Sprintf(`{"version":%d,"title":%q}`, c.Version, title), owner.Cookie).Code
		}(title)
	}
	wg.Wait()
	close(results)
	ok, conflict := 0, 0
	for status := range results {
		if status == 200 {
			ok++
		} else if status == 409 {
			conflict++
		} else {
			t.Fatalf("unexpected edit status %d", status)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatal("concurrent edit did not produce one winner")
	}
	c = getTask(t, f, owner, c.ID)
	results = make(chan int, 2)
	for _, status := range []string{"done", "in_review"} {
		wg.Add(1)
		go func(status string) {
			defer wg.Done()
			results <- f.call("POST", "/tasks/"+c.ID+"/move", fmt.Sprintf(`{"version":%d,"status":%q}`, c.Version, status), owner.Cookie).Code
		}(status)
	}
	wg.Wait()
	close(results)
	ok, conflict = 0, 0
	for status := range results {
		if status == 200 {
			ok++
		} else if status == 409 {
			conflict++
		} else {
			t.Fatalf("unexpected move status %d", status)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatal("concurrent move did not produce one winner")
	}
	assertDense(t, f, project.ID)
}
func TestTaskMembershipCleanupAcrossProjects(t *testing.T) {
	f := setupAuth(t)
	owner, member := actor(t, f), actor(t, f)
	a, b := createTeam(t, f, owner), createTeam(t, f, owner)
	ia, ib := createInvite(t, f, owner, a.ID, 20), createInvite(t, f, owner, b.ID, 20)
	join(t, f, member, ia.Code)
	join(t, f, member, ib.Code)
	pa, pArchived, pb := createProject(t, f, owner, a.ID), createProject(t, f, owner, a.ID), createProject(t, f, owner, b.ID)
	fields := func() map[string]any {
		return map[string]any{"title": "负责人的真实任务", "assignee_id": strconv.FormatUint(member.ID, 10)}
	}
	first, archived, other := createTask(t, f, owner, pa.ID, fields()), createTask(t, f, owner, pArchived.ID, fields()), createTask(t, f, owner, pb.ID, fields())
	expect(t, f.call("POST", "/projects/"+pArchived.ID+"/archive", "", owner.Cookie), 204)
	expect(t, f.call("DELETE", "/teams/"+a.ID+"/members/"+strconv.FormatUint(member.ID, 10), "", owner.Cookie), 204)
	for _, task := range []dto.Task{first, archived} {
		value := getTask(t, f, owner, task.ID)
		if value.AssigneeID != nil || value.Version != task.Version+1 {
			t.Fatal("assignment/version not cleared on removal")
		}
	}
	if getTask(t, f, owner, other.ID).AssigneeID == nil {
		t.Fatal("membership removal cleared another team's task")
	}
	expect(t, f.call("GET", "/tasks/"+first.ID, "", member.Cookie), 404)
	patchTask(t, f, owner, first, map[string]any{"title": "清理后的旧版本"}, 409)
	expect(t, f.call("POST", "/teams/"+b.ID+"/leave", "", member.Cookie), 204)
	if getTask(t, f, owner, other.ID).AssigneeID != nil {
		t.Fatal("leaving did not clear task assignment")
	}
	var count int64
	if err := f.Users.DB.Model(&model.ActivityLog{}).Where("action = ? AND target_id IN ?", "task.unassigned", []string{first.ID, archived.ID, other.ID}).Count(&count).Error; err != nil || count != 3 {
		t.Fatal("unassignment audit missing")
	}
}
func TestTaskAssignmentRemovalRaceAndArchive(t *testing.T) {
	f := setupAuth(t)
	owner, member := actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, member, invite.Code)
	project := createProject(t, f, owner, team.ID)
	task := createTask(t, f, owner, project.ID, nil)
	var wg sync.WaitGroup
	var assign, remove int
	wg.Add(2)
	go func() {
		defer wg.Done()
		assign = f.call("PATCH", "/tasks/"+task.ID, fmt.Sprintf(`{"version":%d,"assignee_id":%q}`, task.Version, strconv.FormatUint(member.ID, 10)), owner.Cookie).Code
	}()
	go func() {
		defer wg.Done()
		remove = f.call("DELETE", "/teams/"+team.ID+"/members/"+strconv.FormatUint(member.ID, 10), "", owner.Cookie).Code
	}()
	wg.Wait()
	if remove != 204 || (assign != 200 && assign != 400) {
		t.Fatalf("assignment/removal race invalid: %d/%d", assign, remove)
	}
	if getTask(t, f, owner, task.ID).AssigneeID != nil {
		t.Fatal("removed member retained assignment")
	}
	expect(t, f.call("POST", "/projects/"+project.ID+"/archive", "", owner.Cookie), 204)
	current := getTask(t, f, owner, task.ID)
	if current.CanDelete {
		t.Fatal("archived task still deletable")
	}
	patchTask(t, f, owner, current, map[string]any{"title": "归档不可编辑"}, 409)
	expect(t, f.call("DELETE", "/tasks/"+task.ID+"?version="+strconv.FormatUint(current.Version, 10), "", owner.Cookie), 409)
	expect(t, f.call("POST", "/tasks/"+task.ID+"/move", fmt.Sprintf(`{"version":%d,"status":"done"}`, current.Version), owner.Cookie), 409)
	expect(t, f.call("POST", "/projects/"+project.ID+"/tasks", `{"title":"归档不可创建"}`, owner.Cookie), 409)
	second := createProject(t, f, owner, team.ID)
	live := createTask(t, f, owner, second.ID, nil)
	expect(t, f.call("POST", "/teams/"+team.ID+"/archive", "", owner.Cookie), 204)
	patchTask(t, f, owner, live, map[string]any{"title": "父团队归档不可写"}, 409)
}
func TestTaskRollbackIncludesOrderAndMembership(t *testing.T) {
	f := setupAuth(t)
	owner, member := actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, member, invite.Code)
	project := createProject(t, f, owner, team.ID)
	first := createTask(t, f, owner, project.ID, map[string]any{"title": "回滚任务", "assignee_id": strconv.FormatUint(member.ID, 10)})
	second := createTask(t, f, owner, project.ID, nil)
	before := logCount(t, f, team.ID)
	var p model.Project
	if err := f.Users.DB.First(&p, project.ID).Error; err != nil {
		t.Fatal(err)
	}
	revision := p.TaskRevision
	if err := f.Users.DB.Callback().Create().Before("gorm:create").Register("test:task_audit_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "activity_logs" {
			tx.AddError(errors.New("injected task log failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Users.DB.Callback().Create().Remove("test:task_audit_failure") })
	expect(t, f.call("POST", "/projects/"+project.ID+"/tasks", `{"title":"不能提交的任务"}`, owner.Cookie), 503)
	patchTask(t, f, owner, first, map[string]any{"title": "不能提交的编辑"}, 503)
	expect(t, f.call("POST", "/tasks/"+second.ID+"/move", fmt.Sprintf(`{"version":%d,"status":"todo","before_task_id":%q}`, second.Version, first.ID), owner.Cookie), 503)
	expect(t, f.call("DELETE", "/tasks/"+first.ID+"?version="+strconv.FormatUint(first.Version, 10), "", owner.Cookie), 503)
	expect(t, f.call("DELETE", "/teams/"+team.ID+"/members/"+strconv.FormatUint(member.ID, 10), "", owner.Cookie), 503)
	actual := getTask(t, f, owner, first.ID)
	if actual.Title != first.Title || actual.Version != first.Version || actual.AssigneeID == nil || actual.Position != 0 {
		t.Fatal("task mutation partly committed")
	}
	if getTask(t, f, owner, second.ID).Position != 1 {
		t.Fatal("failed move changed order")
	}
	expect(t, f.call("GET", "/tasks/"+first.ID, "", member.Cookie), 200)
	if err := f.Users.DB.First(&p, project.ID).Error; err != nil || p.TaskRevision != revision {
		t.Fatal("rolled back revision changed")
	}
	if logCount(t, f, team.ID) != before {
		t.Fatal("failed operation left success event")
	}
	var count int64
	f.Users.DB.Model(&model.Task{}).Where("project_id = ?", project.ID).Count(&count)
	if count != 2 {
		t.Fatal("failed create left row")
	}
	assertDense(t, f, project.ID)
}
func TestBoardPaginationSnapshotAndValidation(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	team := createTeam(t, f, owner)
	project := createProject(t, f, owner, team.ID)
	for i := 0; i < 55; i++ {
		createTask(t, f, owner, project.ID, map[string]any{"title": fmt.Sprintf("分页任务-%02d", i)})
	}
	w := f.call("GET", "/projects/"+project.ID+"/board", "", owner.Cookie)
	expect(t, w, 200)
	board := dataOf[dto.Board](t, w.Body.Bytes())
	if len(board.Columns) != 4 || len(board.Columns[0].Items) != 50 || board.Columns[0].Total != 55 || !board.Columns[0].HasMore {
		t.Fatal("first board page invalid")
	}
	query := fmt.Sprintf("/projects/%s/tasks?status=todo&page=2&page_size=50&revision=%d", project.ID, board.Revision)
	w = f.call("GET", query, "", owner.Cookie)
	expect(t, w, 200)
	next := dataOf[dto.TaskPage](t, w.Body.Bytes())
	if len(next.Items) != 5 || next.Revision == nil || *next.Revision != board.Revision {
		t.Fatal("board continuation invalid")
	}
	seen := map[string]bool{}
	for _, task := range append(board.Columns[0].Items, next.Items...) {
		if seen[task.ID] {
			t.Fatal("duplicate board card")
		}
		seen[task.ID] = true
	}
	patchTask(t, f, owner, board.Columns[0].Items[0], map[string]any{"priority": "high"}, 200)
	expect(t, f.call("GET", query, "", owner.Cookie), 409)
	for _, suffix := range []string{"?status=invalid", "?priority=invalid", "?overdue=1", "?assignee_id=abc", "?revision=-1", "?page_size=101"} {
		expect(t, f.call("GET", "/projects/"+project.ID+"/tasks"+suffix, "", owner.Cookie), 400)
	}
	expect(t, f.call("GET", "/me/tasks?assignee_id="+strconv.FormatUint(owner.ID, 10), "", owner.Cookie), 400)
}
func TestTaskRevisionMigrationRecovery(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	team := createTeam(t, f, owner)
	project := createProject(t, f, owner, team.ID)
	createTask(t, f, owner, project.ID, nil)
	if err := f.Users.DB.Exec("DELETE FROM schema_migrations WHERE version = ?", 10).Error; err != nil {
		t.Fatal(err)
	}
	pool, err := f.Users.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = migrations.Run(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var p model.Project
	if err = f.Users.DB.First(&p, project.ID).Error; err != nil || p.TaskRevision != 1 {
		t.Fatal("migration retry reset project task revision")
	}
}
