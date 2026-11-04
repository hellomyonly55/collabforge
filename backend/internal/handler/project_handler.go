package handler

import (
	"github.com/gin-gonic/gin"
	"taskflow/internal/dto"
	"taskflow/internal/service"
)

type ProjectHandler struct{ Service *service.ProjectService }

func projectID(c *gin.Context) (uint64, bool) { return parseID(c, c.Param("projectId")) }
func (h *ProjectHandler) List(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.List(c.Request.Context(), id, currentUser(c).ID, c.DefaultQuery("status", "all"), p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *ProjectHandler) Create(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	var req dto.CreateProjectRequest
	if !decode(c, &req) {
		return
	}
	data, err := h.Service.Create(c.Request.Context(), id, currentUser(c).ID, req)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 201, data)
}
func (h *ProjectHandler) Get(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	data, err := h.Service.Get(c.Request.Context(), id, currentUser(c).ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *ProjectHandler) Update(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var req dto.UpdateProjectRequest
	if !decode(c, &req) {
		return
	}
	data, err := h.Service.Update(c.Request.Context(), id, currentUser(c).ID, req)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *ProjectHandler) Archive(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	if err := h.Service.Archive(c.Request.Context(), id, currentUser(c).ID); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
func (h *ProjectHandler) Activities(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	p, ok := pagination(c)
	if !ok {
		return
	}
	filter, ok := activityFilters(c)
	if !ok {
		return
	}
	data, err := h.Service.Activities(c.Request.Context(), id, currentUser(c).ID, p, filter)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
