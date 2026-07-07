package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

var ErrDailyReportForbidden = errors.New("access denied to this daily report")
var ErrDailyReportNotEditable = errors.New("report cannot be edited in its current status")
var ErrDailyReportInvalidStatus = errors.New("report is not in a state that allows this action")

type DailyReportService struct {
	db        *pgxpool.Pool
	drRepo    *repositories.DailyReportRepository
	auditRepo *repositories.AuditRepository
}

func NewDailyReportService(
	db *pgxpool.Pool,
	drRepo *repositories.DailyReportRepository,
	auditRepo *repositories.AuditRepository,
) *DailyReportService {
	return &DailyReportService{db: db, drRepo: drRepo, auditRepo: auditRepo}
}

// ── List ──────────────────────────────────────────────────────────────────────

func (s *DailyReportService) List(ctx context.Context, companyID, callerRole, callerEmpID string, f dto.DailyReportFilter) ([]dto.DailyReportListItem, error) {
	// Poslovoda sees only their own reports
	if callerRole == "poslovoda" {
		f.PoslovodaID = &callerEmpID
	}
	// Administracija: read-only, no extra scope filter (sees all in company)
	return s.drRepo.List(ctx, companyID, f)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (s *DailyReportService) GetByID(ctx context.Context, id, companyID, callerRole, callerEmpID string) (*dto.DailyReportDetail, error) {
	d, err := s.drRepo.GetByID(ctx, id, companyID)
	if err != nil {
		return nil, err
	}
	// Poslovoda can only see their own reports
	if callerRole == "poslovoda" && d.Poslovoda.ID != callerEmpID {
		return nil, ErrDailyReportForbidden
	}
	return d, nil
}

// ── Create ────────────────────────────────────────────────────────────────────

func (s *DailyReportService) Create(ctx context.Context, companyID, callerUserID, callerEmpID, callerRole string, req dto.CreateDailyReportRequest) (string, error) {
	poslovodaEmpID, err := s.resolvePoslovodaID(callerRole, callerEmpID)
	if err != nil {
		return "", err
	}

	loc, _ := time.LoadLocation("Europe/Zagreb")
	today := time.Now().In(loc).Format("2006-01-02")
	if req.ReportDate != today {
		return "", fmt.Errorf("Dnevni izvještaj se može kreirati samo za današnji datum (%s)", today)
	}

	if err := s.validateReport(ctx, companyID, req.ProjectID, poslovodaEmpID, callerRole, req); err != nil {
		return "", err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	reportID, err := s.drRepo.Create(ctx, tx, companyID, req.ProjectID, poslovodaEmpID, callerUserID, req)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		EmployeeID: nilIfEmpty(callerEmpID),
		Action:     "daily_report.create",
		EntityType: "daily_report",
		EntityID:   &reportID,
		NewData:    map[string]string{"project_id": req.ProjectID, "report_date": req.ReportDate},
	})

	return reportID, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (s *DailyReportService) Update(ctx context.Context, id, companyID, callerUserID, callerEmpID, callerRole string, req dto.CreateDailyReportRequest) (*dto.DailyReportDetail, error) {
	status, poslovodaID, err := s.drRepo.GetByIDForEdit(ctx, id, companyID)
	if err != nil {
		return nil, err
	}

	if err := s.checkEditPermission(callerRole, callerEmpID, poslovodaID, status); err != nil {
		return nil, err
	}

	poslovodaEmpID, err := s.resolvePoslovodaID(callerRole, callerEmpID)
	if err != nil {
		return nil, err
	}
	// For update, the project_id comes from the stored report; the request may change worker_hours/activities.
	// We re-validate using the same project_id.
	if err := s.validateReport(ctx, companyID, req.ProjectID, poslovodaEmpID, callerRole, req); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.drRepo.Update(ctx, tx, id, companyID, req); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		EmployeeID: nilIfEmpty(callerEmpID),
		Action:     "daily_report.update",
		EntityType: "daily_report",
		EntityID:   &id,
	})

	return s.drRepo.GetByID(ctx, id, companyID)
}

// ── Approve ───────────────────────────────────────────────────────────────────

func (s *DailyReportService) Approve(ctx context.Context, id, companyID, callerUserID, callerEmpID string) error {
	status, _, err := s.drRepo.GetByIDForEdit(ctx, id, companyID)
	if err != nil {
		return err
	}
	if status != "submitted" && status != "draft" {
		return ErrDailyReportInvalidStatus
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.drRepo.SetStatus(ctx, tx, id, companyID, "approved", nil); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		EmployeeID: nilIfEmpty(callerEmpID),
		Action:     "daily_report.approve",
		EntityType: "daily_report",
		EntityID:   &id,
	})
	return nil
}

// ── Reject ────────────────────────────────────────────────────────────────────

func (s *DailyReportService) Reject(ctx context.Context, id, companyID, callerUserID, callerEmpID, reason string) error {
	status, _, err := s.drRepo.GetByIDForEdit(ctx, id, companyID)
	if err != nil {
		return err
	}
	if status != "submitted" && status != "draft" {
		return ErrDailyReportInvalidStatus
	}

	var notesVal *string
	if strings.TrimSpace(reason) != "" {
		notesVal = &reason
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.drRepo.SetStatus(ctx, tx, id, companyID, "rejected", notesVal); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		EmployeeID: nilIfEmpty(callerEmpID),
		Action:     "daily_report.reject",
		EntityType: "daily_report",
		EntityID:   &id,
		NewData:    map[string]string{"reason": reason},
	})
	return nil
}

// ── Form data ─────────────────────────────────────────────────────────────────

func (s *DailyReportService) GetFormData(ctx context.Context, companyID, callerEmpID, callerRole, projectID string) (*dto.DailyReportFormData, error) {
	var (
		projects  []dto.FormDataProject
		workers   []dto.FormDataWorker
		materials []dto.FormDataMaterial
		err       error
	)

	switch callerRole {
	case "poslovoda":
		if callerEmpID == "" {
			return nil, errors.New("poslovoda must have a linked employee record")
		}
		projects, err = s.drRepo.GetProjectsForPoslovoda(ctx, callerEmpID, companyID)
		if err != nil {
			return nil, err
		}
		workers, err = s.drRepo.GetWorkersForPoslovoda(ctx, callerEmpID, companyID)
		if err != nil {
			return nil, err
		}
	default:
		projects, err = s.drRepo.GetActiveProjects(ctx, companyID)
		if err != nil {
			return nil, err
		}
		workers, err = s.drRepo.GetAllActiveWorkers(ctx, companyID)
		if err != nil {
			return nil, err
		}
	}

	materials = []dto.FormDataMaterial{}
	if projectID != "" {
		materials, err = s.drRepo.GetMaterialsForProject(ctx, projectID, companyID)
		if err != nil {
			return nil, err
		}
	}

	return &dto.DailyReportFormData{
		Projects:  projects,
		Workers:   workers,
		Materials: materials,
	}, nil
}

// ── Validation ────────────────────────────────────────────────────────────────

func (s *DailyReportService) validateReport(ctx context.Context, companyID, projectID, poslovodaEmpID, callerRole string, req dto.CreateDailyReportRequest) error {
	// Validate date format
	if _, err := time.Parse("2006-01-02", req.ReportDate); err != nil {
		return fmt.Errorf("report_date must be in YYYY-MM-DD format")
	}

	// Check project
	projStatus, err := s.drRepo.GetProjectStatus(ctx, projectID, companyID)
	if err != nil {
		return err
	}
	if projStatus == "" {
		return fmt.Errorf("projekt nije pronađen")
	}
	if projStatus != "active" {
		return fmt.Errorf("dnevni izvještaj se može unijeti samo za aktivne projekte")
	}

	// Poslovoda must be assigned to the project
	if callerRole == "poslovoda" {
		assigned, err := s.drRepo.IsProjectAssignedToPoslovoda(ctx, projectID, poslovodaEmpID, companyID)
		if err != nil {
			return err
		}
		if !assigned {
			return fmt.Errorf("poslovođa nije dodijeljen ovom projektu")
		}
	}

	// At least one entry required
	if len(req.WorkerHours) == 0 && len(req.Activities) == 0 {
		return fmt.Errorf("izvještaj mora sadržavati barem jednog radnika ili jednu aktivnost")
	}

	// Validate worker hours
	for _, wh := range req.WorkerHours {
		role, supervisorID, err := s.drRepo.GetWorkerInfo(ctx, wh.WorkerID, companyID)
		if err != nil {
			return err
		}
		if role == "" {
			return fmt.Errorf("radnik %s nije pronađen u ovoj tvrtki", wh.WorkerID)
		}
		if role != "radnik" {
			return fmt.Errorf("samo radnici mogu biti uneseni u sate rada")
		}
		if callerRole == "poslovoda" {
			if supervisorID == nil || *supervisorID != poslovodaEmpID {
				return fmt.Errorf("radnik %s nije pod nadzorom ove poslovođe", wh.WorkerID)
			}
		}
		if wh.HoursWorked < 0 || wh.HoursWorked > 24 {
			return fmt.Errorf("sati rada moraju biti između 0 i 24")
		}
	}

	// Validate activities
	validActivityTypes := map[string]bool{"montaza": true, "demontaza": true, "other": true}
	for i, a := range req.Activities {
		if a.Quantity <= 0 {
			return fmt.Errorf("aktivnost %d: količina mora biti pozitivna", i+1)
		}
		if strings.TrimSpace(a.Unit) == "" {
			return fmt.Errorf("aktivnost %d: jedinica mjere je obavezna", i+1)
		}
		if !validActivityTypes[a.ActivityType] {
			return fmt.Errorf("aktivnost %d: tip aktivnosti mora biti montaza, demontaza ili other", i+1)
		}
		if a.IsVTK {
			if a.CustomMaterialName == nil || strings.TrimSpace(*a.CustomMaterialName) == "" {
				return fmt.Errorf("aktivnost %d (VTK): naziv materijala je obavezan", i+1)
			}
			if a.ProjectMaterialID != nil {
				return fmt.Errorf("aktivnost %d (VTK): project_material_id mora biti null", i+1)
			}
		} else {
			if a.ProjectMaterialID == nil || strings.TrimSpace(*a.ProjectMaterialID) == "" {
				return fmt.Errorf("aktivnost %d: project_material_id je obavezan za standardne aktivnosti", i+1)
			}
			ok, err := s.drRepo.IsMaterialInProject(ctx, *a.ProjectMaterialID, projectID, companyID)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aktivnost %d: materijal ne pripada odabranom projektu", i+1)
			}
		}
	}

	return nil
}

func (s *DailyReportService) resolvePoslovodaID(callerRole, callerEmpID string) (string, error) {
	if callerRole == "poslovoda" {
		if callerEmpID == "" {
			return "", fmt.Errorf("poslovođa mora imati povezan zapis zaposlenika")
		}
		return callerEmpID, nil
	}
	// Direktor/inženjer creating a report use their own employee ID as poslovoda.
	// In practice directors don't often create reports, but the endpoint allows it.
	if callerEmpID == "" {
		return "", fmt.Errorf("korisnik mora imati povezan zapis zaposlenika za izradu izvještaja")
	}
	return callerEmpID, nil
}

func (s *DailyReportService) checkEditPermission(callerRole, callerEmpID, poslovodaID, status string) error {
	if status == "approved" {
		return ErrDailyReportNotEditable
	}
	if callerRole == "poslovoda" && callerEmpID != poslovodaID {
		return ErrDailyReportForbidden
	}
	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
