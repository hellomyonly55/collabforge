package integration

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"taskflow/internal/dto"
	"taskflow/internal/infrastructure/password"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"taskflow/internal/service"
	"testing"
	"time"
)

func adminActor(t *testing.T, f *authFixture) teamActor {
	t.Helper()
	email := "admin-" + unique() + "@example.test"
	u, err := service.BootstrapAdmin(context.Background(), &repository.AdminRepository{DB: f.Users.DB}, dto.RegisterRequest{Email: email, Nickname: "平台管理员", Password: "test-password-123"})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := strconv.ParseUint(u.ID, 10, 64)
	return teamActor{ID: id, Cookie: login(t, f, email, "test-password-123")}
}
func TestAdminBootstrapAndAuthorization(t *testing.T) {
	f := setupAuth(t)
	admin, normal := adminActor(t, f), actor(t, f)
	u, err := f.Users.ByID(context.Background(), admin.ID)
	if err != nil || u.PlatformRole != "admin" || !password.Matches(u.PasswordHash, "test-password-123") || u.PasswordHash == "test-password-123" {
		t.Fatal("unsafe admin bootstrap")
	}
	target, _ := f.Users.ByID(context.Background(), normal.ID)
	_, err = service.BootstrapAdmin(context.Background(), &repository.AdminRepository{DB: f.Users.DB}, dto.RegisterRequest{Email: target.Email, Nickname: "不能提权", Password: "test-password-123"})
	if service.AsAppError(err).Status != 409 {
		t.Fatal("existing email overwritten")
	}
	target, _ = f.Users.ByID(context.Background(), normal.ID)
	if target.PlatformRole != "user" {
		t.Fatal("ordinary user promoted")
	}
	for _, path := range []string{"/admin/users", "/admin/teams", "/admin/statistics"} {
		expect(t, f.call("GET", path, "", nil), 401)
		expect(t, f.call("GET", path, "", normal.Cookie), 403)
		w := f.call("GET", path, "", admin.Cookie)
		expect(t, w, 200)
		if strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), "session_version") || strings.Contains(w.Body.String(), "$2") {
			t.Fatal("private user fields leaked")
		}
	}
	expect(t, f.call("PATCH", fmt.Sprintf("/admin/users/%d/status", normal.ID), `{"status":"disabled","reason":"越权"}`, normal.Cookie), 403)
	for _, body := range []string{`{"status":"unknown","reason":"x"}`, `{"status":"disabled"}`, `{"status":"disabled","reason":"x","platform_role":"admin"}`} {
		expect(t, f.call("PATCH", fmt.Sprintf("/admin/users/%d/status", normal.ID), body, admin.Cookie), 400)
	}
	expect(t, f.call("PATCH", fmt.Sprintf("/admin/users/%d/status", admin.ID), `{"status":"disabled","reason":"不能禁用自己"}`, admin.Cookie), 400)
}
func TestAdminStatusRevokesAllSessionsWithoutTransferringOwnership(t *testing.T) {
	f := setupAuth(t)
	admin, owner := adminActor(t, f), actor(t, f)
	u, _ := f.Users.ByID(context.Background(), owner.ID)
	second := login(t, f, u.Email, "test-password-123")
	team := createTeam(t, f, owner)
	p := createProject(t, f, owner, team.ID)
	task := createTask(t, f, owner, p.ID, map[string]any{"title": "保留分配", "assignee_id": strconv.FormatUint(owner.ID, 10)})
	path := fmt.Sprintf("/admin/users/%d/status", owner.ID)
	w := f.call("PATCH", path, `{"status":"disabled","reason":"安全处置"}`, admin.Cookie)
	expect(t, w, 200)
	row := dataOf[dto.AdminUser](t, w.Body.Bytes())
	if row.Status != "disabled" || row.OwnedTeamCount != 1 {
		t.Fatal("wrong admin DTO")
	}
	for _, cookie := range []*http.Cookie{owner.Cookie, second} {
		expect(t, f.call("GET", "/me", "", cookie), 401)
	}
	expect(t, f.call("POST", "/auth/login", loginBody(u.Email, "test-password-123"), nil), 401)
	var storedTeam model.Team
	var storedTask model.Task
	f.Users.DB.First(&storedTeam, team.ID)
	f.Users.DB.First(&storedTask, task.ID)
	if storedTeam.OwnerID != owner.ID || storedTask.AssigneeID == nil || *storedTask.AssigneeID != owner.ID {
		t.Fatal("disable changed ownership/assignment")
	}
	expect(t, f.call("PATCH", path, `{"status":"active","reason":"核查通过"}`, admin.Cookie), 200)
	for _, cookie := range []*http.Cookie{owner.Cookie, second} {
		expect(t, f.call("GET", "/me", "", cookie), 401)
	}
	fresh := login(t, f, u.Email, "test-password-123")
	expect(t, f.call("GET", "/teams/"+team.ID, "", fresh), 200)
	var count int64
	f.Users.DB.Model(&model.PlatformAuditLog{}).Where("target_type = ? AND target_id = ? AND action = ?", "user", owner.ID, "user.status_changed").Count(&count)
	if count != 2 {
		t.Fatal("status audit missing")
	}
	expect(t, f.call("PATCH", path, `{"status":"active","reason":"重复请求"}`, admin.Cookie), 200)
	var after int64
	f.Users.DB.Model(&model.PlatformAuditLog{}).Where("target_type = ? AND target_id = ? AND action = ?", "user", owner.ID, "user.status_changed").Count(&after)
	if after != count {
		t.Fatal("no-op created audit")
	}
}
func TestAdminArchiveListsAndStatistics(t *testing.T) {
	f := setupAuth(t)
	admin, owner := adminActor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	p := createProject(t, f, owner, team.ID)
	createTask(t, f, owner, p.ID, map[string]any{"title": "真实逾期", "due_at": time.Now().Add(-72 * time.Hour).Format(time.DateOnly)})
	expect(t, f.call("GET", "/teams/"+team.ID, "", admin.Cookie), 404)
	expect(t, f.call("GET", "/projects/"+p.ID+"/tasks", "", admin.Cookie), 404)
	user, _ := f.Users.ByID(context.Background(), owner.ID)
	w := f.call("GET", "/admin/users?keyword="+user.Email+"&role=user&status=active", "", admin.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.AdminUser]](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("user filters wrong")
	}
	w = f.call("GET", "/admin/teams?keyword="+team.Name, "", admin.Cookie)
	expect(t, w, 200)
	teams := dataOf[dto.Page[dto.AdminTeam]](t, w.Body.Bytes())
	if teams.Total != 1 || teams.Items[0].OwnerID != strconv.FormatUint(owner.ID, 10) || teams.Items[0].TaskCount != 1 || teams.Items[0].ProjectCount != 1 {
		t.Fatal("team summary wrong")
	}
	w = f.call("GET", "/admin/statistics", "", admin.Cookie)
	expect(t, w, 200)
	stats := dataOf[dto.AdminStatistics](t, w.Body.Bytes())
	var count int64
	f.Users.DB.Model(&model.User{}).Count(&count)
	if stats.Users != count || stats.ActiveUsers+stats.DisabledUsers != stats.Users || stats.Tasks < 1 || stats.OverdueTasks < 1 {
		t.Fatal("global statistics wrong")
	}
	expect(t, f.call("POST", "/admin/teams/"+team.ID+"/archive", `{"reason":"异常团队处置"}`, admin.Cookie), 204)
	expect(t, f.call("POST", "/admin/teams/"+team.ID+"/archive", `{"reason":"重复"}`, admin.Cookie), 409)
	expect(t, f.call("POST", "/teams/"+team.ID+"/projects", `{"name":"不能创建"}`, owner.Cookie), 409)
	expect(t, f.call("GET", "/teams/"+team.ID, "", admin.Cookie), 404)
	w = f.call("GET", "/teams/"+team.ID+"/activities", "", owner.Cookie)
	expect(t, w, 200)
	if !strings.Contains(w.Body.String(), "platform_admin") || !strings.Contains(w.Body.String(), "异常团队处置") {
		t.Fatal("team audit missing")
	}
	for _, path := range []string{"/admin/users?status=bad", "/admin/users?role=bad", "/admin/teams?status=disabled", "/admin/users?page_size=101"} {
		expect(t, f.call("GET", path, "", admin.Cookie), 400)
	}
}
func TestAdminMutationsAuditRollbackAndConcurrentDisables(t *testing.T) {
	f := setupAuth(t)
	admin, other, normal := adminActor(t, f), adminActor(t, f), actor(t, f)
	team := createTeam(t, f, normal)
	before, _ := f.Users.ByID(context.Background(), normal.ID)
	name := "test:platform_audit_failure"
	if err := f.Users.DB.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "platform_audit_logs" {
			tx.AddError(errors.New("private audit failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Users.DB.Callback().Create().Remove(name) })
	expect(t, f.call("PATCH", fmt.Sprintf("/admin/users/%d/status", normal.ID), `{"status":"disabled","reason":"回滚"}`, admin.Cookie), 503)
	expect(t, f.call("POST", "/admin/teams/"+team.ID+"/archive", `{"reason":"回滚"}`, admin.Cookie), 503)
	after, _ := f.Users.ByID(context.Background(), normal.ID)
	var stored model.Team
	f.Users.DB.First(&stored, team.ID)
	if after.Status != before.Status || after.SessionVersion != before.SessionVersion || stored.ArchivedAt != nil {
		t.Fatal("admin write partly committed")
	}
	var teamEvents int64
	f.Users.DB.Model(&model.ActivityLog{}).Where("team_id = ? AND action = ?", team.ID, "team.archived").Count(&teamEvents)
	if teamEvents != 0 {
		t.Fatal("failed archive left team event")
	}
	email := "failed-" + unique() + "@example.test"
	_, err := service.BootstrapAdmin(context.Background(), &repository.AdminRepository{DB: f.Users.DB}, dto.RegisterRequest{Email: email, Nickname: "回滚管理员", Password: "test-password-123"})
	if err == nil {
		t.Fatal("bootstrap ignored audit error")
	}
	if _, err = f.Users.ByEmail(context.Background(), email); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("failed bootstrap left user")
	}
	f.Users.DB.Callback().Create().Remove(name)
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for _, pair := range [][2]teamActor{{admin, other}, {other, admin}} {
		pair := pair
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- f.call("PATCH", fmt.Sprintf("/admin/users/%d/status", pair[1].ID), `{"status":"disabled","reason":"并发处置"}`, pair[0].Cookie).Code
		}()
	}
	wg.Wait()
	close(statuses)
	success := 0
	for code := range statuses {
		if code == 200 {
			success++
		} else if code != 401 && code != 403 && code != 409 {
			t.Fatalf("unexpected status %d", code)
		}
	}
	var active int64
	f.Users.DB.Model(&model.User{}).Where("id IN ? AND status = ?", []uint64{admin.ID, other.ID}, "active").Count(&active)
	if active != 1 || success != 1 {
		t.Fatal("concurrent admins disabled each other")
	}
}
func TestAdminSessionVersionAuthorityWhenRedisCleanupFails(t *testing.T) {
	f := setupAuth(t)
	admin, target := adminActor(t, f), actor(t, f)
	actorUser, _ := f.Users.ByID(context.Background(), admin.ID)
	second := setupAuth(t)
	second.Sessions.Prefix = f.Sessions.Prefix
	svc := &service.AdminService{Repo: &repository.AdminRepository{DB: f.Users.DB}, Auth: f.Auth}
	if err := f.Sessions.Client.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Status(context.Background(), actorUser, target.ID, dto.UserStatusRequest{Status: "disabled", Reason: "Redis 不可用时撤销"}); err != nil {
		t.Fatal(err)
	}
	expect(t, second.call("GET", "/me", "", target.Cookie), 401)
	if _, err := svc.Status(context.Background(), actorUser, target.ID, dto.UserStatusRequest{Status: "active", Reason: "重新启用"}); err != nil {
		t.Fatal(err)
	}
	expect(t, second.call("GET", "/me", "", target.Cookie), 401)
}
