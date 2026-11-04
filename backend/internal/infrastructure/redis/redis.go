package redis

import (
	redisclient "github.com/redis/go-redis/v9"
	"time"
)

func New(addr string) *redisclient.Client {
	return redisclient.NewClient(&redisclient.Options{Addr: addr, DialTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second, MaxRetries: -1, ContextTimeoutEnabled: true})
}
