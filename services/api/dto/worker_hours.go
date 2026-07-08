package dto

import "time"

// ── Request ───────────────────────────────────────────────────────────────────

type SubmitWorkerHoursRequest struct {
	ProjectID   string  `json:"project_id" binding:"required"`
	WorkDate    string  `json:"work_date" binding:"required"` // YYYY-MM-DD
	HoursWorked float64 `json:"hours_worked" binding:"required"`
	Notes       *string `json:"notes"`
}

// ── Response ──────────────────────────────────────────────────────────────────

type WorkerHoursEntry struct {
	ID          string    `json:"id"`
	WorkerID    string    `json:"worker_id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	WorkDate    string    `json:"work_date"`
	HoursWorked float64   `json:"hours_worked"`
	Notes       *string   `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WorkerProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
