package integration

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"taskflow/internal/config"
	"taskflow/internal/handler"
	"taskflow/internal/infrastructure/database"
	redisinfra "taskflow/internal/infrastructure/redis"
	"taskflow/internal/repository"
	"taskflow/internal/router"
	"taskflow/internal/service"
	"taskflow/migrations"
	"testing"
	"time"
)

// This test intentionally does not drop tables or delete any records.
func TestRealDependenciesAndIdempotentMigrations(t *testing.T) {
	if os.Getenv("TASKFLOW_INTEGRATION") != "1" {
		t.Skip("set TASKFLOW_INTEGRATION=1 and configure real MySQL/Redis")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatal("database connection setup failed")
	}
	defer db.SQL.Close()
	cache := redisinfra.New(cfg.RedisAddr)
	defer cache.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		if err := migrations.Run(ctx, db.SQL); err != nil {
			t.Fatalf("migration run %d failed: %v", i, err)
		}
	}
	var count int
	if err := db.SQL.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migrations.LatestVersion).Scan(&count); err != nil || count != 1 {
		t.Fatalf("migration version must occur exactly once: count=%d error=%v", count, err)
	}
	h := &handler.HealthHandler{Service: &service.HealthService{Checker: &repository.HealthRepository{DB: db.SQL, Redis: cache}}}
	r := router.New(h, slog.New(slog.NewTextHandler(io.Discard, nil)), cfg.AppOrigin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/health/ready", nil))
	if w.Code != 200 {
		t.Fatalf("real readiness failed: %s", w.Body.String())
	}
}
