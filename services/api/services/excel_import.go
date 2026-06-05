package services

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/xuri/excelize/v2"

	"github.com/gradiliste/api/dto"
)

// ── Field constants ───────────────────────────────────────────────────────────

const (
	FieldMaterialName = "material_name"
	FieldQuantity     = "quantity"
	FieldUnit         = "unit"
	FieldUnitPrice    = "unit_price"
	FieldTotalPrice   = "total_price"
	FieldCategory     = "category"
	FieldNote         = "note"
	FieldPosition     = "position"
)

// allKnownFields is the ordered list of importable fields (order matters for tie-breaking).
var allKnownFields = []string{
	FieldMaterialName, FieldQuantity, FieldUnit,
	FieldUnitPrice, FieldTotalPrice, FieldCategory, FieldNote, FieldPosition,
}

// fieldAliases maps each internal field to pre-normalized alias strings.
// All aliases must already be in their normalized form: lowercase ASCII, spaces only,
// no punctuation, no Croatian characters, ² → "2", € → "eur", / → " ".
var fieldAliases = map[string][]string{
	FieldMaterialName: {
		"materijal", "naziv materijala", "naziv", "opis", "opis stavke",
		"opis pozicije", "opis radova", "radovi", "vrsta radova",
		"roba", "artikl", "proizvod", "usluga",
		"item", "item name", "material", "material name",
		"naziv robe", "opis rada", "opis usluge", "description",
	},
	FieldQuantity: {
		"kolicina", "kol", "kolic", "qty", "quantity", "amount",
		"obim", "mjera", "izmjera", "planirano", "planirana kolicina",
	},
	FieldUnit: {
		"jm", "j m", "jed mjere", "jed mj", "mj", "m j",
		"jedinica", "jedinica mjere", "unit", "measure", "mjerna jedinica",
		"jed mj", "j mj",
	},
	FieldUnitPrice: {
		"cijena", "jed cijena", "j cijena", "jedinicna cijena", "cijena eur", "price eur",
		"cijena po jedinici", "cijena jm", "cijena po jm",
		"eur jm", "usd jm", "price", "unit price", "jednicna cijena",
	},
	FieldTotalPrice: {
		"ukupno", "ukupna cijena", "uk cijena", "iznos", "vrijednost",
		"ukupna vrijednost", "ukupan iznos", "sveukupno",
		"total", "total price", "amount eur",
	},
	FieldCategory: {
		"kategorija", "grupa", "grupa radova", "group", "category",
		"tip", "vrsta", "poglavlje", "section",
	},
	FieldNote: {
		"napomena", "opis dodatno", "komentar", "note", "comment", "biljeska",
	},
	FieldPosition: {
		"poz", "rbr", "r br", "redni broj", "pozicija", "broj stavke",
		"stavka", "pos", "position", "item no", "item number", "br stavke",
	},
}

// knownUnits is the normalized set of unit-of-measure strings for value-based scoring.
var knownUnits = map[string]bool{
	"kom": true, "kg": true, "t": true, "tona": true,
	"m": true, "m1": true, "m2": true, "m3": true, "lm": true, "km": true,
	"l": true, "lit": true,
	"h": true, "sat": true, "sati": true, "dan": true,
	"pausal": true, "set": true, "par": true, "kos": true, "kpl": true,
}

// constructionWords are pre-normalized Croatian construction terms used to boost material_name scoring.
var constructionWords = []string{
	"beton", "cement", "armatura", "oplata", "iskop", "nasip",
	"zbuka", "plocice", "cijev", "kabel", "izolacija", "hidroizolacija",
	"fasada", "vrata", "prozor", "dobava", "ugradnja", "radovi",
	"betoniranje", "armirani", "polaganje", "montaza", "sanacija",
	"rekonstrukcija", "demontaza", "ojacanje", "zidanje", "zid",
	"temelj", "strop", "ploce", "stubiste", "balkon", "terasa",
}

// ── Header normalization ──────────────────────────────────────────────────────

// normalizeHeader converts an Excel column header into a canonical form for alias matching:
//   - lowercase + trim
//   - ² → "2", ³ → "3", € → "eur", $ → "usd" (before stripping)
//   - Croatian characters: č/ć → c, š → s, đ → d, ž → z
//   - everything that is not a letter or digit becomes a space
//   - collapse multiple spaces
//
// Examples: "Jed. cijena (€/jm)" → "jed cijena eur jm"
//
//	"J.M."  → "j m"     "KOL."  → "kol"   "m²"    → "m2"
func normalizeHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	// Symbol expansions (must happen before stripping non-letters)
	s = strings.ReplaceAll(s, "²", "2")
	s = strings.ReplaceAll(s, "³", "3")
	s = strings.ReplaceAll(s, "€", "eur")
	s = strings.ReplaceAll(s, "$", "usd")

	// Croatian → ASCII
	s = strings.NewReplacer(
		"č", "c", "ć", "c",
		"š", "s", "đ", "d",
		"ž", "z",
	).Replace(s)

	// Keep only letters and digits; everything else becomes a space
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(r)
		} else {
			buf.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(buf.String()), " ")
}

// ── Header row detection ──────────────────────────────────────────────────────

// detectHeaderRow scans the first up to 15 rows of the sheet and returns the 0-based
// index of the row that most likely contains column headers, based on alias match count.
func detectHeaderRow(rows [][]string) int {
	const maxScan = 15
	limit := min(len(rows), maxScan)

	bestIdx := 0
	bestScore := -1

	for i := 0; i < limit; i++ {
		score := 0
		for _, cell := range rows[i] {
			if strings.TrimSpace(cell) == "" {
				continue
			}
			score++ // bonus per non-empty cell
			norm := normalizeHeader(cell)
			for _, f := range allKnownFields {
				for _, a := range fieldAliases[f] {
					if norm == a || strings.Contains(norm, a) || strings.Contains(a, norm) {
						score += 3
						break
					}
				}
			}
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	return bestIdx
}

// ── Value-based scoring helpers ───────────────────────────────────────────────

type colStats struct {
	nonEmpty          int
	numericCount      int
	numericSum        float64
	unitCount         int
	textLenSum        int
	constructionCount int
}

func computeColStats(values []string) colStats {
	var s colStats
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		s.nonEmpty++
		s.textLenSum += len([]rune(v))

		if n, err := parseNumber(v); err == nil {
			s.numericCount++
			s.numericSum += n
		}

		if knownUnits[normalizeHeader(v)] {
			s.unitCount++
		}

		vn := normalizeHeader(v)
		for _, w := range constructionWords {
			if strings.Contains(vn, w) {
				s.constructionCount++
				break
			}
		}
	}
	return s
}

type fieldBonus struct {
	bonus  float64
	reason string
}

func computeValueBonuses(stats colStats) map[string]fieldBonus {
	result := make(map[string]fieldBonus)
	if stats.nonEmpty == 0 {
		return result
	}

	numFrac := float64(stats.numericCount) / float64(stats.nonEmpty)
	unitFrac := float64(stats.unitCount) / float64(stats.nonEmpty)
	avgLen := float64(stats.textLenSum) / float64(stats.nonEmpty)

	// material_name: long text, not mostly numeric
	if numFrac < 0.25 && avgLen > 10 {
		bonus := 0.10
		reason := fmt.Sprintf("Values are long text (avg %.0f chars)", avgLen)
		if stats.constructionCount > 0 {
			bonus = 0.14
			reason += ", contains construction terms"
		}
		result[FieldMaterialName] = fieldBonus{bonus, reason}
	} else if numFrac < 0.2 && avgLen > 5 {
		result[FieldMaterialName] = fieldBonus{0.05, "Values are mostly text"}
	}

	// unit: known unit strings are very distinctive
	if unitFrac >= 0.50 {
		result[FieldUnit] = fieldBonus{0.18, fmt.Sprintf("%.0f%% of values are known unit strings", unitFrac*100)}
	} else if unitFrac >= 0.25 {
		result[FieldUnit] = fieldBonus{0.08, "Some values are known unit strings"}
	}

	// numeric columns: quantity vs price — use average value and avg length to distinguish
	if numFrac >= 0.70 {
		avgVal := 0.0
		if stats.numericCount > 0 {
			avgVal = stats.numericSum / float64(stats.numericCount)
		}

		// Quantity: moderate values, shorter strings
		if avgVal > 0 && avgVal < 5000 && avgLen < 8 {
			result[FieldQuantity] = fieldBonus{0.08, fmt.Sprintf("%.0f%% numeric, avg value %.1f (quantity-like)", numFrac*100, avgVal)}
		}
		// Price fields get a small bonus; relation check will further disambiguate
		if avgVal > 0 {
			result[FieldUnitPrice] = fieldBonus{0.06, "Values are numeric (price-like)"}
			result[FieldTotalPrice] = fieldBonus{0.05, "Values are numeric (possible total)"}
		}
	}

	return result
}

// ── Per-column scoring ────────────────────────────────────────────────────────

type fieldScore struct {
	headerScore float64
	valueBonus  float64
	total       float64
	reasons     []string
}

func scoreAllFields(normalized string, stats colStats) map[string]fieldScore {
	scores := make(map[string]fieldScore, len(allKnownFields))

	for _, f := range allKnownFields {
		var fs fieldScore

		// Exact alias match
		for _, a := range fieldAliases[f] {
			if normalized == a {
				fs.headerScore = 0.90
				fs.reasons = append(fs.reasons, "Header matches alias: "+a)
				break
			}
		}
		// Contains match (only if no exact match)
		if fs.headerScore == 0 {
			for _, a := range fieldAliases[f] {
				if strings.Contains(normalized, a) || strings.Contains(a, normalized) {
					fs.headerScore = 0.65
					fs.reasons = append(fs.reasons, "Header contains: "+a)
					break
				}
			}
		}

		scores[f] = fs
	}

	// Add value bonuses
	for f, b := range computeValueBonuses(stats) {
		fs := scores[f]
		fs.valueBonus = b.bonus
		fs.reasons = append(fs.reasons, b.reason)
		scores[f] = fs
	}

	// Compute totals
	for f, fs := range scores {
		fs.total = math.Min(fs.headerScore+fs.valueBonus, 0.98)
		scores[f] = fs
	}

	return scores
}

func bestField(scores map[string]fieldScore) (field string, total float64, reasons []string) {
	for _, f := range allKnownFields { // ordered iteration for deterministic tie-breaking
		fs := scores[f]
		if fs.total > total {
			total = fs.total
			field = f
			reasons = fs.reasons
		}
	}
	return
}

func confidenceLabelGo(conf float64) string {
	switch {
	case conf >= 0.85:
		return "high"
	case conf >= 0.60:
		return "medium"
	case conf > 0:
		return "low"
	default:
		return "none"
	}
}

// ── Troškovnik row classification ─────────────────────────────────────────────

const (
	RowTypeBlank        = "blank"
	RowTypeCategory     = "category"
	RowTypeItem         = "item"
	RowTypeContinuation = "continuation"
	RowTypeSubtotal     = "subtotal"
)

// trosParseSummary holds row-type counters produced by parseRowsWithMapping.
type trosParseSummary struct {
	TotalScanned         int
	SkippedCategories    int
	SkippedContinuations int
	SkippedSubtotals     int
}

// nonPricePhrases are non-numeric strings that can appear in price cells.
var nonPricePhrases = []string{
	"ne nudimo", "nenudimo", "ne nudi", "bez ponude", "n/a",
}

// isNonPriceText returns true if a price cell contains a "no bid" phrase instead of a number.
func isNonPriceText(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, p := range nonPricePhrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return strings.Trim(lower, "- /.") == ""
}

// normalizeUnit maps common unit-of-measure abbreviations to canonical lowercase forms.
// Examples: "kom." → "kom", "m²" → "m2", "paušal" → "pausal"
func normalizeUnit(u string) string {
	u = strings.TrimSpace(u)
	u = strings.ReplaceAll(u, "²", "2")
	u = strings.ReplaceAll(u, "³", "3")
	u = strings.NewReplacer(
		"š", "s", "Š", "s",
		"č", "c", "Č", "c",
		"ć", "c", "Ć", "c",
		"đ", "d", "Đ", "d",
		"ž", "z", "Ž", "z",
	).Replace(u)
	u = strings.TrimRight(u, ". ")
	lower := strings.ToLower(u)
	switch lower {
	case "pausal", "pausalno", "pausalan":
		return "pausal"
	case "litar", "litr", "litra", "lit":
		return "l"
	case "kos":
		return "kom"
	}
	return lower
}

// isCategoryLike returns true when a name looks like a section heading:
// predominantly uppercase letters, or starts with a Roman numeral prefix.
func isCategoryLike(name string) bool {
	letters, upper := 0, 0
	for _, r := range name {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if letters > 3 && float64(upper)/float64(letters) >= 0.75 {
		return true
	}
	trimmed := strings.TrimSpace(name)
	for _, prefix := range []string{
		"I.", "II.", "III.", "IV.", "V.", "VI.", "VII.", "VIII.", "IX.", "X.",
		"XI.", "XII.",
	} {
		if strings.HasPrefix(trimmed, prefix) && len(trimmed) > len(prefix) {
			return true
		}
	}
	return false
}

// classifyRow determines the row type from the resolved mapped field values.
func classifyRow(name, qtyStr, unit string, prevWasItem bool) string {
	name = strings.TrimSpace(name)
	qtyStr = strings.TrimSpace(qtyStr)
	unit = strings.TrimSpace(unit)

	if name == "" && qtyStr == "" && unit == "" {
		return RowTypeBlank
	}

	nameLower := strings.ToLower(name)
	if strings.Contains(nameLower, "ukupno") || strings.Contains(nameLower, "sveukupno") {
		return RowTypeSubtotal
	}

	_, qtyErr := parseNumber(qtyStr)
	hasNumericQty := qtyStr != "" && qtyErr == nil
	hasUnit := unit != ""

	if hasNumericQty || hasUnit {
		return RowTypeItem
	}

	if name != "" {
		if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "•") || strings.HasPrefix(name, "*") {
			return RowTypeContinuation
		}
		if isCategoryLike(name) {
			return RowTypeCategory
		}
		if prevWasItem {
			return RowTypeContinuation
		}
		return RowTypeCategory
	}

	return RowTypeBlank
}

// ── Relation check ────────────────────────────────────────────────────────────

// tryRelationCheck verifies (or disambiguates) unit_price / total_price assignments
// by testing whether quantity × unit_price ≈ total_price across sample rows.
// It modifies suggestions in-place: swaps field assignments if needed, boosts confidence.
func tryRelationCheck(suggestions []dto.AnalyzeHeaderSuggestion, colValues [][]string) {
	// Locate quantity column
	qtyIdx := -1
	for i, s := range suggestions {
		if s.SuggestedField != nil && *s.SuggestedField == FieldQuantity {
			if qtyIdx == -1 || s.Confidence > suggestions[qtyIdx].Confidence {
				qtyIdx = i
			}
		}
	}
	if qtyIdx == -1 || qtyIdx >= len(colValues) {
		return
	}

	// Collect price-field candidates
	type priceCandidate struct {
		idx   int
		field string
		conf  float64
	}
	var priceCands []priceCandidate
	for i, s := range suggestions {
		if s.SuggestedField == nil || i >= len(colValues) {
			continue
		}
		f := *s.SuggestedField
		if f == FieldUnitPrice || f == FieldTotalPrice {
			priceCands = append(priceCands, priceCandidate{i, f, s.Confidence})
		}
	}
	if len(priceCands) < 2 {
		return
	}

	// Test all pairs: find which ordering best satisfies qty * A ≈ B
	for ai := 0; ai < len(priceCands); ai++ {
		for bi := ai + 1; bi < len(priceCands); bi++ {
			a, b := priceCands[ai], priceCands[bi]

			scoreAB := testRelation(colValues[qtyIdx], colValues[a.idx], colValues[b.idx]) // A=unit, B=total
			scoreBA := testRelation(colValues[qtyIdx], colValues[b.idx], colValues[a.idx]) // B=unit, A=total

			const minMatchRate = 0.40
			if scoreAB < minMatchRate && scoreBA < minMatchRate {
				continue
			}

			// Pick whichever ordering has a higher match rate
			upIdx, tpIdx, matchRate := a.idx, b.idx, scoreAB
			if scoreBA > scoreAB {
				upIdx, tpIdx, matchRate = b.idx, a.idx, scoreBA
			}

			reason := fmt.Sprintf("Relation check: qty × unit_price ≈ total_price (%.0f%% of rows)", matchRate*100)
			upField := FieldUnitPrice
			tpField := FieldTotalPrice

			suggestions[upIdx].SuggestedField = &upField
			suggestions[upIdx].Confidence = math.Min(suggestions[upIdx].Confidence+0.05, 0.98)
			suggestions[upIdx].Reasons = append(suggestions[upIdx].Reasons, reason)

			suggestions[tpIdx].SuggestedField = &tpField
			suggestions[tpIdx].Confidence = math.Min(suggestions[tpIdx].Confidence+0.05, 0.98)
			suggestions[tpIdx].Reasons = append(suggestions[tpIdx].Reasons, reason)

			return
		}
	}
}

// testRelation returns the fraction of aligned rows where qty * unitPrice ≈ totalPrice
// (within 6% relative tolerance). Empty strings are skipped.
func testRelation(qtyVals, upVals, tpVals []string) float64 {
	n := min(min(len(qtyVals), len(upVals)), len(tpVals))
	matches, checks := 0, 0
	for i := 0; i < n; i++ {
		qty, qErr := parseNumber(qtyVals[i])
		up, uErr := parseNumber(upVals[i])
		tp, tErr := parseNumber(tpVals[i])
		if qErr != nil || uErr != nil || tErr != nil || qty == 0 || up == 0 || tp == 0 {
			continue
		}
		checks++
		if math.Abs(qty*up-tp)/math.Max(math.Abs(tp), 0.01) < 0.06 {
			matches++
		}
	}
	if checks < 2 {
		return 0
	}
	return float64(matches) / float64(checks)
}

// ── Deduplication ─────────────────────────────────────────────────────────────

// deduplicateSuggestions ensures each internal field is suggested for at most one column.
// When two columns compete for the same field, the one with higher confidence wins;
// the other has its SuggestedField cleared.
func deduplicateSuggestions(suggestions []dto.AnalyzeHeaderSuggestion) {
	fieldBest := make(map[string]int) // field → index of winning suggestion

	for i := range suggestions {
		if suggestions[i].SuggestedField == nil {
			continue
		}
		f := *suggestions[i].SuggestedField
		prev, exists := fieldBest[f]
		if !exists {
			fieldBest[f] = i
		} else if suggestions[i].Confidence > suggestions[prev].Confidence {
			suggestions[prev].SuggestedField = nil
			fieldBest[f] = i
		} else {
			suggestions[i].SuggestedField = nil
		}
	}
}

// ── AnalyzeResult ─────────────────────────────────────────────────────────────

// AnalyzeResult is returned by AnalyzeExcel.
type AnalyzeResult struct {
	SheetName       string
	HeaderRowNumber int // 1-based Excel row number of the detected header row
	DataRowOffset   int // 1-based Excel row number of the first data row
	Headers         []dto.AnalyzeHeaderSuggestion
	SampleRows      []dto.AnalyzeSampleRow
	// RawRows holds every non-blank data row keyed by original column header.
	// Row order matches DataRowOffset + index.
	RawRows []map[string]string
}

// ── AnalyzeExcel ──────────────────────────────────────────────────────────────

// AnalyzeExcel reads an xlsx file, auto-detects the header row, scores each column
// using header aliases and sample-value heuristics, and returns mapping suggestions
// plus raw row data for the subsequent wizard steps.
func AnalyzeExcel(data []byte) (*AnalyzeResult, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("cannot open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("Excel file has no sheets")
	}
	sheetName := sheets[0]

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("cannot read sheet: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("Excel file is empty or has only one row")
	}

	// Detect header row
	headerIdx := detectHeaderRow(rows)
	if headerIdx >= len(rows)-1 {
		return nil, errors.New("no data rows found after the detected header row")
	}

	headerRow := rows[headerIdx]
	dataRows := rows[headerIdx+1:]

	if len(headerRow) == 0 {
		return nil, errors.New("detected header row has no cells")
	}

	// Build aligned per-column value arrays (first 20 data rows, preserving row alignment)
	const maxValueSample = 20
	sampleCount := min(len(dataRows), maxValueSample)
	colValues := make([][]string, len(headerRow))
	for j := range headerRow {
		colValues[j] = make([]string, sampleCount)
		for i := 0; i < sampleCount; i++ {
			colValues[j][i] = cellVal(dataRows[i], j)
		}
	}

	// Score each column
	suggestions := make([]dto.AnalyzeHeaderSuggestion, len(headerRow))
	for j, h := range headerRow {
		norm := normalizeHeader(h)

		if norm == "" {
			suggestions[j] = dto.AnalyzeHeaderSuggestion{
				Index:           j,
				Original:        h,
				Normalized:      "",
				Confidence:      0,
				ConfidenceLabel: "none",
				Reasons:         []string{},
			}
			continue
		}

		stats := computeColStats(colValues[j])
		fieldScores := scoreAllFields(norm, stats)
		bestF, bestConf, bestReasons := bestField(fieldScores)

		s := dto.AnalyzeHeaderSuggestion{
			Index:      j,
			Original:   h,
			Normalized: norm,
			Confidence: bestConf,
			Reasons:    nonNilSlice(bestReasons),
		}
		if bestF != "" && bestConf >= 0.60 {
			s.SuggestedField = &bestF
		}
		suggestions[j] = s
	}

	// Relation check (may swap/boost unit_price vs total_price assignments)
	tryRelationCheck(suggestions, colValues)

	// Deduplicate: each field mapped to at most one column
	deduplicateSuggestions(suggestions)

	// Finalize confidence labels
	for i := range suggestions {
		suggestions[i].ConfidenceLabel = confidenceLabelGo(suggestions[i].Confidence)
		suggestions[i].Reasons = nonNilSlice(suggestions[i].Reasons)
	}

	// Collect raw rows (all non-blank data rows) and sample rows (first 5)
	const maxSample = 5
	var rawRows []map[string]string
	var sampleRows []dto.AnalyzeSampleRow

	for i, row := range dataRows {
		rowNum := headerIdx + 2 + i // actual 1-based Excel row number

		allBlank := true
		for j := range headerRow {
			if cellVal(row, j) != "" {
				allBlank = false
				break
			}
		}
		if allBlank {
			continue
		}

		cells := make(map[string]string, len(headerRow))
		vals := make([]string, len(headerRow))
		for j, h := range headerRow {
			v := cellVal(row, j)
			cells[h] = v
			vals[j] = v
		}
		rawRows = append(rawRows, cells)

		if len(sampleRows) < maxSample {
			sampleRows = append(sampleRows, dto.AnalyzeSampleRow{
				RowNumber: rowNum,
				Values:    vals,
			})
		}
	}

	if len(rawRows) == 0 {
		return nil, errors.New("no data rows found after the header")
	}

	return &AnalyzeResult{
		SheetName:       sheetName,
		HeaderRowNumber: headerIdx + 1,
		DataRowOffset:   headerIdx + 2,
		Headers:         suggestions,
		SampleRows:      nonNilSampleRows(sampleRows),
		RawRows:         rawRows,
	}, nil
}

// ── Wizard preview step ───────────────────────────────────────────────────────

// rawMappingRow pairs an actual Excel row number with its raw cell data.
// Used internally to carry correct row numbers through the wizard preview step.
type rawMappingRow struct {
	RowNumber int
	Cells     map[string]string
}

// parseRowsWithMapping applies the user-selected column mapping to raw rows using a
// troškovnik-aware state machine. It classifies each row (blank / category / item /
// continuation / subtotal), tracks the current category heading, merges continuation
// lines into the previous item's note, and returns only item rows plus parse statistics.
// mapping is {"Original Excel Header": "internal_field | ignore"}.
func parseRowsWithMapping(rows []rawMappingRow, mapping map[string]string) ([]dto.WizardPreviewRow, trosParseSummary) {
	// Build reverse map: internal_field → first Excel header mapped to it
	fieldToCol := make(map[string]string, len(mapping))
	for col, field := range mapping {
		if field == "" || field == "ignore" {
			continue
		}
		if _, exists := fieldToCol[field]; !exists {
			fieldToCol[field] = col
		}
	}

	get := func(cells map[string]string, field string) string {
		col, ok := fieldToCol[field]
		if !ok {
			return ""
		}
		return strings.TrimSpace(cells[col])
	}

	results := make([]dto.WizardPreviewRow, 0, len(rows))
	var stats trosParseSummary
	currentCategory := ""
	prevWasItem := false
	prevItemIdx := -1

	for _, r := range rows {
		stats.TotalScanned++

		name := get(r.Cells, FieldMaterialName)
		qtyStr := get(r.Cells, FieldQuantity)
		unit := get(r.Cells, FieldUnit)

		rowType := classifyRow(name, qtyStr, unit, prevWasItem)

		switch rowType {
		case RowTypeBlank:
			prevWasItem = false
			continue

		case RowTypeSubtotal:
			stats.SkippedSubtotals++
			prevWasItem = false
			continue

		case RowTypeCategory:
			currentCategory = name
			stats.SkippedCategories++
			prevWasItem = false
			continue

		case RowTypeContinuation:
			stats.SkippedContinuations++
			if prevItemIdx >= 0 {
				results[prevItemIdx].SourceRowNumbers = append(results[prevItemIdx].SourceRowNumbers, r.RowNumber)
				extra := strings.TrimLeft(name, "-•* ")
				if extra != "" {
					if results[prevItemIdx].Note == nil {
						results[prevItemIdx].Note = &extra
					} else {
						combined := *results[prevItemIdx].Note + " " + extra
						results[prevItemIdx].Note = &combined
					}
				}
			}
			continue
		}

		// ── RowTypeItem ──────────────────────────────────────────────────────────
		errs := make([]string, 0)
		warnings := make([]string, 0)

		if name == "" {
			errs = append(errs, "Naziv materijala je obavezan")
		}

		unit = normalizeUnit(unit)
		if unit == "" {
			errs = append(errs, "Jedinica mjere je obavezna")
		}

		var qty float64
		if qtyStr == "" {
			errs = append(errs, "Količina je obavezna")
		} else if q, err := parseNumber(qtyStr); err != nil {
			errs = append(errs, "Količina nije valjana: "+qtyStr)
		} else if q < 0 {
			errs = append(errs, "Količina mora biti >= 0")
		} else {
			qty = q
		}

		var unitPrice *float64
		if s := get(r.Cells, FieldUnitPrice); s != "" {
			if isNonPriceText(s) {
				warnings = append(warnings, "Cijena nije ponuđena: "+s)
			} else if v, err := parseNumber(s); err != nil {
				errs = append(errs, "Jedinična cijena nije valjana: "+s)
			} else {
				unitPrice = &v
			}
		}

		var totalPrice *float64
		if s := get(r.Cells, FieldTotalPrice); s != "" {
			if isNonPriceText(s) {
				if !strings.Contains(strings.Join(warnings, ""), "Cijena") {
					warnings = append(warnings, "Cijena nije ponuđena: "+s)
				}
			} else if v, err := parseNumber(s); err != nil {
				errs = append(errs, "Ukupna cijena nije valjana: "+s)
			} else {
				totalPrice = &v
			}
		}

		if totalPrice == nil && unitPrice != nil && qty > 0 {
			v := qty * *unitPrice
			totalPrice = &v
		}

		// Prefer mapped category column; fall back to state-machine currentCategory.
		var category *string
		if s := get(r.Cells, FieldCategory); s != "" {
			category = &s
		} else if currentCategory != "" {
			c := currentCategory
			category = &c
		}

		var note *string
		if s := get(r.Cells, FieldNote); s != "" {
			note = &s
		}

		var position *string
		if s := get(r.Cells, FieldPosition); s != "" {
			position = &s
		}

		status := "valid"
		if len(errs) > 0 {
			status = "invalid"
		}

		results = append(results, dto.WizardPreviewRow{
			RowNumber:        r.RowNumber,
			MaterialName:     name,
			Quantity:         qty,
			Unit:             unit,
			UnitPrice:        unitPrice,
			TotalPrice:       totalPrice,
			Category:         category,
			Note:             note,
			Position:         position,
			SourceRowNumbers: []int{r.RowNumber},
			RowType:          RowTypeItem,
			Warnings:         warnings,
			Errors:           errs,
			Status:           status,
		})
		prevItemIdx = len(results) - 1
		prevWasItem = true
	}

	return results, stats
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func nonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilSampleRows(s []dto.AnalyzeSampleRow) []dto.AnalyzeSampleRow {
	if s == nil {
		return []dto.AnalyzeSampleRow{}
	}
	return s
}

// cellVal safely reads a cell from a row slice, returning "" if out of bounds.
func cellVal(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}

// ── Number parsing ────────────────────────────────────────────────────────────

// parseNumber parses a numeric string tolerating Croatian/EU number formats and currency symbols.
//
// Supported formats:
//
//	"1,25"     → 1.25    "1.25"     → 1.25
//	"1.250,50" → 1250.50 "1 250,50" → 1250.50
//	"€ 4,80"   → 4.80    "4,80 EUR" → 4.80
func parseNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty string")
	}

	// Strip currency prefixes
	for _, p := range []string{"EUR", "USD", "€", "$", "£", "¥"} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimSpace(s[len(p):])
		}
	}
	// Strip currency suffixes
	for _, sfx := range []string{" EUR", " USD", " eur", " usd", " €", " $", "EUR", "USD"} {
		if strings.HasSuffix(s, sfx) {
			s = strings.TrimSpace(s[:len(s)-len(sfx)])
		}
	}

	// Remove spaces (thousands separator)
	s = strings.ReplaceAll(s, " ", "")
	// Comma → dot, then collapse multiple dots (European thousands sep)
	s = strings.ReplaceAll(s, ",", ".")
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		// e.g. "1.250.50" from "1.250,50" → join all-but-last, add decimal
		s = strings.Join(parts[:len(parts)-1], "") + "." + parts[len(parts)-1]
	}

	return strconv.ParseFloat(s, 64)
}

// parseQuantity is kept for backward compatibility.
func parseQuantity(s string) (float64, error) { return parseNumber(s) }

// ── Legacy fixed-alias import (kept for backward compatibility) ───────────────

// findCol finds the first column index whose normalized header matches any alias.
func findCol(header []string, aliases []string) int {
	for i, h := range header {
		norm := normalizeHeader(h)
		for _, a := range aliases {
			if norm == a {
				return i
			}
		}
	}
	return -1
}

// ParsedImport is the result of the legacy ParseExcelMaterials function.
type ParsedImport struct {
	Rows []dto.ImportPreviewRow
}

// ParseExcelMaterials parses an xlsx file using the legacy fixed-alias approach.
// The new wizard flow uses AnalyzeExcel + parseRowsWithMapping instead.
func ParseExcelMaterials(data []byte) (*ParsedImport, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("cannot open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("Excel file has no sheets")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("cannot read sheet: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("Excel file must have a header row and at least one data row")
	}

	header := rows[0]
	colName := findCol(header, fieldAliases[FieldMaterialName])
	colQty := findCol(header, fieldAliases[FieldQuantity])
	colUnit := findCol(header, fieldAliases[FieldUnit])

	if colName < 0 {
		return nil, errors.New("cannot find material name column")
	}
	if colQty < 0 {
		return nil, errors.New("cannot find quantity column")
	}
	if colUnit < 0 {
		return nil, errors.New("cannot find unit column")
	}

	var result []dto.ImportPreviewRow
	for i, row := range rows[1:] {
		rowNum := i + 2
		name := cellVal(row, colName)
		qtyStr := cellVal(row, colQty)
		unit := cellVal(row, colUnit)

		if name == "" && qtyStr == "" && unit == "" {
			continue
		}

		pr := dto.ImportPreviewRow{RowNumber: rowNum}
		var errMsgs []string

		if name == "" {
			errMsgs = append(errMsgs, "naziv materijala je obavezan")
		}
		if unit == "" {
			errMsgs = append(errMsgs, "jedinica mjere je obavezna")
		}

		qty, qtyErr := parseNumber(qtyStr)
		if qtyErr != nil {
			errMsgs = append(errMsgs, "količina nije valjana")
		} else if qty < 0 {
			errMsgs = append(errMsgs, "količina mora biti >= 0")
		}

		if len(errMsgs) > 0 {
			msg := strings.Join(errMsgs, "; ")
			pr.Status = "invalid"
			pr.ErrorMessage = &msg
		} else {
			pr.MaterialName = name
			pr.Quantity = qty
			pr.Unit = unit
			pr.Status = "valid"
		}
		result = append(result, pr)
	}
	return &ParsedImport{Rows: result}, nil
}
