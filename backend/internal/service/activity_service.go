package service

import (
	"time"

	"taskflow/internal/dto"
)

// Exact action names avoid SQL wildcards and keep filters aligned with real audit events.
var activityActions = map[string]bool{
	"team.created": true, "team.updated": true, "team.archived": true, "team.ownership_transferred": true,
	"member.joined": true, "member.left": true, "member.removed": true, "member.role_changed": true,
	"invite.created": true, "invite.revoked": true,
	"project.created": true, "project.updated": true, "project.archived": true,
	"task.created": true, "task.updated": true, "task.moved": true, "task.deleted": true, "task.unassigned": true,
	"comment.created": true, "comment.updated": true, "comment.deleted": true,
}

func ValidateActivityFilter(f dto.ActivityFilter) error {
	if f.Action != "" && !activityActions[f.Action] {
		return Fail(400, "VALIDATION_ERROR", "操作类型无效")
	}
	switch f.TargetType {
	case "", "team", "user", "invite", "project", "task", "comment":
	default:
		return Fail(400, "VALIDATION_ERROR", "目标类型无效")
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return Fail(400, "VALIDATION_ERROR", "开始时间不能晚于结束时间")
	}
	return nil
}
func ParseActivityTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || t.UTC().Year() < 1000 || t.UTC().Year() > 9999 {
		return nil, Fail(400, "VALIDATION_ERROR", "日志时间必须为包含时区的 RFC3339 时间")
	}
	t = t.UTC()
	return &t, nil
}
