package service

import (
	"testing"
	"time"

	"github.com/whiterage/link-checker-service/pkg/models"
)

func TestResetStalledTask(t *testing.T) {
	task := &models.Task{
		Status: models.StatusProcessing,
		Results: []models.LinkStatus{
			{URL: "https://ok.ru", Status: models.StatusProcessing, CheckTime: time.Now()},
			{URL: "https://done.ru", Status: models.StatusAvailable},
		},
	}

	resetStalledTask(task)

	if task.Status != models.StatusPending {
		t.Fatalf("expected pending status, got %s", task.Status)
	}
	if task.Results[0].Status != models.StatusPending {
		t.Fatalf("processing result should be reset")
	}
	if !task.Results[0].CheckTime.IsZero() {
		t.Fatalf("check time should be zeroed")
	}
	if task.Results[1].Status != models.StatusAvailable {
		t.Fatalf("completed result should be kept")
	}
}
