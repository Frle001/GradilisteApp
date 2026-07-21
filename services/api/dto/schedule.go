package dto

// ── Request types ─────────────────────────────────────────────────────────────

type CreateShiftRequest struct {
	ProjectID        string   `json:"project_id" binding:"required"`
	ShiftDate        string   `json:"shift_date" binding:"required"` // "YYYY-MM-DD"
	StartTime        string   `json:"start_time" binding:"required"` // "HH:MM"
	EndTime          string   `json:"end_time"   binding:"required"` // "HH:MM"
	Notes            *string  `json:"notes"`
	EmployeeIDs      []string `json:"employee_ids"`      // optional; assigned in the same transaction
	OverrideOverlaps bool     `json:"override_overlaps"` // if true, overlapping employees are still assigned
}

type UpdateShiftRequest struct {
	ShiftDate *string `json:"shift_date"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Notes     *string `json:"notes"`
}

type AssignEmployeesRequest struct {
	EmployeeIDs      []string `json:"employee_ids" binding:"required"`
	OverrideOverlaps bool     `json:"override_overlaps"`
}

type DuplicateShiftRequest struct {
	TargetDate string `json:"target_date" binding:"required"` // "YYYY-MM-DD"
}

type CopyDayRequest struct {
	SourceDate string `json:"source_date" binding:"required"` // "YYYY-MM-DD"
	TargetDate string `json:"target_date" binding:"required"`
}

type CopyWeekRequest struct {
	SourceWeekStart string `json:"source_week_start" binding:"required"` // "YYYY-MM-DD" Monday
	TargetWeekStart string `json:"target_week_start" binding:"required"`
}

// ── Response types ────────────────────────────────────────────────────────────

type ShiftAssignmentItem struct {
	ID                string `json:"id"`
	EmployeeID        string `json:"employee_id"`
	EmployeeName      string `json:"employee_name"`
	OverlapOverridden bool   `json:"overlap_overridden"`
}

type ShiftItem struct {
	ID          string                `json:"id"`
	ProjectID   string                `json:"project_id"`
	ProjectName string                `json:"project_name"`
	ShiftDate   string                `json:"shift_date"`
	StartTime   string                `json:"start_time"`
	EndTime     string                `json:"end_time"`
	Notes       *string               `json:"notes,omitempty"`
	Status      string                `json:"status"`
	CancelledAt *string               `json:"cancelled_at,omitempty"`
	CreatedAt   string                `json:"created_at"`
	Assignments []ShiftAssignmentItem `json:"assignments"`
}

type ShiftOverlapItem struct {
	EmployeeID   string `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	ShiftID      string `json:"shift_id"`
	ShiftDate    string `json:"shift_date"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}

type AssignResponse struct {
	Assignments      []ShiftAssignmentItem `json:"assignments"`
	Overlaps         []ShiftOverlapItem    `json:"overlaps,omitempty"`
	RequiresOverride bool                  `json:"requires_override"`
}

type CopyDayResponse struct {
	Created int         `json:"created"`
	Skipped int         `json:"skipped"`
	Shifts  []ShiftItem `json:"shifts"`
}
