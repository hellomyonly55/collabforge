package service

import (
	"testing"
	"time"

	"taskflow/internal/model"
	"taskflow/internal/repository"
)

func TestTaskDatesAndOverdue(t *testing.T) {
	bad := "2026-02-30"
	if _, err := taskDate(&bad); err == nil {
		t.Fatal("invalid calendar date accepted")
	}
	valid := "2028-02-29"
	date, err := taskDate(&valid)
	if err != nil || date.Format(time.DateOnly) != valid {
		t.Fatal("valid leap date rejected")
	}
	value := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	row := repository.TaskRecord{Task: model.Task{Status: "todo", DueAt: &value}}
	if !taskFrom(row, 0, "2026-10-31").IsOverdue || taskFrom(row, 0, "2026-10-30").IsOverdue {
		t.Fatal("date boundary incorrect")
	}
	row.Status = "done"
	if taskFrom(row, 0, "2026-10-31").IsOverdue {
		t.Fatal("completed task overdue")
	}
}
