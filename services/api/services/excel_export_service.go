package services

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/gradiliste/api/dto"
)

type ExcelExportService struct{}

func NewExcelExportService() *ExcelExportService {
	return &ExcelExportService{}
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func colName(n int) string {
	name, _ := excelize.ColumnNumberToName(n)
	return name
}

func cellRef(col, row int) string {
	return fmt.Sprintf("%s%d", colName(col), row)
}

func parseDate(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return t.Format("02.01.2006")
}

func strOrDash(s *string) string {
	if s == nil || *s == "" {
		return "—"
	}
	return *s
}

func statusLabel(s string) string {
	switch s {
	case "draft":
		return "Nacrt"
	case "submitted":
		return "Predano"
	case "approved":
		return "Odobreno"
	case "rejected":
		return "Odbijeno"
	case "radnik_unos":
		return "—"
	default:
		return s
	}
}

func activityLabel(s string) string {
	switch s {
	case "montaza":
		return "Montaža"
	case "demontaza":
		return "Demontaža"
	case "other":
		return "Ostalo"
	default:
		return s
	}
}

func makeHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1E3A5F"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "bottom", Color: "4A90D9", Style: 2},
		},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: false},
	})
}

func makeDataStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: false},
	})
}

func makeNumberStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    2, // 0.00
	})
}

func makeMetaStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Italic: true, Color: "64748B"},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
}

func setHeaders(f *excelize.File, sheet string, headers []string, style int) {
	for i, h := range headers {
		_ = f.SetCellValue(sheet, cellRef(i+1, 1), h)
	}
	lastCol := colName(len(headers))
	_ = f.SetCellStyle(sheet, "A1", lastCol+"1", style)
	_ = f.SetRowHeight(sheet, 1, 22)
}

func freezeFirstRow(f *excelize.File, sheet string) {
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
		Selection:   []excelize.Selection{{SQRef: "A2", ActiveCell: "A2", Pane: "bottomLeft"}},
	})
}

func addSummarySheet(f *excelize.File, title string, rows int, summaryLines []string) {
	sheet := "Sažetak"
	_, _ = f.NewSheet(sheet)

	metaStyle, _ := makeMetaStyle(f)
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})

	_ = f.SetCellValue(sheet, "A1", title)
	_ = f.SetCellStyle(sheet, "A1", "A1", boldStyle)

	_ = f.SetCellValue(sheet, "A2", fmt.Sprintf("Generirano: %s", time.Now().Format("02.01.2006 15:04")))
	_ = f.SetCellStyle(sheet, "A2", "A2", metaStyle)

	_ = f.SetCellValue(sheet, "A3", fmt.Sprintf("Ukupno redova: %d", rows))
	_ = f.SetCellStyle(sheet, "A3", "A3", metaStyle)

	for i, line := range summaryLines {
		row := i + 4
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), line)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), metaStyle)
	}
	_ = f.SetColWidth(sheet, "A", "A", 50)
}

// ── Građevinski dnevnik ───────────────────────────────────────────────────────

func (s *ExcelExportService) BuildDnevnikXLSX(rows []dto.GradevinskiDnevnikRow, summary dto.GradevinskiDnevnikSummary) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Gradjevinski dnevnik"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Datum", "Gradilište", "Adresa gradilišta",
		"Poslovođa", "Radnik", "Sati",
		"Status izvještaja", "Napomena", "Izvor",
	}

	headerStyle, err := makeHeaderStyle(f)
	if err != nil {
		return nil, err
	}
	dataStyle, err := makeDataStyle(f)
	if err != nil {
		return nil, err
	}
	numStyle, err := makeNumberStyle(f)
	if err != nil {
		return nil, err
	}

	setHeaders(f, sheet, headers, headerStyle)
	freezeFirstRow(f, sheet)

	for i, row := range rows {
		r := i + 2
		_ = f.SetCellValue(sheet, cellRef(1, r), parseDate(row.ReportDate))
		_ = f.SetCellValue(sheet, cellRef(2, r), row.ProjectName)
		_ = f.SetCellValue(sheet, cellRef(3, r), strOrDash(row.ProjectAddress))
		_ = f.SetCellValue(sheet, cellRef(4, r), row.PoslovodaName)
		_ = f.SetCellValue(sheet, cellRef(5, r), row.WorkerName)
		_ = f.SetCellValue(sheet, cellRef(6, r), row.HoursWorked)
		_ = f.SetCellValue(sheet, cellRef(7, r), statusLabel(row.Status))
		_ = f.SetCellValue(sheet, cellRef(8, r), strOrDash(row.Notes))
		_ = f.SetCellValue(sheet, cellRef(9, r), row.Source)

		_ = f.SetCellStyle(sheet, cellRef(1, r), cellRef(5, r), dataStyle)
		_ = f.SetCellStyle(sheet, cellRef(6, r), cellRef(6, r), numStyle)
		_ = f.SetCellStyle(sheet, cellRef(7, r), cellRef(9, r), dataStyle)
	}

	// Column widths
	widths := []float64{12, 30, 25, 22, 22, 8, 18, 30, 16}
	for i, w := range widths {
		_ = f.SetColWidth(sheet, colName(i+1), colName(i+1), w)
	}

	addSummarySheet(f, "Građevinski dnevnik", len(rows), []string{
		fmt.Sprintf("Ukupno sati: %.2f", summary.TotalHours),
		fmt.Sprintf("Broj radnika: %d", summary.WorkersCount),
		fmt.Sprintf("Broj gradilišta: %d", summary.ProjectsCount),
	})

	// Main sheet first
	if idx, err2 := f.GetSheetIndex(sheet); err2 == nil {
		f.SetActiveSheet(idx)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// ── Građevinska knjiga ────────────────────────────────────────────────────────

func (s *ExcelExportService) BuildKnjigaXLSX(rows []dto.GradevinskaKnjigaRow, summary dto.GradevinskaKnjigaSummary) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Gradjevinska knjiga"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Datum", "Gradilište", "Adresa gradilišta",
		"Poslovođa", "Materijal", "Šifra materijala",
		"Količina", "Jedinica", "Aktivnost",
		"VTK", "Status izvještaja", "Napomena",
	}

	headerStyle, err := makeHeaderStyle(f)
	if err != nil {
		return nil, err
	}
	dataStyle, err := makeDataStyle(f)
	if err != nil {
		return nil, err
	}
	numStyle, err := makeNumberStyle(f)
	if err != nil {
		return nil, err
	}

	setHeaders(f, sheet, headers, headerStyle)
	freezeFirstRow(f, sheet)

	for i, row := range rows {
		r := i + 2
		vtk := "Ne"
		if row.IsVTK {
			vtk = "Da"
		}
		matName := row.MaterialName
		if matName == "" {
			matName = strOrDash(row.CustomMaterialName)
		}

		_ = f.SetCellValue(sheet, cellRef(1, r), parseDate(row.ReportDate))
		_ = f.SetCellValue(sheet, cellRef(2, r), row.ProjectName)
		_ = f.SetCellValue(sheet, cellRef(3, r), strOrDash(row.ProjectAddress))
		_ = f.SetCellValue(sheet, cellRef(4, r), row.PoslovodaName)
		_ = f.SetCellValue(sheet, cellRef(5, r), matName)
		_ = f.SetCellValue(sheet, cellRef(6, r), strOrDash(row.MaterialCode))
		_ = f.SetCellValue(sheet, cellRef(7, r), row.Quantity)
		_ = f.SetCellValue(sheet, cellRef(8, r), row.Unit)
		_ = f.SetCellValue(sheet, cellRef(9, r), activityLabel(row.ActivityType))
		_ = f.SetCellValue(sheet, cellRef(10, r), vtk)
		_ = f.SetCellValue(sheet, cellRef(11, r), statusLabel(row.Status))
		_ = f.SetCellValue(sheet, cellRef(12, r), strOrDash(row.Notes))

		_ = f.SetCellStyle(sheet, cellRef(1, r), cellRef(6, r), dataStyle)
		_ = f.SetCellStyle(sheet, cellRef(7, r), cellRef(7, r), numStyle)
		_ = f.SetCellStyle(sheet, cellRef(8, r), cellRef(12, r), dataStyle)
	}

	// Column widths
	widths := []float64{12, 28, 24, 22, 30, 16, 10, 10, 14, 6, 18, 28}
	for i, w := range widths {
		_ = f.SetColWidth(sheet, colName(i+1), colName(i+1), w)
	}

	vtk := 0
	for _, r := range rows {
		if r.IsVTK {
			vtk++
		}
	}
	addSummarySheet(f, "Građevinska knjiga", len(rows), []string{
		fmt.Sprintf("Ukupna količina: %.2f", summary.TotalQuantity),
		fmt.Sprintf("Broj aktivnosti: %d", summary.ActivitiesCount),
		fmt.Sprintf("VTK stavki: %d", vtk),
		fmt.Sprintf("Broj gradilišta: %d", summary.ProjectsCount),
	})

	if idx, err2 := f.GetSheetIndex(sheet); err2 == nil {
		f.SetActiveSheet(idx)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// ExcelFilename returns a safe download filename.
func ExcelFilename(base string) string {
	date := time.Now().Format("2006-01-02")
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
	return fmt.Sprintf("%s-%s.xlsx", safe, date)
}
