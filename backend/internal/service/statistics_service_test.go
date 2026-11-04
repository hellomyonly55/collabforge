package service

import "testing"

func TestCompletionRate(t *testing.T) {
	for _, c := range []struct {
		done, total int64
		want        float64
	}{{0, 0, 0}, {1, 3, 33.3}, {2, 3, 66.7}, {3, 3, 100}, {0, 3, 0}} {
		if got := completionRate(c.done, c.total); got != c.want {
			t.Fatalf("%d/%d = %v", c.done, c.total, got)
		}
	}
}
