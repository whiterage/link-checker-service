package repository

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/whiterage/link-checker-service/pkg/models"
)

type MemoryRepo struct {
	tasks       map[int]*models.Task
	mu          sync.RWMutex
	storagePath string
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		tasks: make(map[int]*models.Task),
	}
}

func NewPersistentRepo(path string) (*MemoryRepo, error) {
	if path == "" {
		return nil, errors.New("storage path is required")
	}

	repo := &MemoryRepo{
		tasks:       make(map[int]*models.Task),
		storagePath: path,
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *MemoryRepo) Save(task *models.Task) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.ID] = task
	r.persistLocked()
}

func (r *MemoryRepo) Get(id int) (*models.Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.tasks[id]
	return task, ok
}

func (r *MemoryRepo) List(ids []int) []*models.Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tasks := make([]*models.Task, 0, len(ids))
	for _, id := range ids {
		if task, ok := r.tasks[id]; ok {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (r *MemoryRepo) PendingTasks() []*models.Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []*models.Task
	for _, task := range r.tasks {
		if task.Status == models.StatusPending || task.Status == models.StatusProcessing {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (r *MemoryRepo) MaxID() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	max := 0
	for id := range r.tasks {
		if id > max {
			max = id
		}
	}
	return max
}

func (r *MemoryRepo) load() error {
	if r.storagePath == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(r.storagePath), 0o755); err != nil {
		return err
	}

	file, err := os.Open(r.storagePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	var state storageState
	if err := json.NewDecoder(file).Decode(&state); err != nil {
		return err
	}

	for _, task := range state.Tasks {
		r.tasks[task.ID] = task
	}

	return nil
}

func (r *MemoryRepo) persistLocked() {
	if r.storagePath == "" {
		return
	}

	state := storageState{
		Tasks: make([]*models.Task, 0, len(r.tasks)),
	}
	for _, task := range r.tasks {
		state.Tasks = append(state.Tasks, task)
	}

	tmp := r.storagePath + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		log.Printf("repository: cannot create temp storage file: %v", err)
		return
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&state); err != nil {
		file.Close()
		log.Printf("repository: cannot encode storage file: %v", err)
		_ = os.Remove(tmp)
		return
	}
	if err := file.Close(); err != nil {
		log.Printf("repository: cannot close temp storage file: %v", err)
		_ = os.Remove(tmp)
		return
	}

	if err := os.Rename(tmp, r.storagePath); err != nil {
		log.Printf("repository: cannot rotate storage file: %v", err)
	}
}

type storageState struct {
	Tasks []*models.Task `json:"tasks"`
}
