package dto

import "time"

// ── Request types ─────────────────────────────────────────────────────────────

type WorkerHoursInput struct {
	WorkerID    string  `json:"worker_id" binding:"required"`
	HoursWorked float64 `json:"hours_worked"`
	Notes       *string `json:"notes"`
}

type ActivityInput struct {
	ProjectMaterialID  *string `json:"project_material_id"`
	CustomMaterialName *string `json:"custom_material_name"`
	Quantity           float64 `json:"quantity"`
	Unit               string  `json:"unit"`
	ActivityType       string  `json:"activity_type"`
	IsVTK              bool    `json:"is_vtk"`
	Notes              *string `json:"notes"`
}

type CreateDailyReportRequest struct {
	ProjectID   string             `json:"project_id" binding:"required"`
	ReportDate  string             `json:"report_date" binding:"required"`
	Notes       *string            `json:"notes"`
	WorkerHours []WorkerHoursInput `json:"worker_hours"`
	Activities  []ActivityInput    `json:"activities"`
}

type RejectDailyReportRequest struct {
	Reason string `json:"reason"`
}

// ── List response ─────────────────────────────────────────────────────────────

type DailyReportListItem struct {
	ID               string     `json:"id"`
	ProjectID        string     `json:"project_id"`
	ProjectName      string     `json:"project_name"`
	PoslovodaID      string     `json:"poslovoda_id"`
	PoslovodaName    string     `json:"poslovoda_name"`
	ReportDate       string     `json:"report_date"`
	Status           string     `json:"status"`
	WorkerHoursCount int        `json:"worker_hours_count"`
	TotalHours       float64    `json:"total_hours"`
	ActivitiesCount  int        `json:"activities_count"`
	SubmittedAt      *time.Time `json:"submitted_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ── Detail response ───────────────────────────────────────────────────────────

type DailyReportProject struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DailyReportPerson struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
}

type DailyReportWorkerHours struct {
	ID          string  `json:"id"`
	WorkerID    string  `json:"worker_id"`
	WorkerName  string  `json:"worker_name"`
	HoursWorked float64 `json:"hours_worked"`
	Notes       *string `json:"notes"`
}

type DailyReportActivity struct {
	ID                 string  `json:"id"`
	ProjectMaterialID  *string `json:"project_material_id"`
	MaterialName       string  `json:"material_name"`
	CustomMaterialName *string `json:"custom_material_name"`
	Quantity           float64 `json:"quantity"`
	Unit               string  `json:"unit"`
	ActivityType       string  `json:"activity_type"`
	IsVTK              bool    `json:"is_vtk"`
	Notes              *string `json:"notes"`
}

type DailyReportAttachment struct {
	ID           string    `json:"id"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	FileSize     int64     `json:"file_size"`
	CreatedAt    time.Time `json:"created_at"`
}

type DailyReportDetail struct {
	ID          string                   `json:"id"`
	Project     DailyReportProject       `json:"project"`
	Poslovoda   DailyReportPerson        `json:"poslovoda"`
	ReportDate  string                   `json:"report_date"`
	Status      string                   `json:"status"`
	Notes       *string                  `json:"notes"`
	SubmittedAt *time.Time               `json:"submitted_at"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
	WorkerHours []DailyReportWorkerHours `json:"worker_hours"`
	Activities  []DailyReportActivity    `json:"activities"`
	Attachments []DailyReportAttachment  `json:"attachments"`
}

// ── Form-data response ────────────────────────────────────────────────────────

type FormDataProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FormDataWorker struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
}

type FormDataMaterial struct {
	ID                string  `json:"id"`
	MaterialName      string  `json:"material_name"`
	MaterialCode      *string `json:"material_code"`
	AvailableQuantity float64 `json:"available_quantity"`
	Unit              string  `json:"unit"`
}

type DailyReportFormData struct {
	Projects  []FormDataProject  `json:"projects"`
	Workers   []FormDataWorker   `json:"workers"`
	Materials []FormDataMaterial `json:"materials"`
}

// ── Filter ────────────────────────────────────────────────────────────────────

type DailyReportFilter struct {
	ProjectID   *string
	PoslovodaID *string // scopes to a specific poslovoda (used for poslovoda role)
	DateFrom    *string
	DateTo      *string
	Status      *string
}
