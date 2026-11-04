// Package config loads explicit environment configuration without logging secrets.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Database struct{ Host, Port, Name, User, Password string }
type Config struct {
	HTTPAddr, AppOrigin, RedisAddr string
	Database                       Database
	TrustedProxies                 []string
	SecureCookie                   bool
}

func Load() (Config, error) {
	c := Config{HTTPAddr: env("HTTP_ADDR", "127.0.0.1:8081"), AppOrigin: env("APP_ORIGIN", "http://localhost:5173"), RedisAddr: env("REDIS_ADDR", "127.0.0.1:6379")}
	c.Database = Database{Host: env("MYSQL_HOST", "127.0.0.1"), Port: env("MYSQL_PORT", "3306"), Name: os.Getenv("MYSQL_DATABASE"), User: os.Getenv("MYSQL_USER"), Password: os.Getenv("MYSQL_PASSWORD")}
	for _, key := range []string{"MYSQL_DATABASE", "MYSQL_USER", "MYSQL_PASSWORD"} {
		if os.Getenv(key) == "" {
			return Config{}, fmt.Errorf("required configuration %s is missing", key)
		}
	}
	p, err := strconv.Atoi(c.Database.Port)
	if err != nil || p < 1 || p > 65535 {
		return Config{}, fmt.Errorf("MYSQL_PORT must be between 1 and 65535")
	}
	u, err := url.Parse(c.AppOrigin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return Config{}, fmt.Errorf("APP_ORIGIN must be an HTTP(S) origin without path")
	}
	c.SecureCookie = u.Scheme == "https"
	if raw := os.Getenv("TRUSTED_PROXIES"); raw != "" {
		for _, value := range strings.Split(raw, ",") {
			value = strings.TrimSpace(value)
			if net.ParseIP(value) == nil {
				if _, _, err := net.ParseCIDR(value); err != nil {
					return Config{}, fmt.Errorf("TRUSTED_PROXIES contains an invalid IP or CIDR")
				}
			}
			c.TrustedProxies = append(c.TrustedProxies, value)
		}
	}
	return c, nil
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
