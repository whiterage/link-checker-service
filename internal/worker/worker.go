package worker

import (
	"context"
	"net/http"
	"time"

	"github.com/whiterage/link-checker-service/pkg/clock"
	"github.com/whiterage/link-checker-service/pkg/models"
)

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{client: &http.Client{Timeout: timeout}}
}

func (c *HTTPChecker) Check(ctx context.Context, url string) models.LinkStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return models.LinkStatus{URL: url, Status: models.StatusNotAvailable}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return models.LinkStatus{URL: url, Status: models.StatusNotAvailable}
	}
	resp.Body.Close()

	status := models.StatusAvailable
	if resp.StatusCode >= http.StatusBadRequest {
		status = models.StatusNotAvailable
	}

	return models.LinkStatus{
		URL:       url,
		Status:    status,
		CheckTime: clock.Now(),
	}
}
