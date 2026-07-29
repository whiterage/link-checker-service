package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	api "github.com/whiterage/link-checker-service/internal/http"
	"github.com/whiterage/link-checker-service/internal/repository"
	"github.com/whiterage/link-checker-service/internal/service"
	"github.com/whiterage/link-checker-service/internal/worker"
)

func main() {
	storagePath := os.Getenv("TASK_STORAGE_PATH")
	if storagePath == "" {
		storagePath = filepath.Join("storage", "tasks.json")
	}

	repo, err := repository.NewPersistentRepo(storagePath)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}

	checker := worker.NewHTTPChecker(5 * time.Second)
	svc := service.NewService(repo, checker, 20)
	pool := service.NewWorkerPool(svc, 4)
	handlers := api.NewHandlers(svc)
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	pool.Start(workerCtx)

	handlers.Register(mux)

	server := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("shutdown: stopping http server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	svc.CloseQueue()

	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-shutdownCtx.Done():
		log.Println("shutdown: timeout exceeded, forcing workers to stop")
		pool.Stop()
	}

	log.Println("shutdown: complete")
}
