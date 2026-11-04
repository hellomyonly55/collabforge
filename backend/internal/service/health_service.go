package service

import (
	"context"
	"taskflow/internal/dto"
	"time"
)

type HealthChecker interface {
	MySQL(context.Context) error
	RedisPing(context.Context) error
	Migrations(context.Context) error
}
type HealthService struct{ Checker HealthChecker }

func (s *HealthService) Ready(ctx context.Context) (dto.Health, bool) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	result := dto.Health{Status: "ready", Checks: map[string]string{}}
	ok := true
	for _, check := range []struct {
		name string
		run  func(context.Context) error
	}{{"mysql", s.Checker.MySQL}, {"redis", s.Checker.RedisPing}, {"migrations", s.Checker.Migrations}} {
		if err := check.run(ctx); err != nil {
			result.Checks[check.name] = "unavailable"
			ok = false
		} else {
			result.Checks[check.name] = "ok"
		}
	}
	if !ok {
		result.Status = "not_ready"
	}
	return result, ok
}
