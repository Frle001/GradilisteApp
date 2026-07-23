package dto

// ── Request types ─────────────────────────────────────────────────────────────

type CreateShiftRequest struct {
	ProjectID        string   `json:"project_id" binding:"required"`
	ShiftDate        string   `json:"shift_date" binding:"required"` // "YYYY-MM-DD"
	StartTime        *string  `json:"start_time"`                    // optional; stored but not enforced
	EndTime          *string  `json:"end_time"`                      // optional; stored but not enforced
	Notes            *string  `json:"notes"`
	EmployeeIDs      []string `json:"employee_ids"`      // optional; assigned in the same transaction
	OverrideOverlaps bool     `json:"override_overlaps"` // ignored; kept for API compatibility
}

type UpdateShiftRequest struct {
	ShiftDate *string `json:"shift_date"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Notes     *string `json:"notes"`
}

type AssignEmployeesRequest struct {
	EmployeeIDs      []string `json:"employee_ids" binding:"required"`
	OverrideOverlaps bool     `json:"override_overlaps"` // ignored; kept for API compatibility
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
	StartTime   *string               `json:"start_time,omitempty"`
	EndTime     *string               `json:"end_time,omitempty"`
	Notes       *string               `json:"notes,omitempty"`
	Status      string                `json:"status"`
	CancelledAt *string               `json:"cancelled_at,omitempty"`
	CreatedAt   string                `json:"created_at"`
	Assignments []ShiftAssignmentItem `json:"assignments"`
}

// ShiftConflictItem describes an employee already assigned to another shift on
// the same date when a duplicate-daily-assignment conflict is detected.
type ShiftConflictItem struct {
	EmployeeID   string `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	ShiftID      string `json:"shift_id"`
	ProjectName  string `json:"project_name"`
}

// ShiftOverlapItem is kept for backward compatibility with the SyncAssignments response.
type ShiftOverlapItem struct {
	EmployeeID   string `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	ShiftID      string `json:"shift_id"`
}

type AssignResponse struct {
	Assignments      []ShiftAssignmentItem `json:"assignments"`
	RequiresOverride bool                  `json:"requires_override"`
}

type CopyDayResponse struct {
	Created int         `json:"created"`
	Skipped int         `json:"skipped"`
	Shifts  []ShiftItem `json:"shifts"`
}

// EmployeeForDateItem is returned by GET /schedule/employees-for-date.
// Assigned is true when the employee already has an active shift on that date
// (excluding the shift identified by ExcludeShiftID, if any).
type EmployeeForDateItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Role        string  `json:"role"`
	Assigned    bool    `json:"assigned"`
	ShiftID     *string `json:"shift_id,omitempty"`
	ProjectName *string `json:"project_name,omitempty"`
}
