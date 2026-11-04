package handler

import (
	"github.com/gin-gonic/gin"

	"taskflow/internal/dto"
	"taskflow/internal/service"
)

type AdminHandler struct{ Service *service.AdminService }

func adminFilters(c *gin.Context) dto.AdminFilter {
	return dto.AdminFilter{Keyword: service.AdminKeyword(c.Query("keyword")), Status: c.DefaultQuery("status", "all"), Role: c.DefaultQuery("role", "all")}
}
func (h *AdminHandler) Statistics(c *gin.Context) {
	data, err := h.Service.Statistics(c.Request.Context(), currentUser(c))
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *AdminHandler) Users(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.Users(c.Request.Context(), currentUser(c), adminFilters(c), p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *AdminHandler) Teams(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.Teams(c.Request.Context(), currentUser(c), adminFilters(c), p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *AdminHandler) Status(c *gin.Context) {
	id, ok := parseID(c, c.Param("userId"))
	if !ok {
		return
	}
	var req dto.UserStatusRequest
	if !decode(c, &req) {
		return
	}
	data, err := h.Service.Status(c.Request.Context(), currentUser(c), id, req)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *AdminHandler) Archive(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	var req dto.AdminArchiveRequest
	if !decode(c, &req) {
		return
	}
	if err := h.Service.Archive(c.Request.Context(), currentUser(c), id, req); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
