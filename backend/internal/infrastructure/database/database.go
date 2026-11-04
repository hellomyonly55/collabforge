package database

import (
	"context"
	"database/sql"
	"net"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"taskflow/internal/config"
)

type Connection struct {
	ORM *gorm.DB
	SQL *sql.DB
}

// Open bounds connection time and avoids printing driver errors containing credentials.
func Open(c config.Database) (*Connection, error) {
	dsn := driver.NewConfig()
	dsn.User = c.User
	dsn.Passwd = c.Password
	dsn.Net = "tcp"
	dsn.Addr = net.JoinHostPort(c.Host, c.Port)
	dsn.DBName = c.Name
	dsn.ParseTime = true
	dsn.Loc = time.UTC
	dsn.Timeout = 3 * time.Second
	dsn.ReadTimeout = 3 * time.Second
	dsn.WriteTimeout = 3 * time.Second
	dsn.Params = map[string]string{"charset": "utf8mb4", "time_zone": "'+00:00'"}
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn.FormatDSN(), SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableAutomaticPing: true, TranslateError: true})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)
	return &Connection{ORM: db, SQL: pool}, nil
}
func (c *Connection) Ping(ctx context.Context) error { return c.SQL.PingContext(ctx) }
