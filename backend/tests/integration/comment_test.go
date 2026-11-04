package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"taskflow/internal/dto"
	"taskflow/internal/model"
)

func createComment(t *testing.T, f *authFixture, a teamActor, taskID, content string) dto.Comment {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"content": content})
	w := f.call("POST", "/tasks/"+taskID+"/comments", string(body), a.Cookie)
	expect(t, w, 201)
	return dataOf[dto.Comment](t, w.Body.Bytes())
}
func commentList(t *testing.T, f *authFixture, a teamActor, taskID string) dto.Page[dto.Comment] {
	t.Helper()
	w := f.call("GET", "/tasks/"+taskID+"/comments", "", a.Cookie)
	expect(t, w, 200)
	return dataOf[dto.Page[dto.Comment]](t, w.Body.Bytes())
}
func updateComment(t *testing.T, f *authFixture, a teamActor, c dto.Comment, content string, status int) dto.Comment {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"content": content, "version": c.Version})
	w := f.call("PATCH", "/comments/"+c.ID, string(body), a.Cookie)
	expect(t, w, status)
	if status == 200 {
		return dataOf[dto.Comment](t, w.Body.Bytes())
	}
	return c
}
func deleteCommentPath(c dto.Comment) string {
	return "/comments/" + c.ID + "?version=" + strconv.FormatUint(c.Version, 10)
}
func TestCommentLifecyclePermissionsAndValidation(t *testing.T) {
	f := setupAuth(t)
	owner, author, member, outside := actor(t, f), actor(t, f), actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, author, invite.Code)
	join(t, f, member, invite.Code)
	project := createProject(t, f, owner, team.ID)
	task := createTask(t, f, owner, project.ID, nil)
	base := "/tasks/" + task.ID + "/comments"
	expect(t, f.call("GET", base, "", nil), 401)
	c := createComment(t, f, author, task.ID, "  真实评论\n<script>alert('xss')</script>  ")
	if c.Content != "真实评论\n<script>alert('xss')</script>" || c.Version != 1 || !c.CanEdit || !c.CanDelete || c.AuthorID != strconv.FormatUint(author.ID, 10) {
		t.Fatal("invalid comment DTO")
	}
	others := commentList(t, f, member, task.ID)
	if others.Total != 1 || others.Items[0].CanEdit || others.Items[0].CanDelete {
		t.Fatal("non-author granted mutation")
	}
	ownerview := commentList(t, f, owner, task.ID)
	if ownerview.Items[0].CanEdit || !ownerview.Items[0].CanDelete {
		t.Fatal("owner comment permissions invalid")
	}
	for _, a := range []teamActor{owner, member} {
		updateComment(t, f, a, c, "越权编辑", 403)
	}
	expect(t, f.call("DELETE", deleteCommentPath(c), "", member.Cookie), 403)
	f.Users.DB.Model(&model.User{}).Where("id = ?", outside.ID).Update("platform_role", "admin")
	for _, p := range []string{base, "/projects/" + project.ID + "/activities", "/teams/" + team.ID + "/activities"} {
		expect(t, f.call("GET", p, "", outside.Cookie), 404)
	}
	updateComment(t, f, outside, c, "跨团队", 404)
	expect(t, f.call("DELETE", deleteCommentPath(c), "", outside.Cookie), 404)
	for _, body := range []string{`{}`, `{"content":"  "}`, `{"content":"ok","author_id":"1"}`, `{"content":null}`, fmt.Sprintf(`{"content":%q}`, strings.Repeat("长", 2001))} {
		expect(t, f.call("POST", base, body, author.Cookie), 400)
	}
	for _, body := range []string{`{"content":"ok"}`, `{"content":"ok","version":0}`, `{"content":"","version":1}`, `{"content":"ok","version":1,"task_id":"1"}`} {
		expect(t, f.call("PATCH", "/comments/"+c.ID, body, author.Cookie), 400)
	}
	expect(t, f.call("DELETE", "/comments/"+c.ID, "", author.Cookie), 400)
	updated := updateComment(t, f, author, c, "已编辑的真实评论", 200)
	if updated.Version != 2 {
		t.Fatal("version did not advance")
	}
	updateComment(t, f, author, c, "旧版本覆盖", 409)
	expect(t, f.call("DELETE", deleteCommentPath(c), "", author.Cookie), 409)
	expect(t, f.call("DELETE", deleteCommentPath(updated), "", owner.Cookie), 204)
	if commentList(t, f, author, task.ID).Total != 0 {
		t.Fatal("deleted comment remained visible")
	}
	var stored model.Comment
	if err := f.Users.DB.First(&stored, c.ID).Error; err != nil || stored.DeletedAt == nil || stored.Content != updated.Content {
		t.Fatal("comment not soft deleted")
	}
	updateComment(t, f, author, updated, "已删除", 404)
	own := createComment(t, f, author, task.ID, "作者可自行删除")
	expect(t, f.call("DELETE", deleteCommentPath(own), "", author.Cookie), 204)
	if getTask(t, f, owner, task.ID).Version != task.Version {
		t.Fatal("comments should not modify task version")
	}
}
func TestCommentArchiveDeletionAndMembership(t *testing.T) {
	f := setupAuth(t)
	owner, author := actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, author, invite.Code)
	p := createProject(t, f, owner, team.ID)
	task := createTask(t, f, owner, p.ID, nil)
	c := createComment(t, f, author, task.ID, "归档保留讨论")
	expect(t, f.call("POST", "/projects/"+p.ID+"/archive", "", owner.Cookie), 204)
	list := commentList(t, f, author, task.ID)
	if list.Items[0].CanEdit || list.Items[0].CanDelete {
		t.Fatal("archived comments writable")
	}
	updateComment(t, f, author, c, "修改归档", 409)
	expect(t, f.call("DELETE", deleteCommentPath(c), "", owner.Cookie), 409)
	expect(t, f.call("POST", "/tasks/"+task.ID+"/comments", `{"content":"归档后创建"}`, author.Cookie), 409)
	p2 := createProject(t, f, owner, team.ID)
	t2 := createTask(t, f, owner, p2.ID, nil)
	c2 := createComment(t, f, author, t2.ID, "任务删除后的历史")
	expect(t, f.call("DELETE", "/tasks/"+t2.ID+"?version=1", "", owner.Cookie), 204)
	expect(t, f.call("GET", "/tasks/"+t2.ID+"/comments", "", author.Cookie), 404)
	updateComment(t, f, author, c2, "删除任务后编辑", 404)
	expect(t, f.call("DELETE", deleteCommentPath(c2), "", owner.Cookie), 404)
	expect(t, f.call("DELETE", "/teams/"+team.ID+"/members/"+strconv.FormatUint(author.ID, 10), "", owner.Cookie), 204)
	expect(t, f.call("GET", "/tasks/"+task.ID+"/comments", "", author.Cookie), 404)
	updateComment(t, f, author, c, "离队编辑", 404)
	p3 := createProject(t, f, owner, team.ID)
	t3 := createTask(t, f, owner, p3.ID, nil)
	c3 := createComment(t, f, owner, t3.ID, "团队归档保留")
	expect(t, f.call("POST", "/teams/"+team.ID+"/archive", "", owner.Cookie), 204)
	updateComment(t, f, owner, c3, "团队归档编辑", 409)
	if commentList(t, f, owner, t3.ID).Items[0].CanEdit {
		t.Fatal("team archive not applied")
	}
}
func TestCommentAuditRollbackAndConcurrency(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	team := createTeam(t, f, owner)
	p := createProject(t, f, owner, team.ID)
	task := createTask(t, f, owner, p.ID, nil)
	c := createComment(t, f, owner, task.ID, "初始内容")
	before := logCount(t, f, team.ID)
	name := "test:comment_audit_failure"
	if err := f.Users.DB.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "activity_logs" {
			tx.AddError(errors.New("injected comment audit failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Users.DB.Callback().Create().Remove(name) })
	expect(t, f.call("POST", "/tasks/"+task.ID+"/comments", `{"content":"必须回滚"}`, owner.Cookie), 503)
	updateComment(t, f, owner, c, "必须回滚", 503)
	expect(t, f.call("DELETE", deleteCommentPath(c), "", owner.Cookie), 503)
	list := commentList(t, f, owner, task.ID)
	if list.Total != 1 || list.Items[0].Content != c.Content || list.Items[0].Version != 1 || logCount(t, f, team.ID) != before {
		t.Fatal("comment and audit were not atomic")
	}
	f.Users.DB.Callback().Create().Remove(name)
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- f.call("PATCH", "/comments/"+c.ID, `{"content":"并发修改","version":1}`, owner.Cookie).Code
		}()
	}
	wg.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent results: %v", counts)
	}
	var logs []model.ActivityLog
	if err := f.Users.DB.Where("team_id = ? AND target_type = ?", team.ID, "comment").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatal("failed request left success event")
	}
	for _, log := range logs {
		if strings.Contains(log.Metadata, "并发修改") || strings.Contains(log.Metadata, "初始内容") {
			t.Fatal("audit should not copy comment content")
		}
	}
}
func TestCommentPaginationAndActivityFilters(t *testing.T) {
	f := setupAuth(t)
	owner, author := actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, author, invite.Code)
	p := createProject(t, f, owner, team.ID)
	task := createTask(t, f, owner, p.ID, nil)
	start := time.Now().UTC().Add(-time.Second)
	for i := 0; i < 23; i++ {
		createComment(t, f, author, task.ID, fmt.Sprintf("评论 %d", i))
	}
	end := time.Now().UTC().Add(time.Second)
	list := commentList(t, f, owner, task.ID)
	if len(list.Items) != 20 || list.Total != 23 {
		t.Fatal("wrong first page")
	}
	w := f.call("GET", "/tasks/"+task.ID+"/comments?page=2", "", owner.Cookie)
	expect(t, w, 200)
	second := dataOf[dto.Page[dto.Comment]](t, w.Body.Bytes())
	if len(second.Items) != 3 || second.Total != 23 || second.Items[0].ID == list.Items[0].ID {
		t.Fatal("wrong second page")
	}
	base := "/teams/" + team.ID + "/activities"
	q := url.Values{"action": {"comment.created"}, "target_type": {"comment"}, "actor_id": {strconv.FormatUint(author.ID, 10)}, "project_id": {p.ID}, "from": {start.Format(time.RFC3339Nano)}, "to": {end.Format(time.RFC3339Nano)}, "page_size": {"100"}}
	w = f.call("GET", base+"?"+q.Encode(), "", owner.Cookie)
	expect(t, w, 200)
	events := dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes())
	if events.Total != 23 || len(events.Items) != 23 {
		t.Fatalf("filters returned %d", events.Total)
	}
	for _, e := range events.Items {
		if e.Action != "comment.created" || e.ActorID != strconv.FormatUint(author.ID, 10) || e.ProjectID == nil || *e.ProjectID != p.ID {
			t.Fatal("filter leak")
		}
	}
	for _, bad := range []string{"action=bogus", "target_type=bogus", "actor_id=0", "project_id=bad", "from=not-a-date", "from=2026-02-01T00%3A00%3A00Z&to=2026-01-01T00%3A00%3A00Z"} {
		expect(t, f.call("GET", base+"?"+bad, "", owner.Cookie), 400)
	}
	p2 := createProject(t, f, owner, team.ID)
	w = f.call("GET", "/projects/"+p2.ID+"/activities?action=comment.created", "", owner.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes()).Total != 0 {
		t.Fatal("project audit isolation failed")
	}
	other := createTeam(t, f, owner)
	w = f.call("GET", "/teams/"+other.ID+"/activities?project_id="+p.ID, "", owner.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes()).Total != 0 {
		t.Fatal("team filter escaped scope")
	}
	expect(t, f.call("POST", "/teams/"+team.ID+"/leave", "", author.Cookie), 204)
	w = f.call("GET", base+"?actor_id="+strconv.FormatUint(author.ID, 10), "", owner.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes()).Total < 23 {
		t.Fatal("removed author audit lost")
	}
	expect(t, f.call("GET", base, "", author.Cookie), 404)
	var n int64
	if err := f.Users.DB.WithContext(context.Background()).Model(&model.Comment{}).Where("task_id = ?", task.ID).Count(&n).Error; err != nil || n != 23 {
		t.Fatal("real comments not persisted")
	}
}
func TestCommentRemovalRaceAndRateLimit(t *testing.T) {
	f := setupAuth(t)
	owner, author := actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	join(t, f, author, invite.Code)
	p := createProject(t, f, owner, team.ID)
	task := createTask(t, f, owner, p.ID, nil)
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		statuses <- f.call("POST", "/tasks/"+task.ID+"/comments", `{"content":"并发移除时评论"}`, author.Cookie).Code
	}()
	go func() {
		defer wg.Done()
		statuses <- f.call("DELETE", "/teams/"+team.ID+"/members/"+strconv.FormatUint(author.ID, 10), "", owner.Cookie).Code
	}()
	wg.Wait()
	close(statuses)
	for s := range statuses {
		if s != 201 && s != 204 && s != 404 {
			t.Fatalf("unexpected race result %d", s)
		}
	}
	expect(t, f.call("POST", "/tasks/"+task.ID+"/comments", `{"content":"移除后评论"}`, author.Cookie), 404)
	// Exercise the actual Redis-backed user limiter without persisting 120 comments.
	for i := 0; i < 120; i++ {
		expect(t, f.call("POST", "/tasks/"+task.ID+"/comments", `{"content":""}`, owner.Cookie), 400)
	}
	expect(t, f.call("POST", "/tasks/"+task.ID+"/comments", `{"content":"限流后"}`, owner.Cookie), 429)
}
