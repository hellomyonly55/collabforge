package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"taskflow/internal/config"
	"taskflow/internal/handler"
	"taskflow/internal/infrastructure/database"
	"taskflow/internal/infrastructure/password"
	redisinfra "taskflow/internal/infrastructure/redis"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"taskflow/internal/router"
	"taskflow/internal/service"
	"taskflow/migrations"
	"testing"
	"time"
)

type authFixture struct {
	Router   *gin.Engine
	Users    *repository.UserRepository
	Sessions *repository.SessionRepository
	Auth     *service.AuthService
}

func unique() string { var b [10]byte; _, _ = rand.Read(b[:]); return hex.EncodeToString(b[:]) }
func setupAuth(t *testing.T) *authFixture {
	t.Helper()
	if os.Getenv("TASKFLOW_INTEGRATION") != "1" {
		t.Skip("run scripts/test-integration.sh for isolated MySQL/Redis tests")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(cfg.Database.Name, "_test") {
		t.Fatal("auth integration tests require a database name ending in _test")
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatal("database setup failed")
	}
	t.Cleanup(func() { _ = db.SQL.Close() })
	cache := redisinfra.New(cfg.RedisAddr)
	t.Cleanup(func() { _ = cache.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = migrations.Run(ctx, db.SQL); err != nil {
		t.Fatal(err)
	}
	sessions := repository.NewSessionRepository(cache, "taskflow:test:"+unique()+":")
	users := &repository.UserRepository{DB: db.ORM}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	auth, err := service.NewAuthService(users, sessions, log)
	if err != nil {
		t.Fatal(err)
	}
	health := &handler.HealthHandler{Service: &service.HealthService{Checker: &repository.HealthRepository{DB: db.SQL, Redis: cache}}}
	routes := router.New(health, log, "http://localhost:5173", &handler.AuthHandler{Service: auth})
	teams := &service.TeamService{Repo: &repository.TeamRepository{DB: db.ORM}}
	router.RegisterTeams(routes, &handler.AuthHandler{Service: auth}, &handler.TeamHandler{Service: teams})
	projects := &service.ProjectService{Teams: teams, Repo: &repository.ProjectRepository{DB: db.ORM}}
	router.RegisterProjects(routes, &handler.AuthHandler{Service: auth}, &handler.ProjectHandler{Service: projects})
	tasks := &service.TaskService{Projects: projects, Repo: &repository.TaskRepository{DB: db.ORM}}
	router.RegisterTasks(routes, &handler.AuthHandler{Service: auth}, &handler.TaskHandler{Service: tasks})
	router.RegisterComments(routes, &handler.AuthHandler{Service: auth}, &handler.CommentHandler{Service: &service.CommentService{Tasks: tasks, Repo: &repository.CommentRepository{DB: db.ORM}}})
	router.RegisterStatistics(routes, &handler.AuthHandler{Service: auth}, &handler.StatisticsHandler{Service: &service.StatisticsService{Projects: projects}})
	router.RegisterAdmin(routes, &handler.AuthHandler{Service: auth}, &handler.AdminHandler{Service: &service.AdminService{Repo: &repository.AdminRepository{DB: db.ORM}, Auth: auth}})
	return &authFixture{routes, users, sessions, auth}
}
func (f *authFixture) call(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	req.RemoteAddr = "127.0.0.1:12345"
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	f.Router.ServeHTTP(w, req)
	return w
}
func expect(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("expected %d, got %d: %s", status, w.Code, w.Body.String())
	}
}
func userBody(email, pwd string) string {
	return fmt.Sprintf(`{"email":%q,"nickname":"测试同学","password":%q}`, email, pwd)
}
func loginBody(email, pwd string) string {
	return fmt.Sprintf(`{"email":%q,"password":%q}`, email, pwd)
}
func login(t *testing.T, f *authFixture, email, pwd string) *http.Cookie {
	t.Helper()
	w := f.call("POST", "/auth/login", loginBody(email, pwd), nil)
	expect(t, w, 200)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("expected one session cookie")
	}
	return cookies[0]
}

func TestAuthLifecycle(t *testing.T) {
	f := setupAuth(t)
	ctx := context.Background()
	email := "auth-" + unique() + "@example.test"
	pwd := "initial-password-123"
	next := "updated-password-456"
	expect(t, f.call("GET", "/me", "", nil), 401)
	w := f.call("POST", "/auth/register", userBody(email, pwd), nil)
	expect(t, w, 201)
	var envelope struct{ Data dtoUser }
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.ID == "" || envelope.Data.PlatformRole != "user" {
		t.Fatal("user DTO invalid")
	}
	if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "session_version") {
		t.Fatal("private fields leaked")
	}
	u, err := f.Users.ByEmail(ctx, email)
	if err != nil || !password.Matches(u.PasswordHash, pwd) || u.PasswordHash == pwd {
		t.Fatal("password was not stored as bcrypt hash")
	}
	expect(t, f.call("POST", "/auth/register", userBody(strings.ToUpper(email), pwd), nil), 409)
	expect(t, f.call("POST", "/auth/register", fmt.Sprintf(`{"email":%q,"nickname":"hi","password":%q,"platform_role":"admin"}`, "role-"+unique()+"@example.test", pwd), nil), 400)
	expect(t, f.call("POST", "/auth/register", userBody("invalid-"+unique()+"@example.test", strings.Repeat("汉", 25)), nil), 400)
	expect(t, f.call("POST", "/auth/login", loginBody(email, "incorrect-password"), nil), 401)
	first := login(t, f, email, pwd)
	if !first.HttpOnly || first.SameSite != http.SameSiteLaxMode || first.Path != "/" || first.MaxAge != int(repository.SessionTTL.Seconds()) || first.Secure {
		t.Fatal("local session cookie flags invalid")
	}
	expect(t, f.call("GET", "/me", "", first), 200)
	expect(t, f.call("PATCH", "/me", `{"nickname":"更新后的昵称"}`, first), 200)
	u, err = f.Users.ByEmail(ctx, email)
	if err != nil || u.Nickname != "更新后的昵称" {
		t.Fatal("profile did not persist")
	}
	second := login(t, f, email, pwd)
	expect(t, f.call("POST", "/auth/logout", "", first), 204)
	expect(t, f.call("GET", "/me", "", first), 401)
	expect(t, f.call("GET", "/me", "", second), 200)
	expect(t, f.call("PUT", "/me/password", `{"old_password":"incorrect-password","new_password":"updated-password-456"}`, second), 400)
	third := login(t, f, email, pwd)
	expect(t, f.call("PUT", "/me/password", fmt.Sprintf(`{"old_password":%q,"new_password":%q}`, pwd, next), second), 204)
	expect(t, f.call("GET", "/me", "", second), 401)
	expect(t, f.call("GET", "/me", "", third), 401)
	if count, err := f.Sessions.Client.ZCard(ctx, f.Sessions.Prefix+"user_sessions:"+envelope.Data.ID).Result(); err != nil || count != 0 {
		t.Fatal("session index not revoked")
	}
	expect(t, f.call("POST", "/auth/login", loginBody(email, pwd), nil), 401)
	current := login(t, f, email, next)
	// Disabling is simulated directly in the isolated test DB; the admin UI belongs to stage 7.
	if err = f.Users.DB.Model(&model.User{}).Where("email = ?", email).Update("status", "disabled").Error; err != nil {
		t.Fatal(err)
	}
	expect(t, f.call("GET", "/me", "", current), 401)
	expect(t, f.call("POST", "/auth/login", loginBody(email, next), nil), 401)
	expect(t, f.call("POST", "/auth/logout", "", current), 204)
	expect(t, f.call("POST", "/auth/logout", "", nil), 204)
}

type dtoUser struct {
	ID           string `json:"id"`
	PlatformRole string `json:"platform_role"`
}

func TestRateLimitAndInputSecurity(t *testing.T) {
	f := setupAuth(t)
	email := "unknown-" + unique() + "@example.test"
	for i := 0; i < 10; i++ {
		expect(t, f.call("POST", "/auth/login", loginBody(email, "incorrect-password"), nil), 401)
	}
	expect(t, f.call("POST", "/auth/login", loginBody(email, "incorrect-password"), nil), 429)
	w := f.call("POST", "/auth/register", strings.Repeat("x", 17000), nil)
	expect(t, w, 400)
	// Separate IP counter: 20 register attempts, then 429. Invalid input never creates an account.
	for i := 1; i < 20; i++ {
		expect(t, f.call("POST", "/auth/register", `{}`, nil), 400)
	}
	expect(t, f.call("POST", "/auth/register", `{}`, nil), 429)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(loginBody(email, "incorrect-password")))
	req.Header.Set("Origin", "https://evil.example")
	w = httptest.NewRecorder()
	f.Router.ServeHTTP(w, req)
	expect(t, w, 403)
}
func TestSessionExpiryAndVersionAuthority(t *testing.T) {
	f := setupAuth(t)
	ctx := context.Background()
	email := "expiry-" + unique() + "@example.test"
	pwd := "initial-password-123"
	expect(t, f.call("POST", "/auth/register", userBody(email, pwd), nil), 201)
	cookie := login(t, f, email, pwd)
	u, err := f.Users.ByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	// Leave Redis key intact: a DB version change must still invalidate the old token.
	if err = f.Users.DB.Model(u).UpdateColumn("session_version", u.SessionVersion+1).Error; err != nil {
		t.Fatal(err)
	}
	expect(t, f.call("GET", "/me", "", cookie), 401)
	short := repository.NewSessionRepository(f.Sessions.Client, f.Sessions.Prefix)
	short.TTL = time.Second
	token, err := short.Create(ctx, u.ID, u.SessionVersion)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err = short.Lookup(ctx, token); err != repository.ErrSessionNotFound {
		t.Fatal("expired session still exists")
	}
}
func TestConcurrentPasswordChanges(t *testing.T) {
	f := setupAuth(t)
	ctx := context.Background()
	email := "race-" + unique() + "@example.test"
	pwd := "initial-password-123"
	expect(t, f.call("POST", "/auth/register", userBody(email, pwd), nil), 201)
	u, err := f.Users.ByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := f.Users.UpdatePassword(ctx, u, u.PasswordHash)
			if err != nil {
				t.Errorf("password update failed: %v", err)
				results <- false
				return
			}
			results <- ok
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for ok := range results {
		if ok {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("expected one winner, got %d", success)
	}
}

func TestLoginRotationAndSecureCookie(t *testing.T) {
	f := setupAuth(t)
	email := "cookie-" + unique() + "@example.test"
	pwd := "initial-password-123"
	expect(t, f.call("POST", "/auth/register", userBody(email, pwd), nil), 201)
	first := login(t, f, email, pwd)
	w := f.call("POST", "/auth/login", loginBody(email, pwd), first)
	expect(t, w, 200)
	second := w.Result().Cookies()[0]
	if first.Value == second.Value {
		t.Fatal("login did not rotate token")
	}
	expect(t, f.call("GET", "/me", "", first), 401)
	expect(t, f.call("GET", "/me", "", second), 200)
	h := &handler.AuthHandler{Service: f.Auth, SecureCookie: true}
	secureRoutes := gin.New()
	secureRoutes.POST("/login", h.Login)
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody(email, pwd)))
	w = httptest.NewRecorder()
	secureRoutes.ServeHTTP(w, req)
	expect(t, w, 200)
	if !w.Result().Cookies()[0].Secure {
		t.Fatal("HTTPS config did not enable Secure cookie")
	}
	// A credential with an unknown token must fail without touching unrelated sessions.
	invalid := &http.Cookie{Name: handler.SessionCookie, Value: strings.Repeat("a", 64)}
	expect(t, f.call("GET", "/me", "", invalid), 401)
}
