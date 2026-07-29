package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/whiterage/link-checker-service/internal/repository"
	"github.com/whiterage/link-checker-service/pkg/clock"
	"github.com/whiterage/link-checker-service/pkg/models"
	"github.com/whiterage/link-checker-service/pkg/pdf"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrEmptyLinks   = errors.New("empty links payload")
)

type Checker interface {
	Check(ctx context.Context, url string) models.LinkStatus
}

type Service struct {
	repo    *repository.MemoryRepo
	queue   chan *models.Task
	checker Checker
	mu      sync.Mutex
	nextID  int
	closed  atomic.Bool
	closeW  sync.Once
}

func NewService(repo *repository.MemoryRepo, checker Checker, queueSize int) *Service {
	if queueSize <= 0 {
		queueSize = 10
	}

	pending := repo.PendingTasks()
	if len(pending) > queueSize {
		queueSize = len(pending) + 5
	}

	s := &Service{
		repo:    repo,
		queue:   make(chan *models.Task, queueSize),
		checker: checker,
		nextID:  repo.MaxID() + 1,
	}

	if s.nextID <= 1 {
		s.nextID = 1
	}

	for _, task := range pending {
		resetStalledTask(task)
		repo.Save(task)
		s.queue <- task
	}

	return s
}

func (s *Service) CreateTask(ctx context.Context, links []string) (int, error) {
	if len(links) == 0 {
		return 0, ErrEmptyLinks
	}
	if s.closed.Load() {
		return 0, errors.New("service is shutting down")
	}

	task := &models.Task{
		ID:        s.nextTaskID(),
		CreatedAt: clock.Now(),
		Status:    models.StatusPending,
		Results:   make([]models.LinkStatus, len(links)),
	}

	for i, url := range links {
		task.Results[i] = models.LinkStatus{
			URL:    url,
			Status: models.StatusPending,
		}
	}

	s.repo.Save(task)

	select {
	case s.queue <- task:
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	return task.ID, nil
}

func (s *Service) GetTask(id int) (*models.Task, error) {
	task, ok := s.repo.Get(id)
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (s *Service) GenerateReport(ctx context.Context, ids []int) ([]byte, error) {
	if len(ids) == 0 {
		return nil, errors.New("empty links_list payload")
	}

	tasks := s.repo.List(ids)
	if len(tasks) == 0 || len(tasks) != len(ids) {
		return nil, ErrTaskNotFound
	}

	data, err := pdf.BuildReport(tasks)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *Service) nextTaskID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	return id
}

func (s *Service) CloseQueue() {
	s.closeW.Do(func() {
		s.closed.Store(true)
		close(s.queue)
	})
}

func resetStalledTask(task *models.Task) {
	if task.Status == models.StatusDone {
		return
	}

	task.Status = models.StatusPending
	for i := range task.Results {
		if task.Results[i].Status == models.StatusProcessing {
			task.Results[i].Status = models.StatusPending
			task.Results[i].CheckTime = time.Time{}
		}
	}
}
