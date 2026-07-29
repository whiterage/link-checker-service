package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/whiterage/link-checker-service/internal/service"
	"github.com/whiterage/link-checker-service/pkg/models"
)

type Handlers struct {
	svc *service.Service
}

func NewHandlers(svc *service.Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/links", h.createLinks)
	mux.HandleFunc("/links/", h.getLink)
	mux.HandleFunc("/links_list", h.generateReport)
}

func (h *Handlers) createLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.LinkRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	id, err := h.svc.CreateTask(r.Context(), req.Links)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrEmptyLinks):
			status = http.StatusBadRequest
		case errors.Is(err, context.Canceled):
			status = http.StatusRequestTimeout
		}
		http.Error(w, err.Error(), status)
		return
	}

	task, err := h.svc.GetTask(id)
	if err != nil {
		http.Error(w, "cannot load task", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"links":     buildLinksMap(task.Results),
		"links_num": task.ID,
		"status":    task.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) getLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/links/")
	if idStr == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid links_num", http.StatusBadRequest)
		return
	}

	task, err := h.svc.GetTask(id)
	if errors.Is(err, service.ErrTaskNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"links":     buildLinksMap(task.Results),
		"links_num": task.ID,
		"status":    task.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) generateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.ReportRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if len(req.LinksList) == 0 {
		http.Error(w, "links_list is required", http.StatusBadRequest)
		return
	}

	data, err := h.svc.GenerateReport(r.Context(), req.LinksList)
	if err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			http.Error(w, "tasks not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"links_report.pdf\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func buildLinksMap(results []models.LinkStatus) map[string]string {
	resp := make(map[string]string, len(results))
	for _, res := range results {
		resp[res.URL] = res.Status
	}
	return resp
}
