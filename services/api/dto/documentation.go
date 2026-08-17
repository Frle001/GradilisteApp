package dto

import "time"

// ── Employee ──────────────────────────────────────────────────────────────────

type DocEmployee struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// ── Document file ─────────────────────────────────────────────────────────────

type DocFile struct {
	ID               string    `json:"id"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	FileSize         int64     `json:"file_size"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

// ── Medical exam ──────────────────────────────────────────────────────────────

type MedicalExam struct {
	ID            string    `json:"id"`
	EmployeeID    string    `json:"employee_id"`
	CompletedDate string    `json:"completed_date"` // YYYY-MM-DD
	ExpiresAt     string    `json:"expires_at"`     // YYYY-MM-DD
	Document      *DocFile  `json:"document,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ── Safety training ───────────────────────────────────────────────────────────

type SafetyRecord struct {
	ID         string     `json:"id"`
	EmployeeID string     `json:"employee_id"`
	Status     string     `json:"status"` // pending / uploaded / verified
	Document   *DocFile   `json:"document,omitempty"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ── Employment contract ───────────────────────────────────────────────────────

type EmploymentContract struct {
	ID                  string    `json:"id"`
	EmployeeID          string    `json:"employee_id"`
	ContractType        string    `json:"contract_type"` // odredeno / neodredeno
	ExpiresAt           *string   `json:"expires_at,omitempty"`
	Document            *DocFile  `json:"document,omitempty"`
	TerminatedAt        *string   `json:"terminated_at,omitempty"`
	TerminationDocument *DocFile  `json:"termination_document,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// ── Monthly obligation ────────────────────────────────────────────────────────

type MonthlyObligation struct {
	ID             string     `json:"id"`
	EmployeeID     string     `json:"employee_id"`
	Year           int        `json:"year"`
	Month          int        `json:"month"`
	ObligationType string     `json:"obligation_type"` // platna_lista / evidencija_radnika
	Status         string     `json:"status"`          // pending / completed
	Hours          *float64   `json:"hours,omitempty"`
	Document       *DocFile   `json:"document,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ── Annual leave ──────────────────────────────────────────────────────────────

type AnnualLeave struct {
	ID         string    `json:"id"`
	EmployeeID string    `json:"employee_id"`
	DateFrom   *string   `json:"date_from,omitempty"`
	DateTo     *string   `json:"date_to,omitempty"`
	Document   *DocFile  `json:"document,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ── Work permit ───────────────────────────────────────────────────────────────

type WorkPermit struct {
	ID         string    `json:"id"`
	EmployeeID string    `json:"employee_id"`
	EnteredAt  string    `json:"entered_at"`
	ExpiresAt  string    `json:"expires_at"`
	Document   *DocFile  `json:"document,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ── Pension record ────────────────────────────────────────────────────────────

type PensionRecord struct {
	ID         string     `json:"id"`
	EmployeeID string     `json:"employee_id"`
	Status     string     `json:"status"` // pending / uploaded / verified
	Document   *DocFile   `json:"document,omitempty"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ── Health record ─────────────────────────────────────────────────────────────

type HealthRecord struct {
	ID         string     `json:"id"`
	EmployeeID string     `json:"employee_id"`
	Status     string     `json:"status"` // pending / uploaded / verified
	Document   *DocFile   `json:"document,omitempty"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ── Notification ──────────────────────────────────────────────────────────────

// DocNotificationType identifies which documentation rule triggered a notification.
const (
	DocNotifMedicalWarning   = "medical_warning"
	DocNotifMedicalUrgent    = "medical_urgent"
	DocNotifMedicalOverdue   = "medical_overdue"
	DocNotifSafetyLow        = "safety_low"
	DocNotifSafetyUrgent     = "safety_urgent"
	DocNotifContractWarning  = "contract_warning"
	DocNotifContractOverdue  = "contract_overdue"
	DocNotifPayrollWarning   = "payroll_warning"
	DocNotifPayrollOverdue   = "payroll_overdue"
	DocNotifTimesheetWarning = "timesheet_warning"
	DocNotifTimesheetOverdue = "timesheet_overdue"
	DocNotifPermitLow        = "permit_low"
	DocNotifPermitWarning    = "permit_warning"
	DocNotifPermitUrgent     = "permit_urgent"
	DocNotifPermitOverdue    = "permit_overdue"
	DocNotifPensionMissing   = "pension_missing"
	DocNotifHealthMissing    = "health_missing"
)

// DocPriority constants.
const (
	PriorityLow     = "low"
	PriorityWarning = "warning"
	PriorityUrgent  = "urgent"
	PriorityOverdue = "overdue"
)

type DocNotification struct {
	Type           string  `json:"type"`
	Priority       string  `json:"priority"`
	EmployeeID     string  `json:"employee_id"`
	EmployeeName   string  `json:"employee_name"`
	RecordID       string  `json:"record_id,omitempty"`
	Message        string  `json:"message"`
	DueDate        *string `json:"due_date,omitempty"`
	DaysRemaining  *int    `json:"days_remaining,omitempty"`
	DaysOverdue    *int    `json:"days_overdue,omitempty"`
	Year           *int    `json:"year,omitempty"`
	Month          *int    `json:"month,omitempty"`
	ObligationType *string `json:"obligation_type,omitempty"`
}

// ── Employee detail (summary) ─────────────────────────────────────────────────

type EmployeeDocSummary struct {
	Employee       DocEmployee         `json:"employee"`
	LatestMedical  *MedicalExam        `json:"latest_medical,omitempty"`
	SafetyRecord   *SafetyRecord       `json:"safety_record,omitempty"`
	ActiveContract *EmploymentContract `json:"active_contract,omitempty"`
	LatestPermit   *WorkPermit         `json:"latest_permit,omitempty"`
	PensionRecord  *PensionRecord      `json:"pension_record,omitempty"`
	HealthRecord   *HealthRecord       `json:"health_record,omitempty"`
}
