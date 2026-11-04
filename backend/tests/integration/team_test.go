package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"taskflow/internal/service"
	"testing"
	"time"
)

type teamActor struct {
	ID     uint64
	Cookie *http.Cookie
}

func actor(t *testing.T, f *authFixture) teamActor {
	t.Helper()
	email := "team-" + unique() + "@example.test"
	pwd := "test-password-123"
	expect(t, f.call("POST", "/auth/register", userBody(email, pwd), nil), 201)
	u, err := f.Users.ByEmail(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	return teamActor{u.ID, login(t, f, email, pwd)}
}
func dataOf[T any](t *testing.T, body []byte) T {
	t.Helper()
	var envelope struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}
func createTeam(t *testing.T, f *authFixture, a teamActor) dto.Team {
	t.Helper()
	w := f.call("POST", "/teams", fmt.Sprintf(`{"name":%q,"description":"真实团队简介"}`, "测试团队-"+unique()), a.Cookie)
	expect(t, w, 201)
	return dataOf[dto.Team](t, w.Body.Bytes())
}
func createInvite(t *testing.T, f *authFixture, a teamActor, id string, uses int) dto.InviteCreated {
	t.Helper()
	w := f.call("POST", "/teams/"+id+"/invites", fmt.Sprintf(`{"max_uses":%d}`, uses), a.Cookie)
	expect(t, w, 201)
	return dataOf[dto.InviteCreated](t, w.Body.Bytes())
}
func join(t *testing.T, f *authFixture, a teamActor, code string) dto.JoinedTeam {
	t.Helper()
	w := f.call("POST", "/teams/join", fmt.Sprintf(`{"code":%q}`, code), a.Cookie)
	expect(t, w, 200)
	return dataOf[dto.JoinedTeam](t, w.Body.Bytes())
}
func assertOwner(t *testing.T, f *authFixture, id string, want uint64) {
	t.Helper()
	var team model.Team
	if err := f.Users.DB.First(&team, id).Error; err != nil {
		t.Fatal(err)
	}
	var owners []model.TeamMember
	if err := f.Users.DB.Where("team_id = ? AND role = 'owner' AND removed_at IS NULL", id).Find(&owners).Error; err != nil {
		t.Fatal(err)
	}
	if len(owners) != 1 || owners[0].UserID != team.OwnerID || team.OwnerID != want {
		t.Fatalf("owner invariant failed: %+v, owners=%+v", team, owners)
	}
}
func logCount(t *testing.T, f *authFixture, id string) int64 {
	t.Helper()
	var count int64
	if err := f.Users.DB.Model(&model.ActivityLog{}).Where("team_id = ?", id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestTeamLifecycleAndPermissions(t *testing.T) {
	f := setupAuth(t)
	owner, admin, member, outsider := actor(t, f), actor(t, f), actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	base := "/teams/" + team.ID
	assertOwner(t, f, team.ID, owner.ID)
	w := f.call("GET", "/teams", "", owner.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Team]](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("owner list incorrect")
	}
	w = f.call("GET", "/teams", "", outsider.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Team]](t, w.Body.Bytes()).Total != 0 {
		t.Fatal("team list leaked")
	}
	invite := createInvite(t, f, owner, team.ID, 20)
	digest := sha256.Sum256([]byte(invite.Code))
	var stored model.TeamInvite
	if err := f.Users.DB.First(&stored, invite.Invite.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.TokenHash != hex.EncodeToString(digest[:]) || stored.TokenHash == invite.Code {
		t.Fatal("invite is not stored as digest")
	}
	if join(t, f, admin, invite.Code).AlreadyJoined || join(t, f, member, invite.Code).AlreadyJoined {
		t.Fatal("first joins marked duplicate")
	}
	if !join(t, f, member, invite.Code).AlreadyJoined {
		t.Fatal("repeat join not idempotent")
	}
	if err := f.Users.DB.First(&stored, invite.Invite.ID).Error; err != nil || stored.UsedCount != 2 {
		t.Fatal("repeat join consumed invite")
	}
	before := logCount(t, f, team.ID)
	expect(t, f.call("POST", base+"/leave", "", owner.Cookie), 409)
	expect(t, f.call("PATCH", base, `{"name":"越权修改"}`, member.Cookie), 403)
	expect(t, f.call("GET", base+"/invites", "", member.Cookie), 403)
	expect(t, f.call("POST", base+"/invites", `{}`, member.Cookie), 403)
	expect(t, f.call("POST", base+"/archive", "", member.Cookie), 403)
	expect(t, f.call("PATCH", base+"/members/"+strconv.FormatUint(admin.ID, 10), `{"role":"admin"}`, member.Cookie), 403)
	if logCount(t, f, team.ID) != before {
		t.Fatal("rejected writes added audit events")
	}
	expect(t, f.call("PATCH", base+"/members/"+strconv.FormatUint(admin.ID, 10), `{"role":"admin"}`, owner.Cookie), 204)
	expect(t, f.call("PATCH", base, `{"description":"管理员可编辑"}`, admin.Cookie), 200)
	expect(t, f.call("PATCH", base+"/members/"+strconv.FormatUint(member.ID, 10), `{"role":"admin"}`, admin.Cookie), 403)
	expect(t, f.call("POST", base+"/archive", "", admin.Cookie), 403)
	expect(t, f.call("DELETE", base+"/members/"+strconv.FormatUint(owner.ID, 10), "", admin.Cookie), 409)
	expect(t, f.call("PATCH", base+"/members/"+strconv.FormatUint(member.ID, 10), `{"role":"admin"}`, owner.Cookie), 204)
	expect(t, f.call("DELETE", base+"/members/"+strconv.FormatUint(member.ID, 10), "", admin.Cookie), 403)
	expect(t, f.call("PATCH", base+"/members/"+strconv.FormatUint(member.ID, 10), `{"role":"member"}`, owner.Cookie), 204)
	expect(t, f.call("DELETE", base+"/members/"+strconv.FormatUint(member.ID, 10), "", admin.Cookie), 204)
	expect(t, f.call("GET", base, "", member.Cookie), 404)
	if join(t, f, member, invite.Code).Team.MyRole != "member" {
		t.Fatal("rejoin role invalid")
	}
	expect(t, f.call("POST", base+"/leave", "", member.Cookie), 204)
	expect(t, f.call("GET", base+"/members", "", member.Cookie), 404)
	join(t, f, member, invite.Code)
	var membershipCount int64
	f.Users.DB.Model(&model.TeamMember{}).Where("team_id = ? AND user_id = ?", team.ID, member.ID).Count(&membershipCount)
	if membershipCount != 1 {
		t.Fatal("rejoin duplicated membership")
	}
	// A platform admin gets no implicit team permission.
	if err := f.Users.DB.Model(&model.User{}).Where("id = ?", outsider.ID).Update("platform_role", "admin").Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{base, base + "/members", base + "/invites", base + "/activities"} {
		expect(t, f.call("GET", path, "", outsider.Cookie), 404)
	}
	other := createTeam(t, f, outsider)
	expect(t, f.call("DELETE", "/teams/"+other.ID+"/invites/"+invite.Invite.ID, "", outsider.Cookie), 404)
	expect(t, f.call("PATCH", base+"/members/"+strconv.FormatUint(outsider.ID, 10), `{"role":"admin"}`, owner.Cookie), 404)
	expect(t, f.call("PATCH", base+"/members/"+strconv.FormatUint(member.ID, 10), `{"role":"owner"}`, owner.Cookie), 400)
	expect(t, f.call("GET", "/teams/0", "", owner.Cookie), 400)
	expect(t, f.call("GET", base+"/members?page_size=101", "", owner.Cookie), 400)
	w = f.call("GET", base+"/invites", "", owner.Cookie)
	expect(t, w, 200)
	if strings.Contains(w.Body.String(), invite.Code) || strings.Contains(w.Body.String(), stored.TokenHash) || strings.Contains(w.Body.String(), "token_hash") {
		t.Fatal("invite list leaked token")
	}
	expect(t, f.call("DELETE", base+"/invites/"+invite.Invite.ID, "", owner.Cookie), 204)
	afterRevoke := logCount(t, f, team.ID)
	expect(t, f.call("DELETE", base+"/invites/"+invite.Invite.ID, "", owner.Cookie), 204)
	if logCount(t, f, team.ID) != afterRevoke {
		t.Fatal("duplicate revoke added log")
	}
	expect(t, f.call("POST", "/teams/join", fmt.Sprintf(`{"code":%q}`, invite.Code), outsider.Cookie), 400)
	if !join(t, f, member, invite.Code).AlreadyJoined {
		t.Fatal("existing member retry should be idempotent")
	}
	expired := createInvite(t, f, owner, team.ID, 20)
	if err := f.Users.DB.Model(&model.TeamInvite{}).Where("id = ?", expired.Invite.ID).Update("expires_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	expect(t, f.call("POST", "/teams/join", fmt.Sprintf(`{"code":%q}`, expired.Code), outsider.Cookie), 400)
	w = f.call("POST", base+"/transfer-ownership", fmt.Sprintf(`{"user_id":%q}`, strconv.FormatUint(member.ID, 10)), owner.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Team](t, w.Body.Bytes()).MyRole != "admin" {
		t.Fatal("previous owner not admin")
	}
	assertOwner(t, f, team.ID, member.ID)
	expect(t, f.call("POST", base+"/archive", "", owner.Cookie), 403)
	expect(t, f.call("POST", base+"/leave", "", owner.Cookie), 204)
	expect(t, f.call("GET", base, "", owner.Cookie), 404)
	expect(t, f.call("POST", base+"/archive", "", member.Cookie), 204)
	expect(t, f.call("GET", base, "", member.Cookie), 200)
	expect(t, f.call("GET", base+"/members", "", admin.Cookie), 200)
	expect(t, f.call("GET", base+"/activities", "", admin.Cookie), 200)
	for _, tc := range []struct{ method, path, body string }{{"PATCH", base, `{"name":"归档修改"}`}, {"POST", base + "/invites", `{}`}, {"POST", base + "/leave", ""}, {"POST", base + "/archive", ""}, {"PATCH", base + "/members/" + strconv.FormatUint(admin.ID, 10), `{"role":"member"}`}, {"DELETE", base + "/members/" + strconv.FormatUint(admin.ID, 10), ""}, {"POST", base + "/transfer-ownership", fmt.Sprintf(`{"user_id":%q}`, strconv.FormatUint(admin.ID, 10))}} {
		expect(t, f.call(tc.method, tc.path, tc.body, member.Cookie), 409)
	}
	expect(t, f.call("POST", "/teams/join", fmt.Sprintf(`{"code":%q}`, invite.Code), outsider.Cookie), 409)
	w = f.call("GET", "/teams?status=archived", "", member.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Team]](t, w.Body.Bytes()).Total != 1 {
		t.Fatal("archive filter failed")
	}
	w = f.call("GET", "/teams?status=active", "", member.Cookie)
	expect(t, w, 200)
	if dataOf[dto.Page[dto.Team]](t, w.Body.Bytes()).Total != 0 {
		t.Fatal("active filter includes archive")
	}
	var logs []model.ActivityLog
	if err := f.Users.DB.Where("team_id = ?", team.ID).Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	for _, event := range logs {
		if strings.Contains(event.Metadata, invite.Code) || strings.Contains(event.Metadata, stored.TokenHash) {
			t.Fatal("audit leaked invite credentials")
		}
	}
}

func TestInviteConcurrencyAndOwnerInvariant(t *testing.T) {
	f := setupAuth(t)
	owner, a, b := actor(t, f), actor(t, f), actor(t, f)
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 1)
	type outcome struct {
		actor  teamActor
		status int
	}
	outcomes := make(chan outcome, 2)
	var wg sync.WaitGroup
	for _, user := range []teamActor{a, b} {
		wg.Add(1)
		go func(user teamActor) {
			defer wg.Done()
			outcomes <- outcome{user, f.call("POST", "/teams/join", fmt.Sprintf(`{"code":%q}`, invite.Code), user.Cookie).Code}
		}(user)
	}
	wg.Wait()
	close(outcomes)
	success := 0
	var loser teamActor
	for result := range outcomes {
		if result.status == 200 {
			success++
		} else if result.status == 400 {
			loser = result.actor
		} else {
			t.Fatalf("unexpected join status: %d", result.status)
		}
	}
	if success != 1 {
		t.Fatal("single-use invite over-consumed")
	}
	second := createInvite(t, f, owner, team.ID, 1)
	codes := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- f.call("POST", "/teams/join", fmt.Sprintf(`{"code":%q}`, second.Code), loser.Cookie).Code
		}()
	}
	wg.Wait()
	close(codes)
	for status := range codes {
		if status != 200 {
			t.Fatalf("repeat join not idempotent: %d", status)
		}
	}
	var stored model.TeamInvite
	if err := f.Users.DB.First(&stored, second.Invite.ID).Error; err != nil || stored.UsedCount != 1 {
		t.Fatal("duplicate joins consumed multiple uses")
	}
	var memberCount, joinCount int64
	f.Users.DB.Model(&model.TeamMember{}).Where("team_id = ? AND removed_at IS NULL", team.ID).Count(&memberCount)
	f.Users.DB.Model(&model.ActivityLog{}).Where("team_id = ? AND action = 'member.joined'", team.ID).Count(&joinCount)
	if memberCount != 3 || joinCount != 2 {
		t.Fatal("join counts or audit events invalid")
	}
	results := make(chan int, 2)
	for _, target := range []teamActor{a, b} {
		wg.Add(1)
		go func(target teamActor) {
			defer wg.Done()
			results <- f.call("POST", "/teams/"+team.ID+"/transfer-ownership", fmt.Sprintf(`{"user_id":%q}`, strconv.FormatUint(target.ID, 10)), owner.Cookie).Code
		}(target)
	}
	wg.Wait()
	close(results)
	success = 0
	for status := range results {
		if status == 200 {
			success++
		} else if status != 403 {
			t.Fatalf("unexpected transfer result: %d", status)
		}
	}
	if success != 1 {
		t.Fatal("both ownership transfers succeeded")
	}
	var fresh model.Team
	if err := f.Users.DB.First(&fresh, team.ID).Error; err != nil {
		t.Fatal(err)
	}
	assertOwner(t, f, team.ID, fresh.OwnerID)
}

func TestTeamMutationAndAuditRollback(t *testing.T) {
	f := setupAuth(t)
	owner, member := actor(t, f), actor(t, f)
	failLogs := func() {
		if err := f.Users.DB.Callback().Create().Before("gorm:create").Register("test:fail_activity", func(tx *gorm.DB) {
			if tx.Statement.Table == "activity_logs" {
				tx.AddError(errors.New("injected audit failure"))
			}
		}); err != nil {
			t.Fatal(err)
		}
	}
	restore := func() { f.Users.DB.Callback().Create().Remove("test:fail_activity") }
	t.Cleanup(restore)
	failLogs()
	name := "rollback-" + unique()
	expect(t, f.call("POST", "/teams", fmt.Sprintf(`{"name":%q}`, name), owner.Cookie), 503)
	var count int64
	f.Users.DB.Model(&model.Team{}).Where("name = ?", name).Count(&count)
	if count != 0 {
		t.Fatal("team persisted despite audit failure")
	}
	restore()
	team := createTeam(t, f, owner)
	invite := createInvite(t, f, owner, team.ID, 20)
	before := logCount(t, f, team.ID)
	failLogs()
	expect(t, f.call("POST", "/teams/join", fmt.Sprintf(`{"code":%q}`, invite.Code), member.Cookie), 503)
	var stored model.TeamInvite
	if err := f.Users.DB.First(&stored, invite.Invite.ID).Error; err != nil || stored.UsedCount != 0 {
		t.Fatal("failed join consumed invite")
	}
	f.Users.DB.Model(&model.TeamMember{}).Where("team_id = ? AND user_id = ?", team.ID, member.ID).Count(&count)
	if count != 0 || logCount(t, f, team.ID) != before {
		t.Fatal("failed join partially committed")
	}
	restore()
	join(t, f, member, invite.Code)
	before = logCount(t, f, team.ID)
	failLogs()
	expect(t, f.call("PATCH", "/teams/"+team.ID+"/members/"+strconv.FormatUint(member.ID, 10), `{"role":"admin"}`, owner.Cookie), 503)
	expect(t, f.call("POST", "/teams/"+team.ID+"/transfer-ownership", fmt.Sprintf(`{"user_id":%q}`, strconv.FormatUint(member.ID, 10)), owner.Cookie), 503)
	assertOwner(t, f, team.ID, owner.ID)
	var membership model.TeamMember
	if err := f.Users.DB.Where("team_id = ? AND user_id = ?", team.ID, member.ID).First(&membership).Error; err != nil || membership.Role != "member" {
		t.Fatal("role update partially committed")
	}
	if logCount(t, f, team.ID) != before {
		t.Fatal("failed mutation left success log")
	}
}

func TestTeamPaginationAndDefaults(t *testing.T) {
	f := setupAuth(t)
	owner := actor(t, f)
	svc := &service.TeamService{Repo: &repository.TeamRepository{DB: f.Users.DB}}
	for i := 0; i < 21; i++ {
		if _, err := svc.Create(context.Background(), owner.ID, dto.CreateTeamRequest{Name: fmt.Sprintf("分页-%s-%d", unique(), i)}); err != nil {
			t.Fatal(err)
		}
	}
	w := f.call("GET", "/teams", "", owner.Cookie)
	expect(t, w, 200)
	first := dataOf[dto.Page[dto.Team]](t, w.Body.Bytes())
	if first.Total != 21 || len(first.Items) != 20 || first.PageSize != 20 {
		t.Fatal("default pagination invalid")
	}
	w = f.call("GET", "/teams?page=2", "", owner.Cookie)
	expect(t, w, 200)
	second := dataOf[dto.Page[dto.Team]](t, w.Body.Bytes())
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatal("page 2 invalid")
	}
	base := "/teams/" + first.Items[0].ID
	w = f.call("POST", base+"/invites", `{}`, owner.Cookie)
	expect(t, w, 201)
	invite := dataOf[dto.InviteCreated](t, w.Body.Bytes())
	if invite.Invite.MaxUses != 20 || time.Until(invite.Invite.ExpiresAt) < 6*24*time.Hour {
		t.Fatal("default invite configuration invalid")
	}
	expect(t, f.call("POST", base+"/invites", `{"max_uses":0}`, owner.Cookie), 400)
	expect(t, f.call("POST", base+"/invites", `{"expires_in_days":31}`, owner.Cookie), 400)
	expect(t, f.call("GET", "/teams?page_size=101", "", owner.Cookie), 400)
	expect(t, f.call("GET", "/teams?status=oops", "", owner.Cookie), 400)
	w = f.call("GET", base+"/activities?page=1&page_size=1", "", owner.Cookie)
	expect(t, w, 200)
	activity1 := dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes())
	w = f.call("GET", base+"/activities?page=2&page_size=1", "", owner.Cookie)
	expect(t, w, 200)
	activity2 := dataOf[dto.Page[dto.Activity]](t, w.Body.Bytes())
	if activity1.Total != 2 || len(activity2.Items) != 1 || activity1.Items[0].ID == activity2.Items[0].ID {
		t.Fatal("audit pagination invalid")
	}
}
