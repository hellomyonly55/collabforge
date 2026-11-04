package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"taskflow/internal/handler"
	"time"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			handler.Error(c, 500, "INTERNAL_ERROR", "服务暂不可用")
			return
		}
		id := hex.EncodeToString(b[:])
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("http request", "request_id", c.GetString("request_id"), "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds())
	}
}
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				log.Error("request panic", "request_id", c.GetString("request_id"))
				handler.Error(c, 500, "INTERNAL_ERROR", "服务暂不可用")
			}
		}()
		c.Next()
	}
}
func Origin(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.Request.Method != http.MethodOptions {
			// Non-browser clients may omit Origin. Browser requests with an origin must match exactly.
			if value := c.GetHeader("Origin"); value != "" && value != origin {
				handler.Error(c, 403, "FORBIDDEN", "请求来源不受信任")
				return
			}
			if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
				handler.Error(c, 403, "FORBIDDEN", "请求来源不受信任")
				return
			}
		}
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}
