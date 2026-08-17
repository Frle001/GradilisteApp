package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Repository interface ──────────────────────────────────────────────────────

type documentationRepoIface interface {
	ListActiveEmployees(ctx context.Context, companyID string) ([]repositories.DocEmployeeRow, error)
	ListActiveRadnikEmployees(ctx context.Context, companyID string) ([]repositories.DocEmployeeRow, error)
	GetEmployee(ctx context.Context, companyID, employeeID string) (*repositories.DocEmployeeRow, error)

	CreateDocFile(ctx context.Context, companyID, employeeID, storageKey, originalFilename, mimeType, uploadedBy string, fileSize int64) (string, error)
	GetDocFile(ctx context.Context, companyID, fileID string) (*repositories.DocumentFileRow, error)

	CreateMedicalExam(ctx context.Context, companyID, employeeID, createdBy, documentID string, completedDate, expiresAt time.Time) (string, error)
	GetLatestMedicalExam(ctx context.Context, companyID, employeeID string) (*repositories.MedicalExamRow, error)
	ListMedicalExams(ctx context.Context, companyID, employeeID string) ([]repositories.MedicalExamRow, error)
	ListLatestMedicalExams(ctx context.Context, companyID string) (map[string]*repositories.MedicalExamRow, error)

	GetSafetyRecord(ctx context.Context, companyID, employeeID string) (*repositories.SafetyRecordRow, error)
	UpsertSafetyUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*repositories.SafetyRecordRow, error)
	VerifySafetyRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*repositories.SafetyRecordRow, error)
	ListSafetyRecords(ctx context.Context, companyID string) (map[string]*repositories.SafetyRecordRow, error)

	CreateContract(ctx context.Context, companyID, employeeID, contractType, createdBy, documentID string, expiresAt *time.Time) (string, error)
	GetContract(ctx context.Context, companyID, contractID string) (*repositories.ContractRow, error)
	TerminateContract(ctx context.Context, companyID, contractID, terminatedBy, terminationDocID string, terminatedAt time.Time) error
	ListContracts(ctx context.Context, companyID, employeeID string) ([]repositories.ContractRow, error)
	GetLatestActiveContract(ctx context.Context, companyID, employeeID string) (*repositories.ContractRow, error)
	ListLatestActiveContracts(ctx context.Context, companyID string) (map[string]*repositories.ContractRow, error)

	GetMonthlyObligation(ctx context.Context, companyID, employeeID string, year, month int, oblType string) (*repositories.MonthlyObligationRow, error)
	UpsertMonthlyCompletion(ctx context.Context, companyID, employeeID, oblType, completedBy string, year, month int, hours *float64, documentID *string) (*repositories.MonthlyObligationRow, error)
	ListMonthlyObligations(ctx context.Context, companyID, employeeID string) ([]repositories.MonthlyObligationRow, error)
	ListCompletedObligations(ctx context.Context, companyID string, fromYear, fromMonth int) (map[string]map[repositories.MonthlyOblKey]bool, error)
	SumMonthlyHoursForWorker(ctx context.Context, companyID, employeeID string, year, month int) (float64, error)

	CreateAnnualLeave(ctx context.Context, companyID, employeeID, createdBy string, dateFrom, dateTo *time.Time, documentID *string) (string, error)
	ListAnnualLeave(ctx context.Context, companyID, employeeID string) ([]repositories.AnnualLeaveRow, error)

	CreateWorkPermit(ctx context.Context, companyID, employeeID, createdBy, documentID string, enteredAt, expiresAt time.Time) (string, error)
	GetLatestWorkPermit(ctx context.Context, companyID, employeeID string) (*repositories.WorkPermitRow, error)
	ListWorkPermits(ctx context.Context, companyID, employeeID string) ([]repositories.WorkPermitRow, error)
	ListLatestWorkPermits(ctx context.Context, companyID string) (map[string]*repositories.WorkPermitRow, error)

	GetPensionRecord(ctx context.Context, companyID, employeeID string) (*repositories.PensionRecordRow, error)
	UpsertPensionUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*repositories.PensionRecordRow, error)
	VerifyPensionRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*repositories.PensionRecordRow, error)
	ListPensionRecords(ctx context.Context, companyID string) (map[string]*repositories.PensionRecordRow, error)

	GetHealthRecord(ctx context.Context, companyID, employeeID string) (*repositories.HealthRecordRow, error)
	UpsertHealthUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*repositories.HealthRecordRow, error)
	VerifyHealthRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*repositories.HealthRecordRow, error)
	ListHealthRecords(ctx context.Context, companyID string) (map[string]*repositories.HealthRecordRow, error)
}

// ── Storage interface ─────────────────────────────────────────────────────────

type docStorageIface interface {
	SaveEmployeeDocFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, employeeID string) (fileKey, contentType string, fileSize int64, err error)
}

// ── Service ───────────────────────────────────────────────────────────────────

type DocumentationService struct {
	repo    documentationRepoIface
	storage docStorageIface
}

func NewDocumentationService(repo documentationRepoIface, storage docStorageIface) *DocumentationService {
	return &DocumentationService{repo: repo, storage: storage}
}

func (s *DocumentationService) requireAdmin(role string) error {
	if role != "administracija" {
		return ErrForbidden
	}
	return nil
}

func (s *DocumentationService) validateEmployee(ctx context.Context, companyID, employeeID string) (*repositories.DocEmployeeRow, error) {
	emp, err := s.repo.GetEmployee(ctx, companyID, employeeID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return emp, nil
}

// ── Employees ─────────────────────────────────────────────────────────────────

func (s *DocumentationService) ListEmployees(ctx context.Context, companyID, callerRole string) ([]dto.DocEmployee, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListActiveEmployees(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListEmployees: %w", err)
	}
	out := make([]dto.DocEmployee, len(rows))
	for i, r := range rows {
		out[i] = toDocEmployeeDTO(r)
	}
	return out, nil
}

// ── File upload (shared) ──────────────────────────────────────────────────────

func (s *DocumentationService) UploadDocFile(ctx context.Context, companyID, callerRole, callerUserID, employeeID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return "", err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return "", err
	}

	key, mime, size, err := s.storage.SaveEmployeeDocFile(ctx, file, header, companyID, employeeID)
	if err != nil {
		return "", fmt.Errorf("doc upload: save file: %w", err)
	}

	fileID, err := s.repo.CreateDocFile(ctx, companyID, employeeID, key, header.Filename, mime, callerUserID, size)
	if err != nil {
		return "", fmt.Errorf("doc upload: create record: %w", err)
	}
	return fileID, nil
}

// ── Medical exams ─────────────────────────────────────────────────────────────

type CreateMedicalExamRequest struct {
	EmployeeID    string
	CompletedDate string // YYYY-MM-DD
	ExpiresAt     string // YYYY-MM-DD
	DocumentID    string
}

func (s *DocumentationService) CreateMedicalExam(ctx context.Context, companyID, callerRole, callerUserID string, req CreateMedicalExamRequest) (*dto.MedicalExam, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, req.EmployeeID); err != nil {
		return nil, err
	}

	completed, err := time.Parse("2006-01-02", req.CompletedDate)
	if err != nil {
		return nil, validationErr("Neispravan format datuma pregleda (YYYY-MM-DD)")
	}
	expires, err := time.Parse("2006-01-02", req.ExpiresAt)
	if err != nil {
		return nil, validationErr("Neispravan format datuma isteka (YYYY-MM-DD)")
	}
	if expires.Before(completed) {
		return nil, validationErr("Datum isteka ne smije biti prije datuma pregleda")
	}

	if req.DocumentID == "" {
		return nil, validationErr("Dokument je obavezan za liječnički pregled")
	}
	if _, err := s.repo.GetDocFile(ctx, companyID, req.DocumentID); err != nil {
		return nil, validationErr("Dokument nije pronađen")
	}

	id, err := s.repo.CreateMedicalExam(ctx, companyID, req.EmployeeID, callerUserID, req.DocumentID, completed, expires)
	if err != nil {
		return nil, fmt.Errorf("doc.CreateMedicalExam: %w", err)
	}

	row, err := s.repo.GetLatestMedicalExam(ctx, companyID, req.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.CreateMedicalExam: refetch: %w", err)
	}
	_ = id
	return toMedicalExamDTO(*row), nil
}

func (s *DocumentationService) ListMedicalExams(ctx context.Context, companyID, callerRole, employeeID string) ([]dto.MedicalExam, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListMedicalExams(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListMedicalExams: %w", err)
	}
	out := make([]dto.MedicalExam, len(rows))
	for i, r := range rows {
		out[i] = *toMedicalExamDTO(r)
	}
	return out, nil
}

// ── Safety records ────────────────────────────────────────────────────────────

func (s *DocumentationService) UploadSafetyRecord(ctx context.Context, companyID, callerRole, callerUserID, employeeID, documentID string) (*dto.SafetyRecord, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	emp, err := s.validateEmployee(ctx, companyID, employeeID)
	if err != nil {
		return nil, err
	}
	if emp.Role != "radnik" {
		return nil, validationErr("Zaštita na radu se evidentira samo za radnike")
	}
	if _, err := s.repo.GetDocFile(ctx, companyID, documentID); err != nil {
		return nil, validationErr("Dokument nije pronađen")
	}
	row, err := s.repo.UpsertSafetyUpload(ctx, companyID, employeeID, documentID, callerUserID)
	if err != nil {
		return nil, fmt.Errorf("doc.UploadSafetyRecord: %w", err)
	}
	return toSafetyRecordDTO(*row), nil
}

func (s *DocumentationService) VerifySafetyRecord(ctx context.Context, companyID, callerRole, callerUserID, employeeID string) (*dto.SafetyRecord, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	row, err := s.repo.VerifySafetyRecord(ctx, companyID, employeeID, callerUserID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, validationErr("Zaštita na radu nije u statusu 'uploaded'")
	}
	if err != nil {
		return nil, err
	}
	return toSafetyRecordDTO(*row), nil
}

func (s *DocumentationService) GetSafetyRecord(ctx context.Context, companyID, callerRole, employeeID string) (*dto.SafetyRecord, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	row, err := s.repo.GetSafetyRecord(ctx, companyID, employeeID)
	if err != nil {
		return nil, err
	}
	return toSafetyRecordDTO(*row), nil
}

// ── Contracts ─────────────────────────────────────────────────────────────────

type CreateContractRequest struct {
	EmployeeID   string
	ContractType string  // odredeno / neodredeno
	ExpiresAt    *string // required for odredeno
	DocumentID   string
}

func (s *DocumentationService) CreateContract(ctx context.Context, companyID, callerRole, callerUserID string, req CreateContractRequest) (*dto.EmploymentContract, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, req.EmployeeID); err != nil {
		return nil, err
	}
	if req.ContractType != "odredeno" && req.ContractType != "neodredeno" {
		return nil, validationErr("Tip ugovora mora biti 'odredeno' ili 'neodredeno'")
	}
	if req.DocumentID == "" {
		return nil, validationErr("Dokument je obavezan za ugovor o radu")
	}
	if _, err := s.repo.GetDocFile(ctx, companyID, req.DocumentID); err != nil {
		return nil, validationErr("Dokument nije pronađen")
	}

	var expiresAt *time.Time
	if req.ContractType == "odredeno" {
		if req.ExpiresAt == nil || *req.ExpiresAt == "" {
			return nil, validationErr("Datum isteka je obavezan za ugovor na određeno")
		}
		t, err := time.Parse("2006-01-02", *req.ExpiresAt)
		if err != nil {
			return nil, validationErr("Neispravan format datuma isteka (YYYY-MM-DD)")
		}
		expiresAt = &t
	}

	id, err := s.repo.CreateContract(ctx, companyID, req.EmployeeID, req.ContractType, callerUserID, req.DocumentID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("doc.CreateContract: %w", err)
	}
	row, err := s.repo.GetContract(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("doc.CreateContract: refetch: %w", err)
	}
	return toContractDTO(*row), nil
}

type TerminateContractRequest struct {
	ContractID       string
	TerminatedAt     string // YYYY-MM-DD
	TerminationDocID string
}

func (s *DocumentationService) TerminateContract(ctx context.Context, companyID, callerRole, callerUserID string, req TerminateContractRequest) error {
	if err := s.requireAdmin(callerRole); err != nil {
		return err
	}
	existing, err := s.repo.GetContract(ctx, companyID, req.ContractID)
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if existing.TerminatedAt != nil {
		return validationErr("Ugovor je već raskinut")
	}
	if _, err := s.repo.GetDocFile(ctx, companyID, req.TerminationDocID); err != nil {
		return validationErr("Dokument o raskidu nije pronađen")
	}
	terminatedAt, err := time.Parse("2006-01-02", req.TerminatedAt)
	if err != nil {
		return validationErr("Neispravan format datuma raskida (YYYY-MM-DD)")
	}
	return s.repo.TerminateContract(ctx, companyID, req.ContractID, callerUserID, req.TerminationDocID, terminatedAt)
}

func (s *DocumentationService) ListContracts(ctx context.Context, companyID, callerRole, employeeID string) ([]dto.EmploymentContract, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListContracts(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListContracts: %w", err)
	}
	out := make([]dto.EmploymentContract, len(rows))
	for i, r := range rows {
		out[i] = *toContractDTO(r)
	}
	return out, nil
}

// ── Monthly obligations ───────────────────────────────────────────────────────

type CompletePayrollRequest struct {
	EmployeeID string
	Year       int
	Month      int
	DocumentID *string
}

func (s *DocumentationService) CompletePayroll(ctx context.Context, companyID, callerRole, callerUserID string, req CompletePayrollRequest) (*dto.MonthlyObligation, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, req.EmployeeID); err != nil {
		return nil, err
	}
	if req.Year < 2020 || req.Month < 1 || req.Month > 12 {
		return nil, validationErr("Neispravan period (godina/mjesec)")
	}
	if req.DocumentID != nil && *req.DocumentID != "" {
		if _, err := s.repo.GetDocFile(ctx, companyID, *req.DocumentID); err != nil {
			return nil, validationErr("Dokument nije pronađen")
		}
	}
	row, err := s.repo.UpsertMonthlyCompletion(ctx, companyID, req.EmployeeID, "platna_lista", callerUserID, req.Year, req.Month, nil, req.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("doc.CompletePayroll: %w", err)
	}
	return toMonthlyObligationDTO(*row), nil
}

type CompleteTimesheetRequest struct {
	EmployeeID string
	Year       int
	Month      int
}

func (s *DocumentationService) CompleteTimesheet(ctx context.Context, companyID, callerRole, callerUserID string, req CompleteTimesheetRequest) (*dto.MonthlyObligation, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	emp, err := s.validateEmployee(ctx, companyID, req.EmployeeID)
	if err != nil {
		return nil, err
	}
	if emp.Role != "radnik" {
		return nil, validationErr("Evidencija radnika se vodi samo za radnike")
	}
	if req.Year < 2020 || req.Month < 1 || req.Month > 12 {
		return nil, validationErr("Neispravan period (godina/mjesec)")
	}

	hours, err := s.repo.SumMonthlyHoursForWorker(ctx, companyID, req.EmployeeID, req.Year, req.Month)
	if err != nil {
		return nil, fmt.Errorf("doc.CompleteTimesheet: sum hours: %w", err)
	}

	row, err := s.repo.UpsertMonthlyCompletion(ctx, companyID, req.EmployeeID, "evidencija_radnika", callerUserID, req.Year, req.Month, &hours, nil)
	if err != nil {
		return nil, fmt.Errorf("doc.CompleteTimesheet: %w", err)
	}
	return toMonthlyObligationDTO(*row), nil
}

func (s *DocumentationService) ListMonthlyObligations(ctx context.Context, companyID, callerRole, employeeID string) ([]dto.MonthlyObligation, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListMonthlyObligations(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListMonthlyObligations: %w", err)
	}
	out := make([]dto.MonthlyObligation, len(rows))
	for i, r := range rows {
		out[i] = *toMonthlyObligationDTO(r)
	}
	return out, nil
}

// ── Annual leave ──────────────────────────────────────────────────────────────

type CreateAnnualLeaveRequest struct {
	EmployeeID string
	DateFrom   *string
	DateTo     *string
	DocumentID *string
}

func (s *DocumentationService) CreateAnnualLeave(ctx context.Context, companyID, callerRole, callerUserID string, req CreateAnnualLeaveRequest) (*dto.AnnualLeave, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, req.EmployeeID); err != nil {
		return nil, err
	}
	if req.DocumentID == nil || *req.DocumentID == "" {
		return nil, validationErr("Dokument je obavezan za godišnji odmor")
	}
	if _, err := s.repo.GetDocFile(ctx, companyID, *req.DocumentID); err != nil {
		return nil, validationErr("Dokument nije pronađen")
	}

	var dateFrom, dateTo *time.Time
	if req.DateFrom != nil && *req.DateFrom != "" {
		t, err := time.Parse("2006-01-02", *req.DateFrom)
		if err != nil {
			return nil, validationErr("Neispravan format datuma početka (YYYY-MM-DD)")
		}
		dateFrom = &t
	}
	if req.DateTo != nil && *req.DateTo != "" {
		t, err := time.Parse("2006-01-02", *req.DateTo)
		if err != nil {
			return nil, validationErr("Neispravan format datuma kraja (YYYY-MM-DD)")
		}
		dateTo = &t
	}
	if dateFrom != nil && dateTo != nil && dateTo.Before(*dateFrom) {
		return nil, validationErr("Datum kraja ne smije biti prije datuma početka")
	}

	id, err := s.repo.CreateAnnualLeave(ctx, companyID, req.EmployeeID, callerUserID, dateFrom, dateTo, req.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("doc.CreateAnnualLeave: %w", err)
	}

	leaves, err := s.repo.ListAnnualLeave(ctx, companyID, req.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.CreateAnnualLeave: refetch: %w", err)
	}
	for _, l := range leaves {
		if l.ID == id {
			return toAnnualLeaveDTO(l), nil
		}
	}
	return nil, fmt.Errorf("doc.CreateAnnualLeave: not found after insert")
}

func (s *DocumentationService) ListAnnualLeave(ctx context.Context, companyID, callerRole, employeeID string) ([]dto.AnnualLeave, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListAnnualLeave(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListAnnualLeave: %w", err)
	}
	out := make([]dto.AnnualLeave, len(rows))
	for i, r := range rows {
		out[i] = *toAnnualLeaveDTO(r)
	}
	return out, nil
}

// ── Work permits ──────────────────────────────────────────────────────────────

type CreateWorkPermitRequest struct {
	EmployeeID string
	EnteredAt  string // YYYY-MM-DD
	ExpiresAt  string // YYYY-MM-DD
	DocumentID string
}

func (s *DocumentationService) CreateWorkPermit(ctx context.Context, companyID, callerRole, callerUserID string, req CreateWorkPermitRequest) (*dto.WorkPermit, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, req.EmployeeID); err != nil {
		return nil, err
	}
	if req.DocumentID == "" {
		return nil, validationErr("Dokument je obavezan za radnu dozvolu")
	}
	if _, err := s.repo.GetDocFile(ctx, companyID, req.DocumentID); err != nil {
		return nil, validationErr("Dokument nije pronađen")
	}
	entered, err := time.Parse("2006-01-02", req.EnteredAt)
	if err != nil {
		return nil, validationErr("Neispravan format datuma ulaska (YYYY-MM-DD)")
	}
	expires, err := time.Parse("2006-01-02", req.ExpiresAt)
	if err != nil {
		return nil, validationErr("Neispravan format datuma isteka (YYYY-MM-DD)")
	}
	if expires.Before(entered) {
		return nil, validationErr("Datum isteka ne smije biti prije datuma ulaska")
	}

	id, err := s.repo.CreateWorkPermit(ctx, companyID, req.EmployeeID, callerUserID, req.DocumentID, entered, expires)
	if err != nil {
		return nil, fmt.Errorf("doc.CreateWorkPermit: %w", err)
	}
	row, err := s.repo.GetLatestWorkPermit(ctx, companyID, req.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.CreateWorkPermit: refetch: %w", err)
	}
	_ = id
	return toWorkPermitDTO(*row), nil
}

func (s *DocumentationService) ListWorkPermits(ctx context.Context, companyID, callerRole, employeeID string) ([]dto.WorkPermit, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListWorkPermits(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListWorkPermits: %w", err)
	}
	out := make([]dto.WorkPermit, len(rows))
	for i, r := range rows {
		out[i] = *toWorkPermitDTO(r)
	}
	return out, nil
}

// ── Pension ───────────────────────────────────────────────────────────────────

func (s *DocumentationService) UploadPensionRecord(ctx context.Context, companyID, callerRole, callerUserID, employeeID, documentID string) (*dto.PensionRecord, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	emp, err := s.validateEmployee(ctx, companyID, employeeID)
	if err != nil {
		return nil, err
	}
	if emp.Role != "radnik" {
		return nil, validationErr("Mirovinsko osiguranje se evidentira samo za radnike")
	}
	if _, err := s.repo.GetDocFile(ctx, companyID, documentID); err != nil {
		return nil, validationErr("Dokument nije pronađen")
	}
	row, err := s.repo.UpsertPensionUpload(ctx, companyID, employeeID, documentID, callerUserID)
	if err != nil {
		return nil, fmt.Errorf("doc.UploadPensionRecord: %w", err)
	}
	return toPensionRecordDTO(*row), nil
}

func (s *DocumentationService) VerifyPensionRecord(ctx context.Context, companyID, callerRole, callerUserID, employeeID string) (*dto.PensionRecord, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	row, err := s.repo.VerifyPensionRecord(ctx, companyID, employeeID, callerUserID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, validationErr("Mirovinsko osiguranje nije u statusu 'uploaded'")
	}
	if err != nil {
		return nil, err
	}
	return toPensionRecordDTO(*row), nil
}

// ── Health ────────────────────────────────────────────────────────────────────

func (s *DocumentationService) UploadHealthRecord(ctx context.Context, companyID, callerRole, callerUserID, employeeID, documentID string) (*dto.HealthRecord, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	emp, err := s.validateEmployee(ctx, companyID, employeeID)
	if err != nil {
		return nil, err
	}
	if emp.Role != "radnik" {
		return nil, validationErr("Zdravstveno osiguranje se evidentira samo za radnike")
	}
	if _, err := s.repo.GetDocFile(ctx, companyID, documentID); err != nil {
		return nil, validationErr("Dokument nije pronađen")
	}
	row, err := s.repo.UpsertHealthUpload(ctx, companyID, employeeID, documentID, callerUserID)
	if err != nil {
		return nil, fmt.Errorf("doc.UploadHealthRecord: %w", err)
	}
	return toHealthRecordDTO(*row), nil
}

func (s *DocumentationService) VerifyHealthRecord(ctx context.Context, companyID, callerRole, callerUserID, employeeID string) (*dto.HealthRecord, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.validateEmployee(ctx, companyID, employeeID); err != nil {
		return nil, err
	}
	row, err := s.repo.VerifyHealthRecord(ctx, companyID, employeeID, callerUserID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, validationErr("Zdravstveno osiguranje nije u statusu 'uploaded'")
	}
	if err != nil {
		return nil, err
	}
	return toHealthRecordDTO(*row), nil
}

// ── Employee summary ──────────────────────────────────────────────────────────

func (s *DocumentationService) GetEmployeeSummary(ctx context.Context, companyID, callerRole, employeeID string) (*dto.EmployeeDocSummary, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	emp, err := s.validateEmployee(ctx, companyID, employeeID)
	if err != nil {
		return nil, err
	}

	summary := &dto.EmployeeDocSummary{Employee: toDocEmployeeDTO(*emp)}

	if m, err := s.repo.GetLatestMedicalExam(ctx, companyID, employeeID); err == nil {
		summary.LatestMedical = toMedicalExamDTO(*m)
	}
	if sr, err := s.repo.GetSafetyRecord(ctx, companyID, employeeID); err == nil {
		summary.SafetyRecord = toSafetyRecordDTO(*sr)
	}
	if c, err := s.repo.GetLatestActiveContract(ctx, companyID, employeeID); err == nil {
		summary.ActiveContract = toContractDTO(*c)
	}
	if wp, err := s.repo.GetLatestWorkPermit(ctx, companyID, employeeID); err == nil {
		summary.LatestPermit = toWorkPermitDTO(*wp)
	}
	if pr, err := s.repo.GetPensionRecord(ctx, companyID, employeeID); err == nil {
		summary.PensionRecord = toPensionRecordDTO(*pr)
	}
	if hr, err := s.repo.GetHealthRecord(ctx, companyID, employeeID); err == nil {
		summary.HealthRecord = toHealthRecordDTO(*hr)
	}

	return summary, nil
}

// ── Notifications ─────────────────────────────────────────────────────────────

func (s *DocumentationService) GetNotifications(ctx context.Context, companyID, callerRole string, today time.Time) ([]dto.DocNotification, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}

	employees, err := s.repo.ListActiveEmployees(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.GetNotifications: employees: %w", err)
	}

	medicals, err := s.repo.ListLatestMedicalExams(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.GetNotifications: medicals: %w", err)
	}

	safety, err := s.repo.ListSafetyRecords(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.GetNotifications: safety: %w", err)
	}

	contracts, err := s.repo.ListLatestActiveContracts(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.GetNotifications: contracts: %w", err)
	}

	// Look back to 2020-01 so uncompleted obligations are never silently dropped.
	// The per-employee loop in monthlyNotifications starts at the employee's hire month,
	// so no spurious notifications are generated for months before employment.
	obligations, err := s.repo.ListCompletedObligations(ctx, companyID, 2020, 1)
	if err != nil {
		return nil, fmt.Errorf("doc.GetNotifications: obligations: %w", err)
	}

	permits, err := s.repo.ListLatestWorkPermits(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.GetNotifications: permits: %w", err)
	}

	pensions, err := s.repo.ListPensionRecords(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.GetNotifications: pensions: %w", err)
	}

	healthRecs, err := s.repo.ListHealthRecords(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.GetNotifications: health: %w", err)
	}

	return ComputeDocumentationNotifications(employees, medicals, safety, contracts, obligations, permits, pensions, healthRecs, today), nil
}

// ComputeDocumentationNotifications is pure and exported for unit testing.
func ComputeDocumentationNotifications(
	employees []repositories.DocEmployeeRow,
	medicals map[string]*repositories.MedicalExamRow,
	safetyRecords map[string]*repositories.SafetyRecordRow,
	contracts map[string]*repositories.ContractRow,
	obligations map[string]map[repositories.MonthlyOblKey]bool,
	permits map[string]*repositories.WorkPermitRow,
	pensions map[string]*repositories.PensionRecordRow,
	healthRecs map[string]*repositories.HealthRecordRow,
	today time.Time,
) []dto.DocNotification {
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	var out []dto.DocNotification

	for _, emp := range employees {
		name := emp.FullName()

		// ── Liječnički pregled ────────────────────────────────────────────────
		if m, ok := medicals[emp.ID]; ok {
			expiry := time.Date(m.ExpiresAt.Year(), m.ExpiresAt.Month(), m.ExpiresAt.Day(), 0, 0, 0, 0, time.UTC)
			daysLeft := int(expiry.Sub(todayDate).Hours() / 24)
			switch {
			case daysLeft < 0:
				n := -daysLeft
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifMedicalOverdue, Priority: dto.PriorityOverdue,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: m.ID,
					Message:     fmt.Sprintf("Liječnički pregled istekao prije %d dana", n),
					DaysOverdue: &n,
				})
			case daysLeft == 0:
				zero := 0
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifMedicalUrgent, Priority: dto.PriorityUrgent,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: m.ID,
					Message:       "Liječnički pregled istječe danas",
					DaysRemaining: &zero,
				})
			case daysLeft <= 10:
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifMedicalWarning, Priority: dto.PriorityWarning,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: m.ID,
					Message:       fmt.Sprintf("Liječnički pregled istječe za %d dana", daysLeft),
					DaysRemaining: &daysLeft,
				})
			}
		}

		// ── Zaštita na radu (radnik only) ────────────────────────────────────
		if emp.Role == "radnik" {
			sr, hasRecord := safetyRecords[emp.ID]
			if hasRecord && sr.Status == "verified" {
				// No notification once verified.
			} else {
				refDate := time.Date(emp.CreatedAt.Year(), emp.CreatedAt.Month(), emp.CreatedAt.Day(), 0, 0, 0, 0, time.UTC)
				daysSince := int(todayDate.Sub(refDate).Hours() / 24)
				if daysSince < 0 {
					daysSince = 0
				}
				recID := ""
				if hasRecord {
					recID = sr.ID
				}
				if daysSince >= 60 {
					out = append(out, dto.DocNotification{
						Type: dto.DocNotifSafetyUrgent, Priority: dto.PriorityUrgent,
						EmployeeID: emp.ID, EmployeeName: name, RecordID: recID,
						Message: fmt.Sprintf("Zaštita na radu nije verificirana (%d dana od zapošljavanja)", daysSince),
					})
				} else {
					out = append(out, dto.DocNotification{
						Type: dto.DocNotifSafetyLow, Priority: dto.PriorityLow,
						EmployeeID: emp.ID, EmployeeName: name, RecordID: recID,
						Message: fmt.Sprintf("Zaštita na radu u obradi (%d dana od zapošljavanja)", daysSince),
					})
				}
			}
		}

		// ── Ugovor o radu ────────────────────────────────────────────────────
		if c, ok := contracts[emp.ID]; ok {
			if c.ContractType == "odredeno" && c.ExpiresAt != nil {
				expiry := time.Date(c.ExpiresAt.Year(), c.ExpiresAt.Month(), c.ExpiresAt.Day(), 0, 0, 0, 0, time.UTC)
				daysLeft := int(expiry.Sub(todayDate).Hours() / 24)
				switch {
				case daysLeft < 0:
					n := -daysLeft
					out = append(out, dto.DocNotification{
						Type: dto.DocNotifContractOverdue, Priority: dto.PriorityOverdue,
						EmployeeID: emp.ID, EmployeeName: name, RecordID: c.ID,
						Message:     fmt.Sprintf("Ugovor o radu istekao prije %d dana", n),
						DaysOverdue: &n,
					})
				case daysLeft <= 30:
					out = append(out, dto.DocNotification{
						Type: dto.DocNotifContractWarning, Priority: dto.PriorityWarning,
						EmployeeID: emp.ID, EmployeeName: name, RecordID: c.ID,
						Message:       fmt.Sprintf("Ugovor o radu istječe za %d dana", daysLeft),
						DaysRemaining: &daysLeft,
					})
				}
			}
		}

		// ── Platne liste ──────────────────────────────────────────────────────
		oblMap := obligations[emp.ID]
		out = append(out, monthlyNotifications(emp.ID, name, "platna_lista", dto.DocNotifPayrollWarning, dto.DocNotifPayrollOverdue, oblMap, todayDate, emp.CreatedAt)...)

		// ── Evidencija radnika (radnik only) ─────────────────────────────────
		if emp.Role == "radnik" {
			out = append(out, monthlyNotifications(emp.ID, name, "evidencija_radnika", dto.DocNotifTimesheetWarning, dto.DocNotifTimesheetOverdue, oblMap, todayDate, emp.CreatedAt)...)
		}

		// ── Radne dozvole ─────────────────────────────────────────────────────
		if wp, ok := permits[emp.ID]; ok {
			expiry := time.Date(wp.ExpiresAt.Year(), wp.ExpiresAt.Month(), wp.ExpiresAt.Day(), 0, 0, 0, 0, time.UTC)
			daysLeft := int(expiry.Sub(todayDate).Hours() / 24)

			// Calendar-month thresholds: compare today against AddDate offsets from expiry.
			// This handles variable month lengths (28/29/30/31 days) correctly.
			t4m := expiry.AddDate(0, -4, 0)
			t3m15d := expiry.AddDate(0, -3, -15)
			t3m := expiry.AddDate(0, -3, 0)

			switch {
			case todayDate.After(expiry):
				n := -daysLeft
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifPermitOverdue, Priority: dto.PriorityOverdue,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: wp.ID,
					Message:     fmt.Sprintf("Radna dozvola istekla prije %d dana", n),
					DaysOverdue: &n,
				})
			case !todayDate.Before(t3m): // today >= expiry-3m
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifPermitUrgent, Priority: dto.PriorityUrgent,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: wp.ID,
					Message:       fmt.Sprintf("Radna dozvola istječe za %d dana", daysLeft),
					DaysRemaining: &daysLeft,
				})
			case !todayDate.Before(t3m15d): // today >= expiry-3m-15d
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifPermitWarning, Priority: dto.PriorityWarning,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: wp.ID,
					Message:       fmt.Sprintf("Radna dozvola istječe za %d dana", daysLeft),
					DaysRemaining: &daysLeft,
				})
			case !todayDate.Before(t4m): // today >= expiry-4m
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifPermitLow, Priority: dto.PriorityLow,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: wp.ID,
					Message:       fmt.Sprintf("Radna dozvola istječe za %d dana", daysLeft),
					DaysRemaining: &daysLeft,
				})
			}
		}

		// ── Mirovinsko (radnik only) ──────────────────────────────────────────
		if emp.Role == "radnik" {
			pr, hasPension := pensions[emp.ID]
			if !hasPension || pr.Status != "verified" {
				recID := ""
				if hasPension {
					recID = pr.ID
				}
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifPensionMissing, Priority: dto.PriorityUrgent,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: recID,
					Message: "Mirovinsko osiguranje nije verificirano",
				})
			}
		}

		// ── Zdravstveno (radnik only) ─────────────────────────────────────────
		if emp.Role == "radnik" {
			hr, hasHealth := healthRecs[emp.ID]
			if !hasHealth || hr.Status != "verified" {
				recID := ""
				if hasHealth {
					recID = hr.ID
				}
				out = append(out, dto.DocNotification{
					Type: dto.DocNotifHealthMissing, Priority: dto.PriorityUrgent,
					EmployeeID: emp.ID, EmployeeName: name, RecordID: recID,
					Message: "Zdravstveno osiguranje nije verificirano",
				})
			}
		}
	}

	return out
}

// monthlyNotifications generates warning/overdue notifications for a given monthly obligation type.
// It checks: (a) warning if current month's last ≤5 days and not completed,
// (b) overdue for every month from the employee's hire month up to last month that isn't completed.
// empCreatedAt bounds how far back we look so no spurious notifications appear before employment.
func monthlyNotifications(
	empID, empName, oblType, warnType, overdueType string,
	oblMap map[repositories.MonthlyOblKey]bool,
	today time.Time,
	empCreatedAt time.Time,
) []dto.DocNotification {
	var out []dto.DocNotification

	curYear, curMonth := today.Year(), int(today.Month())

	// Check current month warning (last 5 days of month).
	lastDay := lastDayOfMonth(curYear, curMonth)
	daysToEnd := lastDay - today.Day()
	if daysToEnd < 5 {
		key := repositories.MonthlyOblKey{Year: curYear, Month: curMonth, Type: oblType}
		if !oblMap[key] {
			humanMonth := hrMonths[curMonth]
			out = append(out, dto.DocNotification{
				Type: warnType, Priority: dto.PriorityWarning,
				EmployeeID: empID, EmployeeName: empName,
				Message:        fmt.Sprintf("%s za %s %d. nije riješena", oblLabel(oblType), humanMonth, curYear),
				Year:           &curYear,
				Month:          &curMonth,
				ObligationType: &oblType,
			})
		}
	}

	// Check every month from hire month to the previous month for overdue.
	// Obligations that are still unresolved must remain outstanding until completed.
	hireMonth := time.Date(empCreatedAt.Year(), empCreatedAt.Month(), 1, 0, 0, 0, 0, time.UTC)
	prevMonth := time.Date(curYear, time.Month(curMonth), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	for cursor := hireMonth; !cursor.After(prevMonth); cursor = cursor.AddDate(0, 1, 0) {
		y, m := cursor.Year(), int(cursor.Month())
		key := repositories.MonthlyOblKey{Year: y, Month: m, Type: oblType}
		if !oblMap[key] {
			humanMonth := hrMonths[m]
			mCopy := m
			yCopy := y
			out = append(out, dto.DocNotification{
				Type: overdueType, Priority: dto.PriorityOverdue,
				EmployeeID: empID, EmployeeName: empName,
				Message:        fmt.Sprintf("%s za %s %d. nije riješena", oblLabel(oblType), humanMonth, y),
				Year:           &yCopy,
				Month:          &mCopy,
				ObligationType: &oblType,
			})
		}
	}

	return out
}

func lastDayOfMonth(year, month int) int {
	return time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
}

func oblLabel(oblType string) string {
	if oblType == "platna_lista" {
		return "Platna lista"
	}
	return "Evidencija radnika"
}

// ── DTO conversions ───────────────────────────────────────────────────────────

func toDocEmployeeDTO(r repositories.DocEmployeeRow) dto.DocEmployee {
	return dto.DocEmployee{
		ID:        r.ID,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Role:      r.Role,
		Active:    r.Active,
		CreatedAt: r.CreatedAt,
	}
}

func toDocFileDTO(f *repositories.DocumentFileRow) *dto.DocFile {
	if f == nil {
		return nil
	}
	return &dto.DocFile{
		ID:               f.ID,
		OriginalFilename: f.OriginalFilename,
		MimeType:         f.MimeType,
		FileSize:         f.FileSize,
		UploadedAt:       f.UploadedAt,
	}
}

func toMedicalExamDTO(r repositories.MedicalExamRow) *dto.MedicalExam {
	return &dto.MedicalExam{
		ID:            r.ID,
		EmployeeID:    r.EmployeeID,
		CompletedDate: r.CompletedDate.Format("2006-01-02"),
		ExpiresAt:     r.ExpiresAt.Format("2006-01-02"),
		Document:      toDocFileDTO(r.DocumentFile),
		CreatedAt:     r.CreatedAt,
	}
}

func toSafetyRecordDTO(r repositories.SafetyRecordRow) *dto.SafetyRecord {
	return &dto.SafetyRecord{
		ID:         r.ID,
		EmployeeID: r.EmployeeID,
		Status:     r.Status,
		Document:   toDocFileDTO(r.DocFile),
		VerifiedAt: r.VerifiedAt,
		CreatedAt:  r.CreatedAt,
	}
}

func toContractDTO(r repositories.ContractRow) *dto.EmploymentContract {
	c := &dto.EmploymentContract{
		ID:                  r.ID,
		EmployeeID:          r.EmployeeID,
		ContractType:        r.ContractType,
		Document:            toDocFileDTO(r.DocFile),
		TerminationDocument: toDocFileDTO(r.TerminationDocFile),
		CreatedAt:           r.CreatedAt,
	}
	if r.ExpiresAt != nil {
		s := r.ExpiresAt.Format("2006-01-02")
		c.ExpiresAt = &s
	}
	if r.TerminatedAt != nil {
		s := r.TerminatedAt.Format("2006-01-02")
		c.TerminatedAt = &s
	}
	return c
}

func toMonthlyObligationDTO(r repositories.MonthlyObligationRow) *dto.MonthlyObligation {
	return &dto.MonthlyObligation{
		ID:             r.ID,
		EmployeeID:     r.EmployeeID,
		Year:           r.Year,
		Month:          r.Month,
		ObligationType: r.ObligationType,
		Status:         r.Status,
		Hours:          r.Hours,
		Document:       toDocFileDTO(r.DocFile),
		CompletedAt:    r.CompletedAt,
		CreatedAt:      r.CreatedAt,
	}
}

func toAnnualLeaveDTO(r repositories.AnnualLeaveRow) *dto.AnnualLeave {
	a := &dto.AnnualLeave{
		ID:         r.ID,
		EmployeeID: r.EmployeeID,
		Document:   toDocFileDTO(r.DocFile),
		CreatedAt:  r.CreatedAt,
	}
	if r.DateFrom != nil {
		s := r.DateFrom.Format("2006-01-02")
		a.DateFrom = &s
	}
	if r.DateTo != nil {
		s := r.DateTo.Format("2006-01-02")
		a.DateTo = &s
	}
	return a
}

func toWorkPermitDTO(r repositories.WorkPermitRow) *dto.WorkPermit {
	return &dto.WorkPermit{
		ID:         r.ID,
		EmployeeID: r.EmployeeID,
		EnteredAt:  r.EnteredAt.Format("2006-01-02"),
		ExpiresAt:  r.ExpiresAt.Format("2006-01-02"),
		Document:   toDocFileDTO(r.DocFile),
		CreatedAt:  r.CreatedAt,
	}
}

func toPensionRecordDTO(r repositories.PensionRecordRow) *dto.PensionRecord {
	return &dto.PensionRecord{
		ID:         r.ID,
		EmployeeID: r.EmployeeID,
		Status:     r.Status,
		Document:   toDocFileDTO(r.DocFile),
		VerifiedAt: r.VerifiedAt,
		CreatedAt:  r.CreatedAt,
	}
}

func toHealthRecordDTO(r repositories.HealthRecordRow) *dto.HealthRecord {
	return &dto.HealthRecord{
		ID:         r.ID,
		EmployeeID: r.EmployeeID,
		Status:     r.Status,
		Document:   toDocFileDTO(r.DocFile),
		VerifiedAt: r.VerifiedAt,
		CreatedAt:  r.CreatedAt,
	}
}
