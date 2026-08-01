package dto

// ── Shared filter ─────────────────────────────────────────────────────────────

type ReportFilter struct {
	ProjectID    *string
	PoslovodaID  *string
	WorkerID     *string
	MaterialID   *string // knjiga: project_material_id
	ActivityType *string // knjiga
	IsVTK        *bool   // knjiga
	DateFrom     *string
	DateTo       *string
	Status       *string
	Search       *string
}

// ── Pagination ────────────────────────────────────────────────────────────────

type ReportPagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ── Građevinski dnevnik ───────────────────────────────────────────────────────

type GradevinskiDnevnikRow struct {
	ReportID        string  `json:"report_id"`
	ReportDate      string  `json:"report_date"`
	ProjectID       string  `json:"project_id"`
	ProjectName     string  `json:"project_name"`
	ProjectAddress  *string `json:"project_address"`
	PoslovodaID     string  `json:"poslovoda_id"`
	PoslovodaName   string  `json:"poslovoda_name"`
	WorkerID        string  `json:"worker_id"`
	WorkerName      string  `json:"worker_name"`
	HoursWorked     float64 `json:"hours_worked"`
	Notes           *string `json:"notes"`
	WorkDescription *string `json:"work_description"`
	WorkerRole      string  `json:"worker_role"`
	WorkerHoursID   *string `json:"worker_hours_id"` // non-null only for worker_daily_hours rows
	HasRevisions    bool    `json:"has_revisions"`
	HasComments     bool    `json:"has_comments"`
	Status          string  `json:"status"`
	Source          string  `json:"source"`
}

type GradevinskiDnevnikSummary struct {
	TotalHours    float64 `json:"total_hours"`
	WorkersCount  int     `json:"workers_count"`
	ProjectsCount int     `json:"projects_count"`
}

type GradevinskiDnevnikResponse struct {
	Data       []GradevinskiDnevnikRow   `json:"data"`
	Pagination ReportPagination          `json:"pagination"`
	Summary    GradevinskiDnevnikSummary `json:"summary"`
}

// ── Građevinska knjiga ────────────────────────────────────────────────────────

type GradevinskaKnjigaRow struct {
	ReportID           string  `json:"report_id"`
	ReportDate         string  `json:"report_date"`
	ProjectID          string  `json:"project_id"`
	ProjectName        string  `json:"project_name"`
	ProjectAddress     *string `json:"project_address"`
	PoslovodaID        string  `json:"poslovoda_id"`
	PoslovodaName      string  `json:"poslovoda_name"`
	ActivityID         string  `json:"activity_id"`
	MaterialID         *string `json:"material_id"` // project_material_id
	MaterialName       string  `json:"material_name"`
	MaterialCode       *string `json:"material_code"`
	CustomMaterialName *string `json:"custom_material_name"`
	Quantity           float64 `json:"quantity"`
	Unit               string  `json:"unit"`
	ActivityType       string  `json:"activity_type"`
	IsVTK              bool    `json:"is_vtk"`
	Notes              *string `json:"notes"`
	Status             string  `json:"status"`
}

type GradevinskaKnjigaSummary struct {
	TotalQuantity   float64 `json:"total_quantity"`
	ActivitiesCount int     `json:"activities_count"`
	VTKCount        int     `json:"vtk_count"`
	ProjectsCount   int     `json:"projects_count"`
}

type GradevinskaKnjigaResponse struct {
	Data       []GradevinskaKnjigaRow   `json:"data"`
	Pagination ReportPagination         `json:"pagination"`
	Summary    GradevinskaKnjigaSummary `json:"summary"`
}

// ── Filter options ────────────────────────────────────────────────────────────

type FilterOptionProject struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type FilterOptionPerson struct {
	ID           string  `json:"id"`
	FullName     string  `json:"full_name"`
	SupervisorID *string `json:"supervisor_id,omitempty"`
}

type FilterOptionMaterial struct {
	ID           string `json:"id"`
	MaterialName string `json:"material_name"`
	ProjectID    string `json:"project_id"`
}

type ReportFilterOptions struct {
	Projects      []FilterOptionProject  `json:"projects"`
	Poslovode     []FilterOptionPerson   `json:"poslovode"`
	Workers       []FilterOptionPerson   `json:"workers"`
	Materials     []FilterOptionMaterial `json:"materials"`
	ActivityTypes []string               `json:"activity_types"`
	Statuses      []string               `json:"statuses"`
}
