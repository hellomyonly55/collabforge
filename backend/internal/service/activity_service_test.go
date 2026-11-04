package service

import (
	"testing"
	"time"

	"taskflow/internal/dto"
)

func TestActivityTimeAndRange(t *testing.T) {
	from, err := ParseActivityTime("2026-10-31T00:00:00+08:00")
	if err != nil || from.Format(time.RFC3339) != "2026-10-30T16:00:00Z" {
		t.Fatal("Shanghai date boundary must convert to UTC")
	}
	for _, bad := range []string{"2026-10-31", "2026-02-30T00:00:00Z", "not-a-time", "0999-01-01T00:00:00Z"} {
		if _, err := ParseActivityTime(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	earlier := from.Add(-time.Hour)
	if ValidateActivityFilter(dto.ActivityFilter{From: from, To: &earlier}) == nil {
		t.Fatal("accepted inverted interval")
	}
}
