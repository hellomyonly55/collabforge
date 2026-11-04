package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"taskflow/internal/handler"
	"taskflow/internal/service"
	"testing"
)

type checker struct{ mysql, redis, migrations bool }

func result(ok bool) error {
	if ok {
		return nil
	}
	return errors.New("sensitive database error")
}
func (c checker) MySQL(context.Context) error      { return result(c.mysql) }
func (c checker) RedisPing(context.Context) error  { return result(c.redis) }
func (c checker) Migrations(context.Context) error { return result(c.migrations) }
func TestHealthResponses(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		check      checker
		status     int
		state      string
	}{{"live without dependencies", "/api/v1/health/live", checker{}, 200, "alive"}, {"ready", "/api/v1/health/ready", checker{true, true, true}, 200, "ready"}, {"mysql failure", "/api/v1/health/ready", checker{false, true, true}, 503, "not_ready"}, {"redis failure", "/api/v1/health/ready", checker{true, false, true}, 503, "not_ready"}, {"pending migration", "/api/v1/health/ready", checker{true, true, false}, 503, "not_ready"}} {
		t.Run(tc.name, func(t *testing.T) {
			r := New(&handler.HealthHandler{Service: &service.HealthService{Checker: tc.check}}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			var b struct {
				Data struct {
					Status string `json:"status"`
				}
				RequestID string `json:"request_id"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
				t.Fatal(err)
			}
			if b.Data.Status != tc.state || b.RequestID == "" || b.RequestID != w.Header().Get("X-Request-ID") {
				t.Fatalf("invalid response: %s", w.Body.String())
			}
		})
	}
}
func TestOriginAndNotFound(t *testing.T) {
	r := New(&handler.HealthHandler{Service: &service.HealthService{Checker: checker{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173")
	for _, tc := range []struct {
		origin string
		status int
	}{{"https://attacker.example", 403}, {"http://localhost:5173", 404}, {"", 404}} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/not-implemented", nil)
		req.Header.Set("Origin", tc.origin)
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("unexpected status: %d", w.Code)
		}
	}
}
