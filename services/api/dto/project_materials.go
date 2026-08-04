package dto

import "time"

type MaterialListItem struct {
	ID                string    `json:"id"`
	MaterialName      string    `json:"material_name"`
	MaterialCode      *string   `json:"material_code"`
	PlannedQuantity   float64   `json:"planned_quantity"`
	UsedQuantity      float64   `json:"used_quantity"`
	AvailableQuantity float64   `json:"available_quantity"`
	Unit              string    `json:"unit"`
	Source            *string   `json:"source"`
	Active            bool      `json:"active"`
	TrackingType      string    `json:"tracking_type"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CreateMaterialRequest struct {
	MaterialName    string  `json:"material_name"    binding:"required"`
	MaterialCode    *string `json:"material_code"`
	PlannedQuantity float64 `json:"planned_quantity" binding:"min=0"`
	Unit            string  `json:"unit"             binding:"required"`
	TrackingType    string  `json:"tracking_type"` // "stock" (default) or "work"
}

type UpdateMaterialRequest struct {
	MaterialName    string  `json:"material_name"    binding:"required"`
	MaterialCode    *string `json:"material_code"`
	PlannedQuantity float64 `json:"planned_quantity" binding:"min=0"`
	Unit            string  `json:"unit"             binding:"required"`
	TrackingType    string  `json:"tracking_type"` // "stock" (default) or "work"
}

// ── Legacy import types (kept for internal use) ───────────────────────────────

type ImportPreviewRow struct {
	RowNumber    int     `json:"row_number"`
	MaterialName string  `json:"material_name"`
	MaterialCode *string `json:"material_code"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	Status       string  `json:"status"` // valid | invalid
	ErrorMessage *string `json:"error_message"`
}

// ── Phase 6.5 wizard import types ────────────────────────────────────────────

// Step 1 — analyze: detect headers and suggest mappings

type AnalyzeHeaderSuggestion struct {
	Index           int      `json:"index"`
	Original        string   `json:"original"`
	Normalized      string   `json:"normalized"`
	SuggestedField  *string  `json:"suggested_field"`  // nil if confidence < 0.60
	Confidence      float64  `json:"confidence"`       // 0.0–1.0
	ConfidenceLabel string   `json:"confidence_label"` // "high" | "medium" | "low" | "none"
	Reasons         []string `json:"reasons"`          // human-readable explanation
}

type AnalyzeSampleRow struct {
	RowNumber int      `json:"row_number"`
	Values    []string `json:"values"` // cell values in column order
}

type ImportAnalyzeResponse struct {
	ImportJobID     string                    `json:"import_job_id"`
	SheetName       string                    `json:"sheet_name"`
	HeaderRowNumber int                       `json:"header_row_number"` // 1-based
	Headers         []AnalyzeHeaderSuggestion `json:"headers"`
	SampleRows      []AnalyzeSampleRow        `json:"sample_rows"`
}

// Step 2 — wizard preview: parse using user-selected mapping

type WizardPreviewRow struct {
	RowNumber        int      `json:"row_number"`
	MaterialName     string   `json:"material_name"`
	Quantity         float64  `json:"quantity"`
	Unit             string   `json:"unit"`
	UnitPrice        *float64 `json:"unit_price"`
	TotalPrice       *float64 `json:"total_price"`
	Category         *string  `json:"category"`
	Note             *string  `json:"note"`
	Position         *string  `json:"position"`
	SourceRowNumbers []int    `json:"source_row_numbers"`
	RowType          string   `json:"row_type"`
	Warnings         []string `json:"warnings"`
	Errors           []string `json:"errors"`
	Status           string   `json:"status"` // valid | invalid
}

type WizardPreviewSummary struct {
	TotalRows            int `json:"total_rows"`
	ValidRows            int `json:"valid_rows"`
	InvalidRows          int `json:"invalid_rows"`
	TotalScanned         int `json:"total_scanned"`
	SkippedCategories    int `json:"skipped_categories"`
	SkippedContinuations int `json:"skipped_continuations"`
	SkippedSubtotals     int `json:"skipped_subtotals"`
}

type WizardPreviewResponse struct {
	Rows    []WizardPreviewRow   `json:"rows"`
	Summary WizardPreviewSummary `json:"summary"`
}

// Step 3 — confirm: import (optionally edited) rows

// WizardConfirmRow represents one row sent from the frontend during confirm.
// The director may have edited field values before confirming.
type WizardConfirmRow struct {
	RowNumber    int      `json:"row_number"`
	Position     *string  `json:"position"`
	MaterialName string   `json:"material_name"`
	Quantity     float64  `json:"quantity"`
	Unit         string   `json:"unit"`
	UnitPrice    *float64 `json:"unit_price"`
	TotalPrice   *float64 `json:"total_price"`
	Category     *string  `json:"category"`
	Note         *string  `json:"note"`
	Include      bool     `json:"include"`
}

type WizardConfirmRowError struct {
	RowNumber int      `json:"row_number"`
	Errors    []string `json:"errors"`
}

type WizardConfirmResponse struct {
	ImportedCount int                     `json:"imported_count"`
	SkippedCount  int                     `json:"skipped_count"`
	FailedCount   int                     `json:"failed_count"`
	Errors        []WizardConfirmRowError `json:"errors"`
}
