package dto

import "time"

// ── Invoice types ─────────────────────────────────────────────────────────────

const (
	InvoiceTypeMaterijal = "materijal"
	InvoiceTypeGorivo    = "gorivo"
	InvoiceTypeLeasing   = "leasing"
	InvoiceTypeAlati     = "alati"
	InvoiceTypeOprema    = "oprema"
)

var AllowedInvoiceTypes = map[string]bool{
	InvoiceTypeMaterijal: true,
	InvoiceTypeGorivo:    true,
	InvoiceTypeLeasing:   true,
	InvoiceTypeAlati:     true,
	InvoiceTypeOprema:    true,
}

// ── Suppliers (for materijal) ─────────────────────────────────────────────────

const (
	SupplierEnergyCentar = "Energy Centar"
	SupplierPondeljak    = "Pondeljak"
	SupplierLipaPromet   = "Lipa Promet"
)

var AllowedSuppliers = map[string]bool{
	SupplierEnergyCentar: true,
	SupplierPondeljak:    true,
	SupplierLipaPromet:   true,
}

// ── Leasing companies (for leasing) ──────────────────────────────────────────

const (
	LeasingImpuls           = "Impuls"
	LeasingUnicreditLeasing = "Unicredit Leasing"
)

var AllowedLeasingCompanies = map[string]bool{
	LeasingImpuls:           true,
	LeasingUnicreditLeasing: true,
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type FinanceDocMeta struct {
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	FileSize         int64     `json:"file_size"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

type CompanyInvoice struct {
	ID             string         `json:"id"`
	InvoiceType    string         `json:"invoice_type"`
	Supplier       *string        `json:"supplier,omitempty"`
	LeasingCompany *string        `json:"leasing_company,omitempty"`
	Document       FinanceDocMeta `json:"document"`
	CreatedBy      string         `json:"created_by"`
	CreatedByName  string         `json:"created_by_name"`
	CreatedAt      time.Time      `json:"created_at"`
}

type R1Receipt struct {
	ID            string         `json:"id"`
	SubmittedBy   string         `json:"submitted_by"`
	SubmitterName string         `json:"submitter_name"`
	Price         float64        `json:"price"`
	Document      FinanceDocMeta `json:"document"`
	CreatedAt     time.Time      `json:"created_at"`
}
