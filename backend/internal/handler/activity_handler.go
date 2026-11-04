package handler

import (
	"github.com/gin-gonic/gin"

	"taskflow/internal/dto"
	"taskflow/internal/service"
)

func activityFilters(c *gin.Context) (dto.ActivityFilter, bool) {
	f := dto.ActivityFilter{Action: c.Query("action"), TargetType: c.Query("target_type")}
	for _, field := range []struct {
		name   string
		target **uint64
	}{{"actor_id", &f.ActorID}, {"project_id", &f.ProjectID}} {
		if raw := c.Query(field.name); raw != "" {
			id, ok := parseID(c, raw)
			if !ok {
				return f, false
			}
			*field.target = &id
		}
	}
	var err error
	f.From, err = service.ParseActivityTime(c.Query("from"))
	if err != nil {
		RespondError(c, err)
		return f, false
	}
	f.To, err = service.ParseActivityTime(c.Query("to"))
	if err != nil {
		RespondError(c, err)
		return f, false
	}
	if err = service.ValidateActivityFilter(f); err != nil {
		RespondError(c, err)
		return f, false
	}
	return f, true
}
