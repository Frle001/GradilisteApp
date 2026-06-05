package dto

import "time"

// ── Inventory view ─────────────────────────────────────────────────────────────

type InventoryEmployeeSummary struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

type InventoryAssetItem struct {
	ID           string    `json:"id"`
	AssetType    string    `json:"asset_type"`
	Name         string    `json:"name"`
	Quantity     float64   `json:"quantity"`
	Unit         *string   `json:"unit"`
	SerialNumber *string   `json:"serial_number"`
	Notes        *string   `json:"notes"`
	AssignedAt   time.Time `json:"assigned_at"`
}

type InventoryMaterialItem struct {
	ProjectMaterialID string  `json:"project_material_id"`
	MaterialName      string  `json:"material_name"`
	MaterialCode      *string `json:"material_code"`
	ProjectID         string  `json:"project_id"`
	ProjectName       string  `json:"project_name"`
	TotalQuantity     float64 `json:"total_quantity"`
	Unit              string  `json:"unit"`
}

type InventoryResponse struct {
	Employee  InventoryEmployeeSummary `json:"employee"`
	Assets    []InventoryAssetItem     `json:"assets"`
	Materials []InventoryMaterialItem  `json:"materials"`
}

// ── Transfer targets ───────────────────────────────────────────────────────────

type TransferTarget struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

// ── Employee active projects ───────────────────────────────────────────────────

type InventoryProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ── Create transfer ────────────────────────────────────────────────────────────

// CreateTransferRequest is bound from the JSON body of POST /api/inventory/transfers.
// Exactly one of EmployeeAssetID or ProjectMaterialID must be set.
type CreateTransferRequest struct {
	ToEmployeeID         string   `json:"to_employee_id"`
	Type                 string   `json:"type"` // "asset" | "material"
	EmployeeAssetID      *string  `json:"employee_asset_id"`
	ProjectMaterialID    *string  `json:"project_material_id"`
	DestinationProjectID *string  `json:"destination_project_id"`
	Quantity             *float64 `json:"quantity"`
	Notes                string   `json:"notes"`
}

// ── Transfer list ──────────────────────────────────────────────────────────────

type TransferListItem struct {
	ID                string    `json:"id"`
	FromEmployeeID    string    `json:"from_employee_id"`
	FromEmployeeName  string    `json:"from_employee_name"`
	ToEmployeeID      string    `json:"to_employee_id"`
	ToEmployeeName    string    `json:"to_employee_name"`
	AssetType         string    `json:"asset_type"`
	ItemName          string    `json:"item_name"`
	Quantity          float64   `json:"quantity"`
	TransferredAt     time.Time `json:"transferred_at"`
	Notes             *string   `json:"notes"`
	TransferredByName string    `json:"transferred_by_name"`
}

type TransferPagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type TransferListResponse struct {
	Data       []TransferListItem `json:"data"`
	Pagination TransferPagination `json:"pagination"`
}
