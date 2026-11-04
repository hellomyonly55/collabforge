package router

import (
	"time"

	"github.com/gin-gonic/gin"

	"taskflow/internal/handler"
	"taskflow/internal/middleware"
)

func RegisterTasks(r *gin.Engine, auth *handler.AuthHandler, tasks *handler.TaskHandler) {
	g := r.Group("/api/v1", func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() }, middleware.RequireAuth(auth.Service))
	g.GET("/projects/:projectId/tasks", tasks.List)
	g.POST("/projects/:projectId/tasks", middleware.RateLimit(auth.Service.Sessions, "task:create", 300, time.Hour, true), tasks.Create)
	g.GET("/projects/:projectId/board", tasks.Board)
	g.GET("/tasks/:taskId", tasks.Get)
	g.PATCH("/tasks/:taskId", tasks.Update)
	g.POST("/tasks/:taskId/move", tasks.Move)
	g.DELETE("/tasks/:taskId", tasks.Delete)
	g.GET("/me/tasks", tasks.Mine)
}
