package router

import (
	"github.com/gin-gonic/gin"
	"taskflow/internal/handler"
	"taskflow/internal/middleware"
	"time"
)

func RegisterTeams(r *gin.Engine, auth *handler.AuthHandler, team *handler.TeamHandler) {
	g := r.Group("/api/v1", func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() }, middleware.RequireAuth(auth.Service))
	g.GET("/teams", team.List)
	g.POST("/teams", middleware.RateLimit(auth.Service.Sessions, "team:create", 20, time.Hour, true), team.Create)
	g.POST("/teams/join", middleware.RateLimit(auth.Service.Sessions, "team:join", 30, time.Minute, true), team.Join)
	g.GET("/teams/:teamId", team.Get)
	g.PATCH("/teams/:teamId", team.Update)
	g.POST("/teams/:teamId/archive", team.Archive)
	g.POST("/teams/:teamId/leave", team.Leave)
	g.POST("/teams/:teamId/transfer-ownership", team.Transfer)
	g.GET("/teams/:teamId/members", team.Members)
	g.PATCH("/teams/:teamId/members/:userId", team.Role)
	g.DELETE("/teams/:teamId/members/:userId", team.Remove)
	g.GET("/teams/:teamId/invites", team.Invites)
	g.POST("/teams/:teamId/invites", middleware.RateLimit(auth.Service.Sessions, "team:invite", 30, time.Hour, true), team.CreateInvite)
	g.DELETE("/teams/:teamId/invites/:inviteId", team.RevokeInvite)
	g.GET("/teams/:teamId/activities", team.Activities)
}
