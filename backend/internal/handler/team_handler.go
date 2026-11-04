package handler

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"taskflow/internal/dto"
	"taskflow/internal/service"
)

type TeamHandler struct{ Service *service.TeamService }

func parseID(c *gin.Context, value string) (uint64, bool) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		Error(c, 400, "VALIDATION_ERROR", "ID 必须是正整数")
		return 0, false
	}
	return id, true
}
func pagination(c *gin.Context) (dto.Pagination, bool) {
	p := dto.Pagination{Page: 1, PageSize: 20}
	for _, v := range []struct {
		key string
		dst *int
		max int
	}{{"page", &p.Page, 1000000}, {"page_size", &p.PageSize, 100}} {
		if raw, exists := c.GetQuery(v.key); exists {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > v.max {
				Error(c, 400, "VALIDATION_ERROR", "分页参数无效")
				return p, false
			}
			*v.dst = n
		}
	}
	return p, true
}
func teamID(c *gin.Context) (uint64, bool) { return parseID(c, c.Param("teamId")) }
func (h *TeamHandler) List(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.List(c.Request.Context(), currentUser(c).ID, c.DefaultQuery("status", "all"), p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *TeamHandler) Create(c *gin.Context) {
	var req dto.CreateTeamRequest
	if !decode(c, &req) {
		return
	}
	data, err := h.Service.Create(c.Request.Context(), currentUser(c).ID, req)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 201, data)
}
func (h *TeamHandler) Get(c *gin.Context) {
	id, ok := teamID(c)
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
func (h *TeamHandler) Update(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	var req dto.UpdateTeamRequest
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
func (h *TeamHandler) Members(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.Members(c.Request.Context(), id, currentUser(c).ID, p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *TeamHandler) Role(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	userID, ok := parseID(c, c.Param("userId"))
	if !ok {
		return
	}
	var req dto.RoleRequest
	if !decode(c, &req) {
		return
	}
	if err := h.Service.SetRole(c.Request.Context(), id, currentUser(c).ID, userID, req.Role); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
func (h *TeamHandler) Remove(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	userID, ok := parseID(c, c.Param("userId"))
	if !ok {
		return
	}
	if err := h.Service.Remove(c.Request.Context(), id, currentUser(c).ID, userID); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
func (h *TeamHandler) Leave(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	if err := h.Service.Leave(c.Request.Context(), id, currentUser(c).ID); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
func (h *TeamHandler) Transfer(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	var req dto.TransferRequest
	if !decode(c, &req) {
		return
	}
	userID, ok := parseID(c, req.UserID)
	if !ok {
		return
	}
	data, err := h.Service.Transfer(c.Request.Context(), id, currentUser(c).ID, userID)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *TeamHandler) Archive(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	if err := h.Service.Archive(c.Request.Context(), id, currentUser(c).ID); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
func (h *TeamHandler) CreateInvite(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	var req dto.CreateInviteRequest
	if !decode(c, &req) {
		return
	}
	data, err := h.Service.CreateInvite(c.Request.Context(), id, currentUser(c).ID, req)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 201, data)
}
func (h *TeamHandler) Invites(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	p, ok := pagination(c)
	if !ok {
		return
	}
	data, err := h.Service.Invites(c.Request.Context(), id, currentUser(c).ID, p)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *TeamHandler) RevokeInvite(c *gin.Context) {
	id, ok := teamID(c)
	if !ok {
		return
	}
	inviteID, ok := parseID(c, c.Param("inviteId"))
	if !ok {
		return
	}
	if err := h.Service.RevokeInvite(c.Request.Context(), id, currentUser(c).ID, inviteID); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
func (h *TeamHandler) Join(c *gin.Context) {
	var req dto.JoinRequest
	if !decode(c, &req) {
		return
	}
	data, err := h.Service.Join(c.Request.Context(), currentUser(c).ID, req.Code)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, data)
}
func (h *TeamHandler) Activities(c *gin.Context) {
	id, ok := teamID(c)
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
