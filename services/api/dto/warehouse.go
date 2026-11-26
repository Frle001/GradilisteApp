package dto

import "time"

// ── Overview ──────────────────────────────────────────────────────────────────

type WarehouseOverview struct {
	TotalItems      int     `json:"total_items"`
	TotalValue      float64 `json:"total_value"`
	LowStockItems   int     `json:"low_stock_items"`
	RecentReceipts  int     `json:"recent_receipts"`
	RecentMovements int     `json:"recent_movements"`
}

// ── Stock ─────────────────────────────────────────────────────────────────────

type WarehouseStockItem struct {
	ID           string    `json:"id"`
	MaterialName string    `json:"material_name"`
	MaterialCode *string   `json:"material_code"`
	Unit         string    `json:"unit"`
	Quantity     float64   `json:"quantity"`
	AvgCost      float64   `json:"avg_cost"`
	TotalValue   float64   `json:"total_value"`
	Active       bool      `json:"active"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type WarehouseStockDetail struct {
	WarehouseStockItem
	Movements []WarehouseMovementItem `json:"movements"`
}

// ── Movements ─────────────────────────────────────────────────────────────────

type WarehouseMovementItem struct {
	ID             string    `json:"id"`
	WarehouseBalID string    `json:"warehouse_balance_id"`
	MaterialName   string    `json:"material_name"`
	Unit           string    `json:"unit"`
	MovementType   string    `json:"movement_type"`
	QuantityDelta  float64   `json:"quantity_delta"`
	QuantityBefore float64   `json:"quantity_before"`
	QuantityAfter  float64   `json:"quantity_after"`
	UnitPrice      float64   `json:"unit_price"`
	AvgCostBefore  float64   `json:"avg_cost_before"`
	AvgCostAfter   float64   `json:"avg_cost_after"`
	ProjectID      *string   `json:"project_id"`
	ProjectName    *string   `json:"project_name"`
	Notes          *string   `json:"notes"`
	CreatedByName  string    `json:"created_by_name"`
	CreatedAt      time.Time `json:"created_at"`
}

type WarehouseMovementListResponse struct {
	Data       []WarehouseMovementItem `json:"data"`
	Pagination WarehousePagination     `json:"pagination"`
}

type WarehousePagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ── Receipts (incoming goods) ─────────────────────────────────────────────────

type CreateReceiptLineRequest struct {
	MaterialName string   `json:"material_name" binding:"required"`
	MaterialCode *string  `json:"material_code"`
	Unit         string   `json:"unit" binding:"required"`
	Quantity     float64  `json:"quantity" binding:"required,gt=0"`
	UnitPrice    *float64 `json:"unit_price" binding:"omitempty,min=0"`
}

type CreateReceiptRequest struct {
	ReceiptNumber *string                    `json:"receipt_number"`
	Supplier      *string                    `json:"supplier"`
	ReceiptDate   *string                    `json:"receipt_date"` // YYYY-MM-DD
	Notes         *string                    `json:"notes"`
	Lines         []CreateReceiptLineRequest `json:"lines" binding:"required,min=1,dive"`
}

type ReceiptResponse struct {
	ID            string    `json:"id"`
	ReceiptNumber *string   `json:"receipt_number"`
	Supplier      *string   `json:"supplier"`
	ReceiptDate   string    `json:"receipt_date"`
	Notes         *string   `json:"notes"`
	LineCount     int       `json:"line_count"`
	TotalValue    float64   `json:"total_value"`
	CreatedAt     time.Time `json:"created_at"`
}

// ── Issues (outgoing to project) ──────────────────────────────────────────────

type CreateIssueRequest struct {
	BalanceID string  `json:"balance_id"  binding:"required"`
	ProjectID string  `json:"project_id"  binding:"required"`
	Quantity  float64 `json:"quantity"    binding:"required,gt=0"`
	Notes     *string `json:"notes"`
}

// ── Returns (from project to warehouse) ──────────────────────────────────────

type CreateReturnRequest struct {
	BalanceID         string  `json:"balance_id"         binding:"required"`
	ProjectID         string  `json:"project_id"         binding:"required"`
	ProjectMaterialID string  `json:"project_material_id" binding:"required"`
	Quantity          float64 `json:"quantity"           binding:"required,gt=0"`
	Notes             *string `json:"notes"`
}

// ── Adjustments ───────────────────────────────────────────────────────────────

type CreateAdjustmentRequest struct {
	BalanceID   string  `json:"balance_id"   binding:"required"`
	NewQuantity float64 `json:"new_quantity" binding:"min=0"`
	Notes       *string `json:"notes"`
}

type MovementIDResponse struct {
	ID string `json:"id"`
}
