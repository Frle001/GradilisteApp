package services

import (
	"os"
	"strings"
	"testing"

	"github.com/gradiliste/api/dto"
)

// ── normalizeHeader ───────────────────────────────────────────────────────────

func TestNormalizeHeader(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Naziv materijala", "naziv materijala"},
		{"Planirana količina", "planirana kolicina"},
		{"Jedinica mjere", "jedinica mjere"},
		{"Šifra materijala", "sifra materijala"},
		{"Jed. cijena (€/jm)", "jed cijena eur jm"},
		{"J.M.", "j m"},
		{"KOL.", "kol"},
		{"m²", "m2"},
		{"  NAZIV  ", "naziv"},
		{"UKUPNA VRIJEDNOST", "ukupna vrijednost"},
	}
	for _, c := range cases {
		got := normalizeHeader(c.input)
		if got != c.want {
			t.Errorf("normalizeHeader(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ── parseNumber ───────────────────────────────────────────────────────────────

func TestParseNumber(t *testing.T) {
	cases := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"5", 5, false},
		{"5.25", 5.25, false},
		{"5,25", 5.25, false},
		{"1.250,50", 1250.50, false},
		{"1,250.50", 1250.50, false},
		{"1 250,50", 1250.50, false},
		{"€ 4,80", 4.80, false},
		{"4,80 EUR", 4.80, false},
		{"0", 0, false},
		{"100", 100, false},
		{"", 0, true},
		{"abc", 0, true},
		{"--", 0, true},
	}
	for _, c := range cases {
		got, err := parseNumber(c.input)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseNumber(%q) expected error, got %v", c.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("parseNumber(%q) unexpected error: %v", c.input, err)
			} else if got != c.want {
				t.Errorf("parseNumber(%q) = %v, want %v", c.input, got, c.want)
			}
		}
	}
}

// ── isCategoryLike ────────────────────────────────────────────────────────────

func TestIsCategoryLike(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"I. RADOVI NA OBJEKTU", true},
		{"II. INSTALACIJE", true},
		{"BETONSKI RADOVI", true},
		{"Betonska opeka", false},
		{"dobava i ugradnja", false},
		{"I.", false}, // prefix without more text
		{"polaganje plocica", false},
	}
	for _, c := range cases {
		got := isCategoryLike(c.input)
		if got != c.want {
			t.Errorf("isCategoryLike(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

// ── classifyRow ───────────────────────────────────────────────────────────────

func TestClassifyRow(t *testing.T) {
	cases := []struct {
		name        string
		qtyStr      string
		unit        string
		prevWasItem bool
		want        string
	}{
		// blank row
		{"", "", "", false, RowTypeBlank},
		{"", "", "", true, RowTypeBlank},
		// subtotals
		{"Ukupno", "", "", false, RowTypeSubtotal},
		{"sveukupno radovi", "500", "eur", false, RowTypeSubtotal},
		// category headings
		{"I. RADOVI NA OBJEKTU", "", "", false, RowTypeCategory},
		{"BETONSKI RADOVI", "", "", false, RowTypeCategory},
		// items with qty
		{"Betonska opeka", "5", "kom", false, RowTypeItem},
		{"Betonska opeka", "5,25", "m2", true, RowTypeItem},
		// items with unit only (blank qty)
		{"Betonska opeka", "", "m2", false, RowTypeItem},
		{"Betonska opeka", "", "kom", true, RowTypeItem},
		// name-only rows: now classified as item (not silently skipped)
		{"Betonska opeka", "", "", false, RowTypeItem},
		{"Betonska opeka", "", "", true, RowTypeItem},
		// continuation markers
		{"- sa posebnim zahtjevima", "", "", true, RowTypeContinuation},
		{"• dobava materijala", "", "", false, RowTypeContinuation},
		{"* napomena", "", "", true, RowTypeContinuation},
	}
	for _, c := range cases {
		got := classifyRow(c.name, c.qtyStr, c.unit, c.prevWasItem)
		if got != c.want {
			t.Errorf("classifyRow(%q, %q, %q, %v) = %q, want %q",
				c.name, c.qtyStr, c.unit, c.prevWasItem, got, c.want)
		}
	}
}

// ── parseRowsWithMapping ──────────────────────────────────────────────────────

func mapping(pairs ...string) map[string]string {
	m := make(map[string]string, len(pairs)/2)
	for i := 0; i < len(pairs)-1; i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}

func rows(items ...rawMappingRow) []rawMappingRow { return items }

func row(num int, cells map[string]string) rawMappingRow {
	return rawMappingRow{RowNumber: num, Cells: cells}
}

func cells(pairs ...string) map[string]string {
	m := make(map[string]string, len(pairs)/2)
	for i := 0; i < len(pairs)-1; i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}

func TestParseRowsWithMapping_StandardCase(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(
		row(2, cells("Naziv", "Betonska opeka", "Količina", "5", "JM", "kom")),
		row(3, cells("Naziv", "Armatura", "Količina", "100", "JM", "kg")),
	)
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	if result[0].MaterialName != "Betonska opeka" || result[0].Quantity != 5 || result[0].Unit != "kom" {
		t.Errorf("row 0 unexpected: %+v", result[0])
	}
	if result[1].MaterialName != "Armatura" || result[1].Quantity != 100 || result[1].Unit != "kg" {
		t.Errorf("row 1 unexpected: %+v", result[1])
	}
	for _, r := range result {
		if r.Status != "valid" {
			t.Errorf("row %d expected valid, got %q (errors: %v)", r.RowNumber, r.Status, r.Errors)
		}
	}
}

func TestParseRowsWithMapping_BlankQtyBecomesWarning(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(
		row(2, cells("Naziv", "Betonska opeka", "Količina", "", "JM", "kom")),
		row(3, cells("Naziv", "Armatura", "Količina", "100", "JM", "kg")),
	)
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}

	// Row 0: blank qty should be valid with qty=0 and a warning
	r0 := result[0]
	if r0.Status != "valid" {
		t.Errorf("blank-qty row expected valid, got %q (errors: %v)", r0.Status, r0.Errors)
	}
	if r0.Quantity != 0 {
		t.Errorf("blank-qty row expected qty=0, got %v", r0.Quantity)
	}
	if len(r0.Warnings) == 0 {
		t.Error("blank-qty row expected at least one warning")
	} else {
		found := false
		for _, w := range r0.Warnings {
			if strings.Contains(w, "planirana količina je prazna") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("blank-qty row warning did not mention blank quantity: %v", r0.Warnings)
		}
	}

	// Row 1: non-blank qty should have no warnings about qty
	r1 := result[1]
	if r1.Status != "valid" {
		t.Errorf("row 1 expected valid, got %q", r1.Status)
	}
	if r1.Quantity != 100 {
		t.Errorf("row 1 expected qty=100, got %v", r1.Quantity)
	}
}

func TestParseRowsWithMapping_NegativeQtyIsError(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(row(2, cells("Naziv", "Cement", "Količina", "-5", "JM", "kg")))
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0].Status != "invalid" {
		t.Errorf("negative qty expected invalid status, got %q", result[0].Status)
	}
}

func TestParseRowsWithMapping_MissingMaterialName(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(row(2, cells("Naziv", "", "Količina", "5", "JM", "kom")))
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0].Status != "invalid" {
		t.Errorf("missing name expected invalid, got %q (errs: %v)", result[0].Status, result[0].Errors)
	}
}

func TestParseRowsWithMapping_MissingUnit(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(row(2, cells("Naziv", "Cement", "Količina", "5", "JM", "")))
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0].Status != "invalid" {
		t.Errorf("missing unit expected invalid, got %q (errs: %v)", result[0].Status, result[0].Errors)
	}
}

func TestParseRowsWithMapping_CategorySkipped(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(
		row(2, cells("Naziv", "I. BETONSKI RADOVI", "Količina", "", "JM", "")),
		row(3, cells("Naziv", "Beton C20/25", "Količina", "10", "JM", "m3")),
	)
	result, stats := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 item row (category skipped), got %d", len(result))
	}
	if stats.SkippedCategories != 1 {
		t.Errorf("expected 1 skipped category, got %d", stats.SkippedCategories)
	}
	// Category should propagate to item
	if result[0].Category == nil || *result[0].Category != "I. BETONSKI RADOVI" {
		t.Errorf("item should inherit category heading, got %v", result[0].Category)
	}
}

func TestParseRowsWithMapping_SubtotalSkipped(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(
		row(2, cells("Naziv", "Cement", "Količina", "5", "JM", "kg")),
		row(3, cells("Naziv", "Ukupno", "Količina", "5", "JM", "")),
	)
	result, stats := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 item row (subtotal skipped), got %d", len(result))
	}
	if stats.SkippedSubtotals != 1 {
		t.Errorf("expected 1 skipped subtotal, got %d", stats.SkippedSubtotals)
	}
}

func TestParseRowsWithMapping_BlankRowSkipped(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(
		row(2, cells("Naziv", "Cement", "Količina", "5", "JM", "kg")),
		row(3, cells("Naziv", "", "Količina", "", "JM", "")),
		row(4, cells("Naziv", "Pijesak", "Količina", "2", "JM", "t")),
	)
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 2 {
		t.Fatalf("expected 2 rows (blank skipped), got %d", len(result))
	}
}

func TestParseRowsWithMapping_ContinuationMergedIntoNote(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(
		row(2, cells("Naziv", "Beton C20/25", "Količina", "5", "JM", "m3")),
		row(3, cells("Naziv", "- dobava i ugradnja", "Količina", "", "JM", "")),
	)
	result, stats := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 item row (continuation merged), got %d", len(result))
	}
	if stats.SkippedContinuations != 1 {
		t.Errorf("expected 1 skipped continuation, got %d", stats.SkippedContinuations)
	}
	if result[0].Note == nil || !strings.Contains(*result[0].Note, "dobava i ugradnja") {
		t.Errorf("continuation text should be merged into note: %v", result[0].Note)
	}
}

func TestParseRowsWithMapping_DecimalCommaQty(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(row(2, cells("Naziv", "Plocice", "Količina", "12,50", "JM", "m2")))
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0].Quantity != 12.5 {
		t.Errorf("expected qty=12.5, got %v", result[0].Quantity)
	}
	if result[0].Status != "valid" {
		t.Errorf("expected valid, got %q", result[0].Status)
	}
}

func TestParseRowsWithMapping_InvalidQtyText(t *testing.T) {
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(row(2, cells("Naziv", "Cement", "Količina", "abc", "JM", "kg")))
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0].Status != "invalid" {
		t.Errorf("invalid qty text expected invalid status, got %q", result[0].Status)
	}
}

func TestParseRowsWithMapping_NameOnlyRowBecomesItem(t *testing.T) {
	// After classifyRow fix: name-only (no qty, no unit) → RowTypeItem (not silently category)
	mp := mapping("Naziv", FieldMaterialName, "Količina", FieldQuantity, "JM", FieldUnit)
	rs := rows(row(2, cells("Naziv", "Betonska opeka", "Količina", "", "JM", "")))
	result, _ := parseRowsWithMapping(rs, mp)
	if len(result) != 1 {
		t.Fatalf("expected 1 item row (name-only), got %d", len(result))
	}
	if result[0].RowType != RowTypeItem {
		t.Errorf("name-only row should be item, got %q", result[0].RowType)
	}
	// Status invalid because unit is missing
	if result[0].Status != "invalid" {
		t.Errorf("name-only row (no unit) expected invalid status, got %q", result[0].Status)
	}
}

func TestParseRowsWithMapping_EmptyMapping(t *testing.T) {
	// No columns mapped → all rows are blank (all fields empty)
	rs := rows(
		row(2, cells("Naziv", "Cement", "Količina", "5", "JM", "kg")),
	)
	result, _ := parseRowsWithMapping(rs, map[string]string{})
	if len(result) != 0 {
		t.Errorf("empty mapping should produce 0 items, got %d", len(result))
	}
}

// ── validateConfirmRow ────────────────────────────────────────────────────────

func TestValidateConfirmRow_ZeroQtyIsValid(t *testing.T) {
	row := dto.WizardConfirmRow{
		RowNumber:    1,
		MaterialName: "Cement",
		Quantity:     0,
		Unit:         "kg",
		Include:      true,
	}
	errs := validateConfirmRow(row)
	if len(errs) != 0 {
		t.Errorf("qty=0 should be valid, got errors: %v", errs)
	}
}

func TestValidateConfirmRow_NegativeQtyIsInvalid(t *testing.T) {
	row := dto.WizardConfirmRow{
		RowNumber:    1,
		MaterialName: "Cement",
		Quantity:     -1,
		Unit:         "kg",
		Include:      true,
	}
	errs := validateConfirmRow(row)
	if len(errs) == 0 {
		t.Error("negative qty should be invalid")
	}
}

func TestValidateConfirmRow_PositiveQtyIsValid(t *testing.T) {
	row := dto.WizardConfirmRow{
		RowNumber:    1,
		MaterialName: "Cement",
		Quantity:     5,
		Unit:         "kg",
		Include:      true,
	}
	errs := validateConfirmRow(row)
	if len(errs) != 0 {
		t.Errorf("positive qty should be valid, got errors: %v", errs)
	}
}

func TestValidateConfirmRow_MissingNameIsInvalid(t *testing.T) {
	row := dto.WizardConfirmRow{
		RowNumber: 1,
		Quantity:  5,
		Unit:      "kg",
		Include:   true,
	}
	errs := validateConfirmRow(row)
	if len(errs) == 0 {
		t.Error("missing material name should be invalid")
	}
}

// ── Integration test: priority workbook ──────────────────────────────────────

func TestAnalyzeExcel_PriorityWorkbook(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/excel/sablonska-tablica-troskovnika.xlsx")
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}

	result, err := AnalyzeExcel(data)
	if err != nil {
		t.Fatalf("AnalyzeExcel failed: %v", err)
	}

	// Build mapping from auto-detected suggestions
	mp := make(map[string]string, len(result.Headers))
	for _, h := range result.Headers {
		if h.SuggestedField != nil {
			mp[h.Original] = *h.SuggestedField
		} else {
			mp[h.Original] = "ignore"
		}
	}

	// Verify key columns were detected
	hasMaterialName := false
	hasQuantity := false
	hasUnit := false
	for _, h := range result.Headers {
		if h.SuggestedField == nil {
			continue
		}
		switch *h.SuggestedField {
		case FieldMaterialName:
			hasMaterialName = true
		case FieldQuantity:
			hasQuantity = true
		case FieldUnit:
			hasUnit = true
		}
	}
	if !hasMaterialName {
		t.Error("auto-detection missed material_name column")
	}
	if !hasQuantity {
		t.Error("auto-detection missed quantity column")
	}
	if !hasUnit {
		t.Error("auto-detection missed unit column")
	}

	// Build rawMappingRow slice from the result
	inputs := make([]rawMappingRow, len(result.RawRows))
	for i, c := range result.RawRows {
		inputs[i] = rawMappingRow{RowNumber: result.DataRowOffset + i, Cells: c}
	}

	rows, stats := parseRowsWithMapping(inputs, mp)

	// All 279 material rows must be detected
	const wantTotal = 279
	if len(rows) != wantTotal {
		t.Errorf("expected %d item rows, got %d (categories=%d, continuations=%d, subtotals=%d)",
			wantTotal, len(rows), stats.SkippedCategories, stats.SkippedContinuations, stats.SkippedSubtotals)
	}

	// Count blank-qty rows: expect 102
	blankQty := 0
	for _, r := range rows {
		for _, w := range r.Warnings {
			if strings.Contains(w, "planirana količina je prazna") {
				blankQty++
				break
			}
		}
	}
	const wantBlankQty = 102
	if blankQty != wantBlankQty {
		t.Errorf("expected %d blank-qty warnings, got %d", wantBlankQty, blankQty)
	}

	// No row should have qty < 0
	for _, r := range rows {
		if r.Quantity < 0 {
			t.Errorf("row %d has negative quantity %v", r.RowNumber, r.Quantity)
		}
	}

	// All rows must be valid (no errors)
	invalidCount := 0
	for _, r := range rows {
		if r.Status == "invalid" {
			invalidCount++
		}
	}
	if invalidCount > 0 {
		t.Errorf("expected 0 invalid rows, got %d", invalidCount)
		// Print first few invalid rows for debugging
		shown := 0
		for _, r := range rows {
			if r.Status == "invalid" && shown < 5 {
				t.Logf("  invalid row %d: name=%q unit=%q qty=%v errors=%v",
					r.RowNumber, r.MaterialName, r.Unit, r.Quantity, r.Errors)
				shown++
			}
		}
	}
}
