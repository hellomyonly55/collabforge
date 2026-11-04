package config

import (
	"strings"
	"testing"
)

func base(t *testing.T) {
	t.Helper()
	t.Setenv("MYSQL_DATABASE", "taskflow_test")
	t.Setenv("MYSQL_USER", "test")
	t.Setenv("MYSQL_PASSWORD", "not-a-real-secret")
	t.Setenv("MYSQL_PORT", "3306")
	t.Setenv("APP_ORIGIN", "http://localhost:5173")
}
func TestRequiredConfiguration(t *testing.T) {
	base(t)
	t.Setenv("MYSQL_PASSWORD", "")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "MYSQL_PASSWORD") {
		t.Fatal("expected missing password error")
	}
}
func TestInvalidPortAndOrigin(t *testing.T) {
	for _, tc := range []struct{ key, value string }{{"MYSQL_PORT", "0"}, {"MYSQL_PORT", "oops"}, {"APP_ORIGIN", "https://example.com/path"}, {"APP_ORIGIN", "file:///tmp/a"}, {"APP_ORIGIN", "https://user:password@example.com"}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			base(t)
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}
func TestValidConfiguration(t *testing.T) {
	base(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Name != "taskflow_test" {
		t.Fatal("database name was not read")
	}
}

func TestHTTPSCookieAndTrustedProxies(t *testing.T) {
	base(t)
	t.Setenv("APP_ORIGIN", "https://taskflow.example")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1,172.16.0.0/12")
	cfg, err := Load()
	if err != nil || !cfg.SecureCookie || len(cfg.TrustedProxies) != 2 {
		t.Fatal("HTTPS/proxy configuration invalid")
	}
	t.Setenv("TRUSTED_PROXIES", "not-a-network")
	if _, err := Load(); err == nil {
		t.Fatal("invalid trusted proxy accepted")
	}
}
