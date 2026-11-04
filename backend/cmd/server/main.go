package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"taskflow/internal/config"
	"taskflow/internal/handler"
	"taskflow/internal/infrastructure/database"
	"taskflow/internal/infrastructure/logger"
	redisinfra "taskflow/internal/infrastructure/redis"
	"taskflow/internal/repository"
	"taskflow/internal/router"
	"taskflow/internal/service"
	"time"
)

func main() { os.Exit(run()) }
func run() int {
	log := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration", "error", err.Error())
		return 1
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Error("cannot initialize database client; check database configuration")
		return 1
	}
	defer db.SQL.Close()
	cache := redisinfra.New(cfg.RedisAddr)
	defer cache.Close()
	health := &handler.HealthHandler{Service: &service.HealthService{Checker: &repository.HealthRepository{DB: db.SQL, Redis: cache}}}
	authService, err := service.NewAuthService(&repository.UserRepository{DB: db.ORM}, repository.NewSessionRepository(cache, "taskflow:"), log)
	if err != nil {
		log.Error("cannot initialize password hashing")
		return 1
	}
	auth := &handler.AuthHandler{Service: authService, SecureCookie: cfg.SecureCookie}
	routes := router.New(health, log, cfg.AppOrigin, auth)
	teamService := &service.TeamService{Repo: &repository.TeamRepository{DB: db.ORM}}
	router.RegisterTeams(routes, auth, &handler.TeamHandler{Service: teamService})
	projectService := &service.ProjectService{Teams: teamService, Repo: &repository.ProjectRepository{DB: db.ORM}}
	router.RegisterProjects(routes, auth, &handler.ProjectHandler{Service: projectService})
	tasks := &service.TaskService{Projects: projectService, Repo: &repository.TaskRepository{DB: db.ORM}}
	router.RegisterTasks(routes, auth, &handler.TaskHandler{Service: tasks})
	router.RegisterComments(routes, auth, &handler.CommentHandler{Service: &service.CommentService{Tasks: tasks, Repo: &repository.CommentRepository{DB: db.ORM}}})
	router.RegisterStatistics(routes, auth, &handler.StatisticsHandler{Service: &service.StatisticsService{Projects: projectService}})
	router.RegisterAdmin(routes, auth, &handler.AdminHandler{Service: &service.AdminService{Repo: &repository.AdminRepository{DB: db.ORM}, Auth: authService}})
	if err = routes.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Error("invalid trusted proxies")
		return 1
	}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: routes, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	failed := make(chan error, 1)
	go func() { log.Info("server listening", "address", cfg.HTTPAddr); failed <- server.ListenAndServe() }()
	select {
	case err := <-failed:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server failed")
			return 1
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		log.Error("graceful shutdown failed")
		return 1
	}
	log.Info("server stopped")
	return 0
}
