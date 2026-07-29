package repository

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/whiterage/link-checker-service/pkg/models"
)

func TestPersistentRepo_ReloadsTasks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	repo, err := NewPersistentRepo(path)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	task := &models.Task{
		ID:        7,
		CreatedAt: time.Now(),
		Status:    models.StatusDone,
		Results: []models.LinkStatus{
			{URL: "https://example.com", Status: models.StatusAvailable},
		},
	}
	repo.Save(task)

	reloaded, err := NewPersistentRepo(path)
	if err != nil {
		t.Fatalf("reopen repo: %v", err)
	}

	got, ok := reloaded.Get(7)
	if !ok {
		t.Fatalf("task not found after reload")
	}
	if got.Status != models.StatusDone {
		t.Fatalf("unexpected status after reload: %s", got.Status)
	}
	if reloaded.MaxID() != 7 {
		t.Fatalf("unexpected max id: %d", reloaded.MaxID())
	}
}

func TestMemoryRepo_PendingTasks(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepo()
	repo.Save(&models.Task{ID: 1, Status: models.StatusPending})
	repo.Save(&models.Task{ID: 2, Status: models.StatusProcessing})
	repo.Save(&models.Task{ID: 3, Status: models.StatusDone})

	pending := repo.PendingTasks()
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending tasks, got %d", len(pending))
	}
}
