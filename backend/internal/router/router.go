package router

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"taskflow/internal/handler"
	"taskflow/internal/middleware"
	"time"
)

func New(health *handler.HealthHandler, log *slog.Logger, origin string, auth ...*handler.AuthHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.RemoteIPHeaders = []string{"X-Real-IP"}
	r.Use(middleware.RequestID(), middleware.AccessLog(log), middleware.Recovery(log), middleware.Origin(origin))
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() })
	v1.GET("/health/live", health.Live)
	v1.GET("/health/ready", health.Ready)
	if len(auth) > 0 && auth[0] != nil {
		h := auth[0]
		v1.POST("/auth/register", middleware.RateLimit(h.Service.Sessions, "register:ip", 20, time.Hour, false), h.Register)
		v1.POST("/auth/login", middleware.RateLimit(h.Service.Sessions, "login:ip", 60, 15*time.Minute, false), h.Login)
		v1.POST("/auth/logout", h.Logout)
		protected := v1.Group("")
		protected.Use(middleware.RequireAuth(h.Service))
		protected.GET("/me", h.Me)
		protected.PATCH("/me", h.Profile)
		protected.PUT("/me/password", middleware.RateLimit(h.Service.Sessions, "password:user", 5, 15*time.Minute, true), h.Password)
	}
	r.NoRoute(func(c *gin.Context) { handler.Error(c, 404, "NOT_FOUND", "接口不存在或尚未实现") })
	return r
}
