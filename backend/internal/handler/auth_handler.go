package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"taskflow/internal/dto"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	"taskflow/internal/service"
	"time"
)

const SessionCookie = "taskflow_session"

type AuthHandler struct {
	Service      *service.AuthService
	SecureCookie bool
}

func RespondError(c *gin.Context, err error) {
	app := service.AsAppError(err)
	Error(c, app.Status, app.Code, app.Message)
}
func decode(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		Error(c, 400, "VALIDATION_ERROR", "请求参数无效或请求体过大")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		Error(c, 400, "VALIDATION_ERROR", "请求体应为一个 JSON 对象")
		return false
	}
	return true
}
func (h *AuthHandler) cookie(c *gin.Context, token string, clear bool) {
	age := int(repository.SessionTTL.Seconds())
	expires := time.Now().Add(repository.SessionTTL)
	if clear {
		age = -1
		expires = time.Unix(1, 0)
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: SessionCookie, Value: token, Path: "/", MaxAge: age, Expires: expires, HttpOnly: true, Secure: h.SecureCookie, SameSite: http.SameSiteLaxMode})
}
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !decode(c, &req) {
		return
	}
	user, err := h.Service.Register(c.Request.Context(), req)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 201, user)
}
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !decode(c, &req) {
		return
	}
	old, _ := c.Cookie(SessionCookie)
	user, token, err := h.Service.Login(c.Request.Context(), req, old)
	if err != nil {
		RespondError(c, err)
		return
	}
	h.cookie(c, token, false)
	Success(c, 200, user)
}
func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(SessionCookie)
	if err := h.Service.Logout(c.Request.Context(), token); err != nil {
		RespondError(c, err)
		return
	}
	h.cookie(c, "", true)
	c.Status(204)
}
func currentUser(c *gin.Context) *model.User { return c.MustGet("user").(*model.User) }
func (h *AuthHandler) Me(c *gin.Context)     { Success(c, 200, dto.UserFrom(currentUser(c))) }
func (h *AuthHandler) Profile(c *gin.Context) {
	var req dto.ProfileRequest
	if !decode(c, &req) {
		return
	}
	user, err := h.Service.UpdateProfile(c.Request.Context(), currentUser(c), req)
	if err != nil {
		RespondError(c, err)
		return
	}
	Success(c, 200, user)
}
func (h *AuthHandler) Password(c *gin.Context) {
	var req dto.PasswordRequest
	if !decode(c, &req) {
		return
	}
	if err := h.Service.ChangePassword(c.Request.Context(), currentUser(c), req); err != nil {
		RespondError(c, err)
		return
	}
	h.cookie(c, "", true)
	c.Status(204)
}
