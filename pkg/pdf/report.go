package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/whiterage/link-checker-service/pkg/clock"
	"github.com/whiterage/link-checker-service/pkg/models"
)

func BuildReport(tasks []*models.Task) ([]byte, error) {
	doc := gofpdf.New("P", "mm", "A4", "")
	doc.SetTitle("Links status report", false)
	doc.SetAuthor("Webserver Go", false)
	doc.AddPage()

	doc.SetFont("Arial", "B", 16)
	doc.Cell(0, 10, "Links Status Report")
	doc.Ln(8)

	doc.SetFont("Arial", "", 11)
	doc.Cell(0, 6, fmt.Sprintf("Generated at: %s", clock.Now().Format(time.RFC3339)))
	doc.Ln(10)

	for _, task := range tasks {
		writeTaskBlock(doc, task)
		doc.Ln(4)
	}

	buffer := bytes.NewBuffer(nil)
	if err := doc.Output(buffer); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func writeTaskBlock(doc *gofpdf.Fpdf, task *models.Task) {
	doc.SetFont("Arial", "B", 12)
	doc.Cell(0, 7, fmt.Sprintf("Task #%d — status: %s", task.ID, task.Status))
	doc.Ln(6)

	doc.SetFont("Arial", "", 10)
	doc.Cell(0, 5, fmt.Sprintf("Created at: %s", task.CreatedAt.Format(time.RFC3339)))
	doc.Ln(8)

	doc.SetFont("Arial", "B", 11)
	doc.CellFormat(90, 7, "URL", "1", 0, "", false, 0, "")
	doc.CellFormat(40, 7, "Status", "1", 0, "", false, 0, "")
	doc.CellFormat(0, 7, "Checked At", "1", 1, "", false, 0, "")

	doc.SetFont("Arial", "", 10)
	for _, res := range task.Results {
		checked := "-"
		if !res.CheckTime.IsZero() {
			checked = res.CheckTime.Format(time.RFC3339)
		}

		doc.CellFormat(90, 6, res.URL, "1", 0, "", false, 0, "")
		doc.CellFormat(40, 6, res.Status, "1", 0, "", false, 0, "")
		doc.CellFormat(0, 6, checked, "1", 1, "", false, 0, "")
	}
}
