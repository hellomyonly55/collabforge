package middleware

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"taskflow/internal/handler"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"taskflow/internal/service"
	"time"
)

func RequireAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(handler.SessionCookie)
		u, err := auth.Authenticate(c.Request.Context(), token)
		if err != nil {
			handler.RespondError(c, err)
			return
		}
		c.Set("user", u)
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}
func RateLimit(sessions *repository.SessionRepository, scope string, limit int, window time.Duration, byUser bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if byUser {
			key = strconv.FormatUint(c.MustGet("user").(*model.User).ID, 10)
		}
		allowed, err := sessions.Allow(c.Request.Context(), scope+":"+key, limit, window)
		if err != nil {
			handler.Error(c, 503, "SERVICE_UNAVAILABLE", "服务暂不可用，请稍后重试")
			return
		}
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			handler.Error(c, 429, "RATE_LIMITED", "操作过于频繁，请稍后重试")
			return
		}
		c.Next()
	}
}
