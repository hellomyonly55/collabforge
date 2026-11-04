package service

import (
	"strings"
	"testing"
)

func TestTeamTextAndRoles(t *testing.T) {
	value, err := boundedText(" 团队 ", "团队名称", 1, 80)
	if err != nil || value != "团队" {
		t.Fatal("team text normalization failed")
	}
	if _, err = boundedText(strings.Repeat("汉", 81), "团队名称", 1, 80); err == nil {
		t.Fatal("oversize name accepted")
	}
	if _, err = boundedText(" ", "团队名称", 1, 80); err == nil {
		t.Fatal("empty name accepted")
	}
	if manager("member") || !manager("admin") || !manager("owner") || ownerOnly("admin") == nil {
		t.Fatal("role policy failed")
	}
}
