package dto

import "time"

// ── Asset types ───────────────────────────────────────────────────────────────

const (
	AssetTypeAlat   = "alat"
	AssetTypeOprema = "oprema"
	AssetTypeVozilo = "vozilo"
)

var ValidAssetTypes = map[string]bool{
	AssetTypeAlat:   true,
	AssetTypeOprema: true,
	AssetTypeVozilo: true,
}

var ValidAssetSizes = map[string]bool{
	"XS": true, "S": true, "M": true, "L": true,
	"XL": true, "XXL": true, "3XL": true, "Univerzalna": true,
}

// ── Request DTOs ──────────────────────────────────────────────────────────────

type CreateCompanyAssetRequest struct {
	AssetType          string  `json:"asset_type"             binding:"required"`
	Name               string  `json:"name"                   binding:"required"`
	AssignedEmployeeID *string `json:"assigned_employee_id"`
	Notes              *string `json:"notes"`
	// alat + oprema
	PurchasedAt       *string `json:"purchased_at"`
	WarrantyExpiresAt *string `json:"warranty_expires_at"`
	// oprema only
	Size *string `json:"size"`
	// vozilo only
	RegistrationPlate     *string `json:"registration_plate"`
	RegistrationDate      *string `json:"registration_date"`
	RegistrationExpiresAt *string `json:"registration_expires_at"`
	// vozilo leasing
	IsLeasing      bool    `json:"is_leasing"`
	LeasingCompany *string `json:"leasing_company"`
	LeasingEndDate *string `json:"leasing_end_date"`
}

type UpdateCompanyAssetRequest struct {
	Name                  string  `json:"name"                   binding:"required"`
	AssignedEmployeeID    *string `json:"assigned_employee_id"`
	Status                string  `json:"status"`
	Notes                 *string `json:"notes"`
	PurchasedAt           *string `json:"purchased_at"`
	WarrantyExpiresAt     *string `json:"warranty_expires_at"`
	Size                  *string `json:"size"`
	RegistrationPlate     *string `json:"registration_plate"`
	RegistrationDate      *string `json:"registration_date"`
	RegistrationExpiresAt *string `json:"registration_expires_at"`
	// vozilo leasing
	IsLeasing      bool    `json:"is_leasing"`
	LeasingCompany *string `json:"leasing_company"`
	LeasingEndDate *string `json:"leasing_end_date"`
}

// ── Leasing payment ───────────────────────────────────────────────────────────

type LeasingPaymentRequest struct {
	PeriodMonth string `json:"period_month" binding:"required"`
}

type LeasingPayment struct {
	ID          string    `json:"id"`
	AssetID     string    `json:"asset_id"`
	PeriodMonth string    `json:"period_month"`
	CompletedAt time.Time `json:"completed_at"`
	CompletedBy string    `json:"completed_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type AssetEmployee struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

type CompanyAsset struct {
	ID                    string         `json:"id"`
	CompanyID             string         `json:"company_id"`
	AssetType             string         `json:"asset_type"`
	Name                  string         `json:"name"`
	Status                string         `json:"status"`
	Notes                 *string        `json:"notes"`
	AssignedEmployee      *AssetEmployee `json:"assigned_employee"`
	PurchasedAt           *string        `json:"purchased_at"`
	WarrantyExpiresAt     *string        `json:"warranty_expires_at"`
	Size                  *string        `json:"size"`
	RegistrationPlate     *string        `json:"registration_plate"`
	RegistrationDate      *string        `json:"registration_date"`
	RegistrationExpiresAt *string        `json:"registration_expires_at"`
	// Leasing (vozilo only)
	IsLeasing      bool      `json:"is_leasing"`
	LeasingCompany *string   `json:"leasing_company"`
	LeasingEndDate *string   `json:"leasing_end_date"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ── Notifications ─────────────────────────────────────────────────────────────

type AssetNotificationKind = string
type AssetNotificationState = string

const (
	NotificationKindWarranty       AssetNotificationKind = "warranty"
	NotificationKindRegistration   AssetNotificationKind = "registration"
	NotificationKindLeasingWarning AssetNotificationKind = "leasing_warning"
	NotificationKindLeasingOverdue AssetNotificationKind = "leasing_overdue"

	NotificationStateWarning AssetNotificationState = "warning" // 1–10 days (warranty/reg), days 1-15 (leasing)
	NotificationStateUrgent  AssetNotificationState = "urgent"  // today (warranty/reg), days 16+ (leasing)
	NotificationStateExpired AssetNotificationState = "expired" // past expiry (warranty/reg)
)

type AssetNotification struct {
	AssetID          string         `json:"asset_id"`
	AssetName        string         `json:"asset_name"`
	AssetType        string         `json:"asset_type"`
	Kind             string         `json:"kind"`
	State            string         `json:"state"`
	DaysRemaining    int            `json:"days_remaining"` // negative = overdue
	ExpiresAt        string         `json:"expires_at"`     // "YYYY-MM-DD" expiry or due date
	AssignedEmployee *AssetEmployee `json:"assigned_employee"`
	Message          string         `json:"message"`
	// Leasing-specific (empty for non-leasing notifications)
	RegistrationPlate *string `json:"registration_plate,omitempty"`
	PeriodMonth       string  `json:"period_month,omitempty"` // "YYYY-MM-01" for leasing
}

type NotificationsResponse struct {
	Notifications []AssetNotification `json:"notifications"`
	Count         int                 `json:"count"`
}

// ── Form data ─────────────────────────────────────────────────────────────────

type AssetFormDataResponse struct {
	Employees []AssetEmployee `json:"employees"`
}

// ── Filter ────────────────────────────────────────────────────────────────────

type CompanyAssetFilter struct {
	AssetType  string
	EmployeeID string
	Status     string
	Search     string
}
