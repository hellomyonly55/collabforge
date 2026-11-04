package router

import (
	"github.com/gin-gonic/gin"
	"taskflow/internal/handler"
	"taskflow/internal/middleware"
	"time"
)

func RegisterProjects(r *gin.Engine, auth *handler.AuthHandler, project *handler.ProjectHandler) {
	g := r.Group("/api/v1", func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() }, middleware.RequireAuth(auth.Service))
	g.GET("/teams/:teamId/projects", project.List)
	g.POST("/teams/:teamId/projects", middleware.RateLimit(auth.Service.Sessions, "project:create", 60, time.Hour, true), project.Create)
	g.GET("/projects/:projectId", project.Get)
	g.PATCH("/projects/:projectId", project.Update)
	g.POST("/projects/:projectId/archive", project.Archive)
	g.GET("/projects/:projectId/activities", project.Activities)
}
