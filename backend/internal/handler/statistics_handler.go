package handler

import (
	"github.com/gin-gonic/gin"

	"taskflow/internal/service"
)

type StatisticsHandler struct{ Service *service.StatisticsService }

func (h *StatisticsHandler) Team(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	data, err := h.Service.Team(c.Request.Context(), id, currentUser(c).ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *StatisticsHandler) Project(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	data, err := h.Service.Project(c.Request.Context(), id, currentUser(c).ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
