package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"

	"github.com/whiterage/link-checker-service/pkg/clock"
	"github.com/whiterage/link-checker-service/pkg/models"
)

type WorkerPool struct {
	service *Service
	workers int
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

func NewWorkerPool(service *Service, workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}
	return &WorkerPool{
		service: service,
		workers: workers,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	workerCtx, cancel := context.WithCancel(ctx)
	wp.cancel = cancel
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.workerLoop(workerCtx)
	}
}

func (wp *WorkerPool) workerLoop(ctx context.Context) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-wp.service.queue:
			if !ok {
				return
			}
			wp.processTask(ctx, task)
		}
	}
}

func (wp *WorkerPool) processTask(ctx context.Context, task *models.Task) {
	task.Status = models.StatusProcessing
	wp.service.repo.Save(task)

	for i := range task.Results {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resolvedURL, err := normalizeURL(task.Results[i].URL)
		if err != nil {
			task.Results[i].Status = models.StatusNotAvailable
			task.Results[i].CheckTime = clock.Now()
			continue
		}

		result := wp.service.checker.Check(ctx, resolvedURL)
		task.Results[i].Status = result.Status
		task.Results[i].CheckTime = result.CheckTime
	}

	task.Status = models.StatusDone
	wp.service.repo.Save(task)
}

func (wp *WorkerPool) Stop() {
	if wp.cancel != nil {
		wp.cancel()
	}
	wp.wg.Wait()
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

func normalizeURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("empty url")
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Host == "" {
		return "", errors.New("invalid url")
	}
	return parsed.String(), nil
}
