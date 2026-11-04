package repository

import (
	"context"
	"database/sql"
	redisclient "github.com/redis/go-redis/v9"
	"taskflow/migrations"
)

type HealthRepository struct {
	DB    *sql.DB
	Redis *redisclient.Client
}

func (r *HealthRepository) MySQL(ctx context.Context) error     { return r.DB.PingContext(ctx) }
func (r *HealthRepository) RedisPing(ctx context.Context) error { return r.Redis.Ping(ctx).Err() }
func (r *HealthRepository) Migrations(ctx context.Context) error {
	var count int
	if err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migrations.LatestVersion).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}
