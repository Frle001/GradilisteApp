package dto

import "time"

// ── Project list / detail ─────────────────────────────────────────────────────

type ProjectListItem struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Address              *string    `json:"address"`
	Description          *string    `json:"description"`
	Status               string     `json:"status"`
	StartDate            *string    `json:"start_date"`
	EndDate              *string    `json:"end_date"`
	ClosedAt             *time.Time `json:"closed_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	PrimaryPoslovodaID   *string    `json:"primary_poslovoda_id"`
	PrimaryPoslovodaName *string    `json:"primary_poslovoda_name"`
	AssignmentCount      int        `json:"assignment_count"`
}

type AssignmentItem struct {
	ID            string    `json:"id"`
	EmployeeID    string    `json:"employee_id"`
	EmployeeName  string    `json:"employee_name"`
	EmployeeRole  string    `json:"employee_role"`
	RoleOnProject string    `json:"role_on_project"`
	Active        bool      `json:"active"`
	AssignedAt    time.Time `json:"assigned_at"`
}

type ProjectDetail struct {
	ID                   string           `json:"id"`
	CompanyID            string           `json:"company_id"`
	Name                 string           `json:"name"`
	Address              *string          `json:"address"`
	Description          *string          `json:"description"`
	Status               string           `json:"status"`
	StartDate            *string          `json:"start_date"`
	EndDate              *string          `json:"end_date"`
	ClosedAt             *time.Time       `json:"closed_at"`
	CreatedBy            *string          `json:"created_by"`
	ClosedBy             *string          `json:"closed_by"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
	PrimaryPoslovodaID   *string          `json:"primary_poslovoda_id"`
	PrimaryPoslovodaName *string          `json:"primary_poslovoda_name"`
	MaterialsCount       int              `json:"materials_count"`
	DailyReportsCount    int              `json:"daily_reports_count"`
	Assignments          []AssignmentItem `json:"assignments"`
}

// ── Requests ──────────────────────────────────────────────────────────────────

type CreateProjectRequest struct {
	Name               string  `json:"name" binding:"required"`
	Address            *string `json:"address"`
	Description        *string `json:"description"`
	StartDate          *string `json:"start_date"`
	EndDate            *string `json:"end_date"`
	PrimaryPoslovodaID *string `json:"primary_poslovoda_id"`
}

type UpdateProjectRequest struct {
	Name        string  `json:"name" binding:"required"`
	Address     *string `json:"address"`
	Description *string `json:"description"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
}

type AssignPoslovodaRequest struct {
	PoslovodaID string `json:"poslovoda_id" binding:"required"`
}

// ── Archive list ──────────────────────────────────────────────────────────────

type ArchiveListItem struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	Address                *string    `json:"address"`
	Status                 string     `json:"status"`
	StartDate              *string    `json:"start_date"`
	EndDate                *string    `json:"end_date"`
	ClosedAt               *time.Time `json:"closed_at"`
	ClosedByName           *string    `json:"closed_by_name"`
	PrimaryPoslovodaName   *string    `json:"primary_poslovoda_name"`
	DailyReportsCount      int        `json:"daily_reports_count"`
	MaterialPurchasesCount int        `json:"material_purchases_count"`
	ProjectMaterialsCount  int        `json:"project_materials_count"`
}

type ArchiveListResponse struct {
	Projects   []ArchiveListItem `json:"projects"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PerPage    int               `json:"per_page"`
	TotalPages int               `json:"total_pages"`
}

// ── Archive summary ───────────────────────────────────────────────────────────

type ArchiveSummaryAssignee struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type ArchiveSummary struct {
	ID                          string                   `json:"id"`
	Name                        string                   `json:"name"`
	Address                     *string                  `json:"address"`
	Status                      string                   `json:"status"`
	StartDate                   *string                  `json:"start_date"`
	EndDate                     *string                  `json:"end_date"`
	ClosedAt                    *time.Time               `json:"closed_at"`
	ClosedByName                *string                  `json:"closed_by_name"`
	DailyReportsCount           int                      `json:"daily_reports_count"`
	WorkerHoursTotal            float64                  `json:"worker_hours_total"`
	ActivitiesCount             int                      `json:"activities_count"`
	VtkCount                    int                      `json:"vtk_count"`
	PurchaseSessionsCount       int                      `json:"purchase_sessions_count"`
	PurchaseItemsCount          int                      `json:"purchase_items_count"`
	ProjectMaterialsCount       int                      `json:"project_materials_count"`
	MaterialResponsibilityCount int                      `json:"material_responsibility_count"`
	TransfersCount              int                      `json:"transfers_count"`
	AssignedPoslovode           []ArchiveSummaryAssignee `json:"assigned_poslovode"`
}
