package service

import (
	"strings"
	"testing"
)

func TestEmailNormalization(t *testing.T) {
	value, err := normalizeEmail("  User@Example.COM  ")
	if err != nil || value != "user@example.com" {
		t.Fatal("email normalization failed")
	}
	for _, raw := range []string{"bad", "Name <user@example.com>", "用户@example.com", strings.Repeat("a", 255) + "@example.com"} {
		if _, err := normalizeEmail(raw); err == nil {
			t.Fatalf("invalid email accepted: %q", raw)
		}
	}
}
func TestNicknameValidation(t *testing.T) {
	value, err := nickname(" 小林 ")
	if err != nil || value != "小林" {
		t.Fatal("nickname normalization failed")
	}
	for _, raw := range []string{" ", strings.Repeat("汉", 41)} {
		if _, err := nickname(raw); err == nil {
			t.Fatal("invalid nickname accepted")
		}
	}
}
