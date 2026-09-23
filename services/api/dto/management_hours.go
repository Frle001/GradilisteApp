package dto

import "time"

// ── Requests ──────────────────────────────────────────────────────────────────

// SubmitManagementHoursRequest is the body for POST /management-hours.
// project_id is optional — management employees may log hours without a project.
type SubmitManagementHoursRequest struct {
	WorkDate    string  `json:"work_date" binding:"required"` // YYYY-MM-DD
	HoursWorked float64 `json:"hours_worked" binding:"required"`
	Notes       *string `json:"notes"`
	ProjectID   *string `json:"project_id"` // optional
}

// UpdateManagementHoursRequest is the body for PUT /management-hours/:id.
type UpdateManagementHoursRequest struct {
	HoursWorked float64 `json:"hours_worked" binding:"required"`
	Notes       *string `json:"notes"`
}

// ── Responses ─────────────────────────────────────────────────────────────────

// ManagementHoursEntry is returned for any management-hours operation.
type ManagementHoursEntry struct {
	ID           string    `json:"id"`
	EmployeeID   string    `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	EmployeeRole string    `json:"employee_role"`
	ProjectID    *string   `json:"project_id"`
	ProjectName  *string   `json:"project_name"`
	WorkDate     string    `json:"work_date"`
	HoursWorked  float64   `json:"hours_worked"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
