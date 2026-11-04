package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"taskflow/internal/dto"
	"taskflow/internal/service"
)

type HealthHandler struct{ Service *service.HealthService }

func (h *HealthHandler) Live(c *gin.Context) { Success(c, http.StatusOK, dto.Health{Status: "alive"}) }
func (h *HealthHandler) Ready(c *gin.Context) {
	result, ok := h.Service.Ready(c.Request.Context())
	status := http.StatusOK
	if !ok {
		status = http.StatusServiceUnavailable
	}
	Success(c, status, result)
}
