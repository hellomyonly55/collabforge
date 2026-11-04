package middleware

import (
	"github.com/gin-gonic/gin"

	"taskflow/internal/handler"
	"taskflow/internal/model"
	"taskflow/internal/service"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := service.RequirePlatformAdmin(c.MustGet("user").(*model.User)); err != nil {
			handler.RespondError(c, err)
			return
		}
		c.Next()
	}
}
