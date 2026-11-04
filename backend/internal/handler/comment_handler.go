package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"taskflow/internal/dto"
	"taskflow/internal/service"
)

type CommentHandler struct{ Service *service.CommentService }

func (h *CommentHandler) List(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.List(c.Request.Context(), id, currentUser(c).ID, p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *CommentHandler) Create(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req dto.CreateCommentRequest
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
func (h *CommentHandler) Update(c *gin.Context) {
	id, ok := parseID(c, c.Param("commentId"))
	if !ok {
		return
	}
	var req dto.UpdateCommentRequest
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
func (h *CommentHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, c.Param("commentId"))
	if !ok {
		return
	}
	version, err := strconv.ParseUint(c.Query("version"), 10, 64)
	if err != nil || version == 0 {
		Error(c, 400, "VALIDATION_ERROR", "必须提供有效评论版本")
		return
	}
	if err = h.Service.Delete(c.Request.Context(), id, currentUser(c).ID, version); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
