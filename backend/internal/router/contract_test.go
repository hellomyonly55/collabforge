package router

import (
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"taskflow/internal/handler"
	"taskflow/internal/service"
)

// Route registration is metadata-only here; real HTTP behavior is covered with MySQL/Redis.
func TestPublishedAPIContractMatchesRegisteredRoutes(t *testing.T) {
	auth := &handler.AuthHandler{Service: &service.AuthService{}}
	r := New(&handler.HealthHandler{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:5173", auth)
	RegisterTeams(r, auth, &handler.TeamHandler{})
	RegisterProjects(r, auth, &handler.ProjectHandler{})
	RegisterTasks(r, auth, &handler.TaskHandler{})
	RegisterComments(r, auth, &handler.CommentHandler{})
	RegisterStatistics(r, auth, &handler.StatisticsHandler{})
	RegisterAdmin(r, auth, &handler.AdminHandler{})
	raw, err := os.ReadFile("../../../docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err = yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	published := map[string]bool{}
	for path, operations := range doc.Paths {
		for method := range operations {
			switch method {
			case "get", "post", "put", "patch", "delete":
				published[strings.ToUpper(method)+" "+path] = true
			}
		}
	}
	parameter := regexp.MustCompile(`:([^/]+)`)
	for _, route := range r.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		key := route.Method + " " + parameter.ReplaceAllString(strings.TrimPrefix(route.Path, "/api/v1"), "{$1}")
		if !published[key] {
			t.Errorf("registered route absent from OpenAPI: %s", key)
		}
		delete(published, key)
	}
	for key := range published {
		t.Errorf("documented route not registered: %s", key)
	}
}
