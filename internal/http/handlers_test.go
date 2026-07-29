package api

import (
	"reflect"
	"testing"

	"github.com/whiterage/link-checker-service/pkg/models"
)

func TestBuildLinksMap(t *testing.T) {
	input := []models.LinkStatus{
		{URL: "google.com", Status: models.StatusAvailable},
		{URL: "example.com", Status: models.StatusNotAvailable},
	}

	got := buildLinksMap(input)
	want := map[string]string{
		"google.com":  models.StatusAvailable,
		"example.com": models.StatusNotAvailable,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected map: %+v", got)
	}
}
