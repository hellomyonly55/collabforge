package router

import (
	"github.com/gin-gonic/gin"

	"taskflow/internal/handler"
	"taskflow/internal/middleware"
)

func RegisterStatistics(r *gin.Engine, auth *handler.AuthHandler, statistics *handler.StatisticsHandler) {
	g := r.Group("/api/v1", func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() }, middleware.RequireAuth(auth.Service))
	g.GET("/teams/:teamId/statistics", statistics.Team)
	g.GET("/projects/:projectId/statistics", statistics.Project)
}
