package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"taskflow/internal/dto"
	"taskflow/internal/service"
)

type TaskHandler struct{ Service *service.TaskService }

func taskID(c *gin.Context) (uint64, bool) { return parseID(c, c.Param("taskId")) }
func taskFilters(c *gin.Context, mine bool) (dto.TaskFilter, bool) {
	f := dto.TaskFilter{Keyword: service.TrimTaskKeyword(c.Query("keyword")), Status: c.Query("status"), Priority: c.Query("priority")}
	var err error
	if mine && (c.Query("assignee_id") != "" || c.Query("revision") != "") {
		Error(c, 400, "VALIDATION_ERROR", "我的任务不支持负责人或看板快照参数")
		return f, false
	}
	f.AssigneeID, f.Unassigned, err = service.ParseTaskAssigneeFilter(c.Query("assignee_id"))
	if err != nil {
		RespondError(c, err)
		return f, false
	}
	if raw, exists := c.GetQuery("overdue"); exists {
		if raw != "true" && raw != "false" {
			Error(c, 400, "VALIDATION_ERROR", "overdue 必须为 true 或 false")
			return f, false
		}
		value := raw == "true"
		f.Overdue = &value
	}
	if raw, exists := c.GetQuery("revision"); exists {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			Error(c, 400, "VALIDATION_ERROR", "看板快照参数无效")
			return f, false
		}
		f.Revision = &value
	}
	return f, true
}
func (h *TaskHandler) Create(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var req dto.CreateTaskRequest
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
func (h *TaskHandler) Get(c *gin.Context) {
	id, ok := taskID(c)
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
func (h *TaskHandler) Update(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req dto.UpdateTaskRequest
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
func (h *TaskHandler) Move(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req dto.MoveTaskRequest
	if !decode(c, &req) {
		return
	}
	data, err := h.Service.Move(c.Request.Context(), id, currentUser(c).ID, req)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *TaskHandler) Delete(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	version, err := strconv.ParseUint(c.Query("version"), 10, 64)
	if err != nil || version == 0 {
		Error(c, 400, "VALIDATION_ERROR", "必须提供有效任务版本")
		return
	}
	if err = h.Service.Delete(c.Request.Context(), id, currentUser(c).ID, version); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
func (h *TaskHandler) List(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	f, ok := taskFilters(c, false)
	if !ok {
		return
	}
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.List(c.Request.Context(), id, currentUser(c).ID, f, p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *TaskHandler) Mine(c *gin.Context) {
	f, ok := taskFilters(c, true)
	if !ok {
		return
	}
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.Mine(c.Request.Context(), currentUser(c).ID, f, p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *TaskHandler) Board(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	f, ok := taskFilters(c, false)
	if !ok {
		return
	}
	data, err := h.Service.Board(c.Request.Context(), id, currentUser(c).ID, f)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
