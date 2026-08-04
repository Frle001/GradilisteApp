package dto

import "time"

// ── Requests ──────────────────────────────────────────────────────────────────

type SubmitWorkerHoursRequest struct {
	ProjectID          string  `json:"project_id" binding:"required"`
	WorkDate           string  `json:"work_date" binding:"required"` // YYYY-MM-DD
	HoursWorked        float64 `json:"hours_worked" binding:"required"`
	Notes              *string `json:"notes"`
	WorkDescription    *string `json:"work_description"`
	ClientSubmissionID *string `json:"client_submission_id"`
}

type CorrectWorkerHoursRequest struct {
	NewHours float64 `json:"new_hours" binding:"required"`
	Reason   string  `json:"reason"    binding:"required"`
}

type AddWorkerHoursCommentRequest struct {
	CommentText string `json:"comment_text" binding:"required"`
}

// ── Responses ─────────────────────────────────────────────────────────────────

type WorkerHoursEntry struct {
	ID              string    `json:"id"`
	WorkerID        string    `json:"worker_id"`
	ProjectID       string    `json:"project_id"`
	ProjectName     string    `json:"project_name"`
	WorkDate        string    `json:"work_date"`
	HoursWorked     float64   `json:"hours_worked"`
	Notes           *string   `json:"notes"`
	WorkDescription *string   `json:"work_description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WorkerHoursRevision struct {
	ID            string    `json:"id"`
	PreviousHours float64   `json:"previous_hours"`
	NewHours      float64   `json:"new_hours"`
	Reason        string    `json:"reason"`
	ChangedByName *string   `json:"changed_by_name"`
	CreatedAt     time.Time `json:"created_at"`
}

type WorkerHoursComment struct {
	ID          string    `json:"id"`
	AuthorName  *string   `json:"author_name"`
	CommentText string    `json:"comment_text"`
	CreatedAt   time.Time `json:"created_at"`
}

// WorkerHoursDetail is returned to direktor/inženjer for a single entry.
type WorkerHoursDetail struct {
	WorkerHoursEntry
	WorkerName string                `json:"worker_name"`
	WorkerRole string                `json:"worker_role"`
	Revisions  []WorkerHoursRevision `json:"revisions"`
	Comments   []WorkerHoursComment  `json:"comments"`
}

// ManagerWorkerHoursEntry is returned in the manager overview list.
type ManagerWorkerHoursEntry struct {
	ID              string    `json:"id"`
	WorkerID        string    `json:"worker_id"`
	WorkerName      string    `json:"worker_name"`
	WorkerRole      string    `json:"worker_role"`
	ProjectID       string    `json:"project_id"`
	ProjectName     string    `json:"project_name"`
	WorkDate        string    `json:"work_date"`
	HoursWorked     float64   `json:"hours_worked"`
	Notes           *string   `json:"notes"`
	WorkDescription *string   `json:"work_description,omitempty"`
	HasRevisions    bool      `json:"has_revisions"`
	HasComments     bool      `json:"has_comments"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WorkerProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
