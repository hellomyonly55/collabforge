package router

import (
	"github.com/gin-gonic/gin"

	"taskflow/internal/handler"
	"taskflow/internal/middleware"
)

func RegisterAdmin(r *gin.Engine, auth *handler.AuthHandler, admin *handler.AdminHandler) {
	g := r.Group("/api/v1/admin", middleware.RequireAuth(auth.Service), middleware.RequireAdmin())
	g.GET("/statistics", admin.Statistics)
	g.GET("/users", admin.Users)
	g.PATCH("/users/:userId/status", admin.Status)
	g.GET("/teams", admin.Teams)
	g.POST("/teams/:teamId/archive", admin.Archive)
}
