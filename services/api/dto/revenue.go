package dto

import "time"

// ── Project contracts ─────────────────────────────────────────────────────────

type ProjectContract struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	ProjectName  string    `json:"project_name"`
	ContractType string    `json:"contract_type"` // "initial" | "amendment"
	Reference    *string   `json:"reference,omitempty"`
	ContractDate *string   `json:"contract_date,omitempty"` // YYYY-MM-DD
	NetValue     float64   `json:"net_value"`
	Notes        *string   `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateContractReq struct {
	ProjectID    string  `json:"project_id"`
	ContractType string  `json:"contract_type"` // "initial" | "amendment"
	Reference    *string `json:"reference"`
	ContractDate *string `json:"contract_date"` // YYYY-MM-DD or null
	NetValue     float64 `json:"net_value"`
	Notes        *string `json:"notes"`
}

type UpdateContractReq struct {
	ContractType string  `json:"contract_type"`
	Reference    *string `json:"reference"`
	ContractDate *string `json:"contract_date"`
	NetValue     float64 `json:"net_value"`
	Notes        *string `json:"notes"`
}

// ── Project invoices ──────────────────────────────────────────────────────────

type ProjectInvoice struct {
	ID                string    `json:"id"`
	ProjectID         string    `json:"project_id"`
	ProjectName       string    `json:"project_name"`
	InvoiceNumber     string    `json:"invoice_number"`
	InvoiceDate       string    `json:"invoice_date"` // YYYY-MM-DD
	DueDate           *string   `json:"due_date,omitempty"`
	NetAmount         float64   `json:"net_amount"`
	VatAmount         float64   `json:"vat_amount"`
	GrossAmount       float64   `json:"gross_amount"`
	PaidAmount        float64   `json:"paid_amount"`
	OutstandingAmount float64   `json:"outstanding_amount"` // gross_amount - paid_amount
	Status            string    `json:"status"`
	IsOverdue         bool      `json:"is_overdue"`
	CancelsInvoiceID  *string   `json:"cancels_invoice_id,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type CreateProjectInvoiceReq struct {
	ProjectID     string  `json:"project_id"`
	InvoiceNumber string  `json:"invoice_number"`
	InvoiceDate   string  `json:"invoice_date"`
	DueDate       *string `json:"due_date"`
	NetAmount     float64 `json:"net_amount"`
	VatAmount     float64 `json:"vat_amount"`
	Notes         *string `json:"notes"`
}

type CancelProjectInvoiceReq struct {
	Notes *string `json:"notes"`
}

// ── Invoice payments ──────────────────────────────────────────────────────────

type InvoicePayment struct {
	ID          string    `json:"id"`
	InvoiceID   string    `json:"invoice_id"`
	PaymentDate string    `json:"payment_date"` // YYYY-MM-DD
	Amount      float64   `json:"amount"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type AddPaymentReq struct {
	PaymentDate string  `json:"payment_date"`
	Amount      float64 `json:"amount"`
	Notes       *string `json:"notes"`
}

// ── Revenue analytics ─────────────────────────────────────────────────────────

type ProjectRevenueSummary struct {
	Year               int                  `json:"year"`
	Month              int                  `json:"month"` // 0 = full year
	TotalContracted    float64              `json:"total_contracted"`
	TotalInvoicedNet   float64              `json:"total_invoiced_net"`
	TotalInvoicedGross float64              `json:"total_invoiced_gross"`
	TotalCollected     float64              `json:"total_collected"`
	TotalOutstanding   float64              `json:"total_outstanding"` // invoiced_gross - collected
	TotalOverdue       float64              `json:"total_overdue"`
	OverdueCount       int                  `json:"overdue_count"`
	MonthlyTrend       []MonthlyRevenueTrend `json:"monthly_trend"`
	Projects           []ProjectRevenueRow  `json:"projects"`
}

type ProjectRevenueRow struct {
	ProjectID     string  `json:"project_id"`
	ProjectName   string  `json:"project_name"`
	ContractValue float64 `json:"contract_value"`
	InvoicedNet   float64 `json:"invoiced_net"`
	InvoicedGross float64 `json:"invoiced_gross"`
	Collected     float64 `json:"collected"`
	Outstanding   float64 `json:"outstanding"`
	InvoiceCount  int     `json:"invoice_count"`
	HasOverdue    bool    `json:"has_overdue"`
}

type MonthlyRevenueTrend struct {
	Month         int     `json:"month"`
	InvoicedNet   float64 `json:"invoiced_net"`
	InvoicedGross float64 `json:"invoiced_gross"`
	Collected     float64 `json:"collected"`
	HasData       bool    `json:"has_data"`
}

type ProjectRevenueDetail struct {
	ProjectID     string               `json:"project_id"`
	ProjectName   string               `json:"project_name"`
	Year          int                  `json:"year"`
	ContractValue float64              `json:"contract_value"`
	InvoicedNet   float64              `json:"invoiced_net"`
	InvoicedGross float64              `json:"invoiced_gross"`
	Collected     float64              `json:"collected"`
	Outstanding   float64              `json:"outstanding"`
	Contracts     []ProjectContract    `json:"contracts"`
	Invoices      []ProjectInvoice     `json:"invoices"`
	MonthlyTrend  []MonthlyRevenueTrend `json:"monthly_trend"`
}
