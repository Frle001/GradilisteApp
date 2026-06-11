package dto

import "time"

// ── Create request ─────────────────────────────────────────────────────────────

type CreatePurchaseItemRequest struct {
	ProjectMaterialID string  `json:"project_material_id"`
	Quantity          float64 `json:"quantity"`
	Unit              string  `json:"unit"`
}

type CreatePurchaseRequest struct {
	ProjectID   string
	PurchasedAt time.Time
	Notes       *string
	Items       []CreatePurchaseItemRequest
}

// ── List ──────────────────────────────────────────────────────────────────────

type PurchaseListItem struct {
	ID                      string    `json:"id"`
	ProjectID               string    `json:"project_id"`
	ProjectName             string    `json:"project_name"`
	BuyerID                 string    `json:"buyer_id"`
	BuyerName               string    `json:"buyer_name"`
	PurchasedAt             time.Time `json:"purchased_at"`
	ItemsCount              int       `json:"items_count"`
	ReceiptFileURL          *string   `json:"receipt_file_url"`
	ReceiptOriginalFilename *string   `json:"receipt_original_filename"`
	Notes                   *string   `json:"notes"`
	CreatedAt               time.Time `json:"created_at"`
}

type PurchasePagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type PurchaseListResponse struct {
	Data       []PurchaseListItem `json:"data"`
	Pagination PurchasePagination `json:"pagination"`
}

// ── Detail ────────────────────────────────────────────────────────────────────

type PurchaseItemDetail struct {
	ID                string  `json:"id"`
	ProjectMaterialID string  `json:"project_material_id"`
	MaterialName      string  `json:"material_name"`
	MaterialCode      *string `json:"material_code"`
	Quantity          float64 `json:"quantity"`
	Unit              string  `json:"unit"`
}

type PurchaseProjectRef struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Address *string `json:"address,omitempty"`
}

type PurchaseBuyerRef struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
}

type PurchaseDetail struct {
	ID                      string               `json:"id"`
	Project                 PurchaseProjectRef   `json:"project"`
	Buyer                   PurchaseBuyerRef     `json:"buyer"`
	PurchasedAt             time.Time            `json:"purchased_at"`
	ReceiptFileURL          *string              `json:"receipt_file_url"`
	ReceiptOriginalFilename *string              `json:"receipt_original_filename"`
	Notes                   *string              `json:"notes"`
	Items                   []PurchaseItemDetail `json:"items"`
	CreatedAt               time.Time            `json:"created_at"`
}

// ── Form data ─────────────────────────────────────────────────────────────────

type PurchaseFormProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PurchaseFormMaterial struct {
	ID                string  `json:"id"`
	MaterialName      string  `json:"material_name"`
	MaterialCode      *string `json:"material_code"`
	AvailableQuantity float64 `json:"available_quantity"`
	Unit              string  `json:"unit"`
}

type PurchaseFormDataResponse struct {
	Projects  []PurchaseFormProject  `json:"projects"`
	Materials []PurchaseFormMaterial `json:"materials"`
}

// ── Filter ────────────────────────────────────────────────────────────────────

type PurchaseFilter struct {
	ProjectID *string
	BuyerID   *string
	DateFrom  *string
	DateTo    *string
	Search    *string
}
