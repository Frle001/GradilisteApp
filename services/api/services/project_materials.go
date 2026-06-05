package services

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

var ErrImportJobConflict = errors.New("import job already confirmed or not in required state")
var ErrProjectArchived = errors.New("cannot import materials into an archived project")

type ProjectMaterialService struct {
	db         *pgxpool.Pool
	matRepo    *repositories.ProjectMaterialRepository
	importRepo *repositories.ImportJobRepository
	auditRepo  *repositories.AuditRepository
	projRepo   *repositories.ProjectRepository
}

func NewProjectMaterialService(
	db *pgxpool.Pool,
	matRepo *repositories.ProjectMaterialRepository,
	importRepo *repositories.ImportJobRepository,
	auditRepo *repositories.AuditRepository,
	projRepo *repositories.ProjectRepository,
) *ProjectMaterialService {
	return &ProjectMaterialService{
		db:         db,
		matRepo:    matRepo,
		importRepo: importRepo,
		auditRepo:  auditRepo,
		projRepo:   projRepo,
	}
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

func (s *ProjectMaterialService) List(ctx context.Context, projectID, companyID string, search string, activeOnly bool) ([]dto.MaterialListItem, error) {
	items, err := s.matRepo.List(ctx, projectID, companyID, repositories.MaterialFilter{
		Search:     search,
		ActiveOnly: activeOnly,
	})
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []dto.MaterialListItem{}
	}
	return items, nil
}

func (s *ProjectMaterialService) Create(ctx context.Context, projectID, companyID, callerUserID string, req dto.CreateMaterialRequest) (*dto.MaterialListItem, error) {
	status, err := s.projRepo.GetStatus(ctx, companyID, projectID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, repositories.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != "active" {
		return nil, ErrProjectNotActive
	}

	m, err := s.matRepo.Create(ctx, projectID, companyID, req)
	if err != nil {
		return nil, err
	}
	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project_material.create",
		EntityType: "project_material",
		EntityID:   &m.ID,
		NewData:    m,
	})
	return m, nil
}

func (s *ProjectMaterialService) Update(ctx context.Context, id, projectID, companyID, callerUserID string, req dto.UpdateMaterialRequest) (*dto.MaterialListItem, error) {
	status, err := s.projRepo.GetStatus(ctx, companyID, projectID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, repositories.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != "active" {
		return nil, ErrProjectNotActive
	}

	m, err := s.matRepo.Update(ctx, id, projectID, companyID, req)
	if err != nil {
		return nil, err
	}
	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project_material.update",
		EntityType: "project_material",
		EntityID:   &m.ID,
		NewData:    m,
	})
	return m, nil
}

func (s *ProjectMaterialService) Deactivate(ctx context.Context, id, projectID, companyID, callerUserID string) error {
	status, err := s.projRepo.GetStatus(ctx, companyID, projectID)
	if errors.Is(err, repositories.ErrNotFound) {
		return repositories.ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != "active" {
		return ErrProjectNotActive
	}

	if err := s.matRepo.Deactivate(ctx, id, projectID, companyID); err != nil {
		return err
	}
	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project_material.deactivate",
		EntityType: "project_material",
		EntityID:   &id,
	})
	return nil
}

// ── Step 1: Analyze ───────────────────────────────────────────────────────────

// Analyze uploads an Excel file, reads headers + raw rows, persists them, and
// returns header suggestions + sample rows so the frontend can present the mapping UI.
func (s *ProjectMaterialService) Analyze(ctx context.Context, projectID, companyID, callerUserID, filename string, data []byte) (*dto.ImportAnalyzeResponse, error) {
	proj, err := s.projRepo.GetByID(ctx, companyID, projectID)
	if err != nil {
		return nil, err
	}
	if proj.Status != "active" {
		return nil, ErrProjectNotActive
	}

	result, err := AnalyzeExcel(data)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	jobID, err := s.importRepo.CreateJob(ctx, tx, companyID, projectID, filename, callerUserID)
	if err != nil {
		return nil, err
	}
	if err := s.importRepo.StoreAnalyzeRows(ctx, tx, companyID, jobID, result.RawRows, result.DataRowOffset); err != nil {
		return nil, err
	}
	if err := s.importRepo.SetJobTotalRows(ctx, tx, jobID, len(result.RawRows)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &dto.ImportAnalyzeResponse{
		ImportJobID:     jobID,
		SheetName:       result.SheetName,
		HeaderRowNumber: result.HeaderRowNumber,
		Headers:         result.Headers,
		SampleRows:      result.SampleRows,
	}, nil
}

// ── Step 2: Wizard preview ────────────────────────────────────────────────────

// WizardPreview applies the user-selected mapping to the stored raw rows, validates
// each row, persists the results, and returns a preview the user can review.
func (s *ProjectMaterialService) WizardPreview(ctx context.Context, jobID, projectID, companyID string, mapping map[string]string) (*dto.WizardPreviewResponse, error) {
	job, err := s.importRepo.GetJob(ctx, jobID, projectID, companyID)
	if err != nil {
		return nil, err
	}
	// Allow re-previewing (status 'uploaded' or 'parsed') but not confirmed.
	if job.Status == "confirmed" {
		return nil, ErrImportJobConflict
	}

	rawRows, err := s.importRepo.GetAllRawRows(ctx, jobID, companyID)
	if err != nil {
		return nil, err
	}

	// Convert to rawMappingRow to preserve actual Excel row numbers from the DB.
	inputs := make([]rawMappingRow, len(rawRows))
	for i, r := range rawRows {
		inputs[i] = rawMappingRow{RowNumber: r.RowNumber, Cells: r.Cells}
	}

	wizardRows, parseSummary := parseRowsWithMapping(inputs, mapping)
	if wizardRows == nil {
		wizardRows = make([]dto.WizardPreviewRow, 0)
	}

	valid, invalid := 0, 0
	for _, r := range wizardRows {
		if r.Status == "valid" {
			valid++
		} else {
			invalid++
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Delete old rows (from analyze step or a previous preview call) and re-insert.
	if err := s.importRepo.DeleteAllRows(ctx, tx, jobID, companyID); err != nil {
		return nil, err
	}
	if err := s.importRepo.StoreWizardRows(ctx, tx, companyID, jobID, wizardRows); err != nil {
		return nil, err
	}
	if err := s.importRepo.UpdateJobCounts(ctx, tx, jobID, len(wizardRows), valid, invalid); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &dto.WizardPreviewResponse{
		Rows: wizardRows,
		Summary: dto.WizardPreviewSummary{
			TotalRows:            len(wizardRows),
			ValidRows:            valid,
			InvalidRows:          invalid,
			TotalScanned:         parseSummary.TotalScanned,
			SkippedCategories:    parseSummary.SkippedCategories,
			SkippedContinuations: parseSummary.SkippedContinuations,
			SkippedSubtotals:     parseSummary.SkippedSubtotals,
		},
	}, nil
}

// ── Step 3: Confirm ───────────────────────────────────────────────────────────

// ConfirmImport imports director-reviewed rows from a previewed import job into project_materials.
// rows is the list of (possibly edited) rows from the frontend. If rows is empty, falls back
// to the stored preview rows from the database.
func (s *ProjectMaterialService) ConfirmImport(ctx context.Context, jobID, projectID, companyID, callerUserID string, rows []dto.WizardConfirmRow) (*dto.WizardConfirmResponse, error) {
	job, err := s.importRepo.GetJob(ctx, jobID, projectID, companyID)
	if err != nil {
		return nil, err
	}
	if job.Status != "parsed" {
		return nil, ErrImportJobConflict
	}

	usingStoredRows := len(rows) == 0

	// Validate included rows in memory before opening a transaction.
	var rowErrors []dto.WizardConfirmRowError
	var validRows []dto.WizardConfirmRow
	skipped, failed := 0, 0

	if !usingStoredRows {
		for _, row := range rows {
			if !row.Include {
				skipped++
				continue
			}
			// Auto-calculate total_price when absent but quantity and unit_price are known.
			if row.TotalPrice == nil && row.UnitPrice != nil && row.Quantity > 0 {
				tp := row.Quantity * *row.UnitPrice
				row.TotalPrice = &tp
			}
			if errs := validateConfirmRow(row); len(errs) > 0 {
				failed++
				rowErrors = append(rowErrors, dto.WizardConfirmRowError{RowNumber: row.RowNumber, Errors: errs})
				continue
			}
			validRows = append(validRows, row)
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Fall back to stored preview rows when the frontend sends none.
	if usingStoredRows {
		stored, err := s.importRepo.GetValidWizardRows(ctx, tx, jobID, companyID)
		if err != nil {
			return nil, err
		}
		for _, r := range stored {
			validRows = append(validRows, dto.WizardConfirmRow{
				RowNumber:    r.RowNumber,
				MaterialName: r.MaterialName,
				Quantity:     r.Quantity,
				Unit:         r.Unit,
				UnitPrice:    r.UnitPrice,
				TotalPrice:   r.TotalPrice,
				Category:     r.Category,
				Note:         r.Note,
				Include:      true,
			})
		}
		skipped = job.TotalRows - job.ValidRows
	}

	imported := 0
	for _, row := range validRows {
		if err := s.matRepo.UpsertConfirmRowWithTx(ctx, tx, projectID, companyID, row); err != nil {
			return nil, err
		}
		imported++
	}

	if err := s.importRepo.MarkRowsImported(ctx, tx, jobID, companyID); err != nil {
		return nil, err
	}
	if err := s.importRepo.MarkJobConfirmed(ctx, tx, jobID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project_material.import_confirm",
		EntityType: "import_job",
		EntityID:   &jobID,
		NewData:    map[string]int{"imported": imported, "skipped": skipped, "failed": failed},
	})

	return &dto.WizardConfirmResponse{
		ImportedCount: imported,
		SkippedCount:  skipped,
		FailedCount:   failed,
		Errors:        nonNilRowErrors(rowErrors),
	}, nil
}

func validateConfirmRow(row dto.WizardConfirmRow) []string {
	var errs []string
	if strings.TrimSpace(row.MaterialName) == "" {
		errs = append(errs, "Naziv materijala je obavezan")
	}
	if row.Quantity <= 0 {
		errs = append(errs, "Količina mora biti > 0")
	}
	if strings.TrimSpace(row.Unit) == "" {
		errs = append(errs, "Jedinica mjere je obavezna")
	}
	if row.UnitPrice != nil && *row.UnitPrice < 0 {
		errs = append(errs, "Jedinična cijena ne može biti negativna")
	}
	if row.TotalPrice != nil && *row.TotalPrice < 0 {
		errs = append(errs, "Ukupna cijena ne može biti negativna")
	}
	return errs
}

func nonNilRowErrors(s []dto.WizardConfirmRowError) []dto.WizardConfirmRowError {
	if s == nil {
		return []dto.WizardConfirmRowError{}
	}
	return s
}
