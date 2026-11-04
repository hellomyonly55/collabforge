package router

import (
	"time"

	"github.com/gin-gonic/gin"

	"taskflow/internal/handler"
	"taskflow/internal/middleware"
)

func RegisterComments(r *gin.Engine, auth *handler.AuthHandler, comments *handler.CommentHandler) {
	g := r.Group("/api/v1", func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() }, middleware.RequireAuth(auth.Service))
	g.GET("/tasks/:taskId/comments", comments.List)
	g.POST("/tasks/:taskId/comments", middleware.RateLimit(auth.Service.Sessions, "comment:create", 120, time.Hour, true), comments.Create)
	g.PATCH("/comments/:commentId", comments.Update)
	g.DELETE("/comments/:commentId", comments.Delete)
}
