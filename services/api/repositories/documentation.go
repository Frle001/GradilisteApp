package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// ── Row types ─────────────────────────────────────────────────────────────────

type DocEmployeeRow struct {
	ID        string
	FirstName string
	LastName  string
	Role      string
	Active    bool
	CreatedAt time.Time
}

func (r DocEmployeeRow) FullName() string { return r.FirstName + " " + r.LastName }

type DocumentFileRow struct {
	ID               string
	StorageKey       string
	OriginalFilename string
	MimeType         string
	FileSize         int64
	UploadedAt       time.Time
}

type MedicalExamRow struct {
	ID            string
	CompanyID     string
	EmployeeID    string
	CompletedDate time.Time
	ExpiresAt     time.Time
	DocumentFile  *DocumentFileRow
	CreatedAt     time.Time
}

type SafetyRecordRow struct {
	ID         string
	CompanyID  string
	EmployeeID string
	Status     string
	DocFile    *DocumentFileRow
	VerifiedAt *time.Time
	CreatedAt  time.Time
}

type ContractRow struct {
	ID                 string
	CompanyID          string
	EmployeeID         string
	ContractType       string
	ExpiresAt          *time.Time
	DocFile            *DocumentFileRow
	TerminatedAt       *time.Time
	TerminationDocFile *DocumentFileRow
	CreatedAt          time.Time
}

type MonthlyObligationRow struct {
	ID             string
	CompanyID      string
	EmployeeID     string
	Year           int
	Month          int
	ObligationType string
	Status         string
	Hours          *float64
	DocFile        *DocumentFileRow
	CompletedAt    *time.Time
	CreatedAt      time.Time
}

type AnnualLeaveRow struct {
	ID         string
	CompanyID  string
	EmployeeID string
	DateFrom   *time.Time
	DateTo     *time.Time
	DocFile    *DocumentFileRow
	CreatedAt  time.Time
}

type WorkPermitRow struct {
	ID         string
	CompanyID  string
	EmployeeID string
	EnteredAt  time.Time
	ExpiresAt  time.Time
	DocFile    *DocumentFileRow
	CreatedAt  time.Time
}

type PensionRecordRow struct {
	ID         string
	CompanyID  string
	EmployeeID string
	Status     string
	DocFile    *DocumentFileRow
	VerifiedAt *time.Time
	CreatedAt  time.Time
}

type HealthRecordRow struct {
	ID         string
	CompanyID  string
	EmployeeID string
	Status     string
	DocFile    *DocumentFileRow
	VerifiedAt *time.Time
	CreatedAt  time.Time
}

// MonthlyOblKey identifies a specific monthly obligation.
type MonthlyOblKey struct {
	Year  int
	Month int
	Type  string
}

// ── Repository ────────────────────────────────────────────────────────────────

type DocumentationRepository struct {
	db *pgxpool.Pool
}

func NewDocumentationRepository(db *pgxpool.Pool) *DocumentationRepository {
	return &DocumentationRepository{db: db}
}

func (r *DocumentationRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// ── Employees ─────────────────────────────────────────────────────────────────

func scanEmployee(rs rowScanner) (DocEmployeeRow, error) {
	var e DocEmployeeRow
	return e, rs.Scan(&e.ID, &e.FirstName, &e.LastName, &e.Role, &e.Active, &e.CreatedAt)
}

func (r *DocumentationRepository) ListActiveEmployees(ctx context.Context, companyID string) ([]DocEmployeeRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, first_name, last_name, role, active, created_at
		FROM employees WHERE company_id = $1::uuid AND active = true
		ORDER BY last_name, first_name
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListActiveEmployees: %w", err)
	}
	defer rows.Close()
	result := []DocEmployeeRow{}
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *DocumentationRepository) ListActiveRadnikEmployees(ctx context.Context, companyID string) ([]DocEmployeeRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, first_name, last_name, role, active, created_at
		FROM employees WHERE company_id = $1::uuid AND active = true AND role = 'radnik'
		ORDER BY last_name, first_name
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListActiveRadnikEmployees: %w", err)
	}
	defer rows.Close()
	result := []DocEmployeeRow{}
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *DocumentationRepository) GetEmployee(ctx context.Context, companyID, employeeID string) (*DocEmployeeRow, error) {
	e, err := scanEmployee(r.db.QueryRow(ctx, `
		SELECT id::text, first_name, last_name, role, active, created_at
		FROM employees WHERE id = $1::uuid AND company_id = $2::uuid
	`, employeeID, companyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetEmployee: %w", err)
	}
	return &e, nil
}

// ── Document files ────────────────────────────────────────────────────────────

func (r *DocumentationRepository) CreateDocFile(ctx context.Context, companyID, employeeID, storageKey, originalFilename, mimeType, uploadedBy string, fileSize int64) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO employee_documents
			(company_id, employee_id, storage_key, original_filename, mime_type, file_size, uploaded_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7::uuid)
		RETURNING id::text
	`, companyID, employeeID, storageKey, originalFilename, mimeType, fileSize, uploadedBy).Scan(&id)
	return id, err
}

func (r *DocumentationRepository) GetDocFile(ctx context.Context, companyID, fileID string) (*DocumentFileRow, error) {
	var f DocumentFileRow
	var storageKey string
	err := r.db.QueryRow(ctx, `
		SELECT id::text, storage_key, original_filename, mime_type, file_size, uploaded_at
		FROM employee_documents
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, fileID, companyID).Scan(&f.ID, &storageKey, &f.OriginalFilename, &f.MimeType, &f.FileSize, &f.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetDocFile: %w", err)
	}
	f.StorageKey = storageKey
	return &f, nil
}

// ── scan helpers for document join ───────────────────────────────────────────

// scanDocFile reads the four nullable doc columns from a row; returns nil if docID is NULL.
func scanDocFile(docID, docName, docMime *string, docSize *int64, docUpAt *time.Time) *DocumentFileRow {
	if docID == nil {
		return nil
	}
	return &DocumentFileRow{
		ID:               *docID,
		OriginalFilename: *docName,
		MimeType:         *docMime,
		FileSize:         *docSize,
		UploadedAt:       *docUpAt,
	}
}

// ── Medical exams ─────────────────────────────────────────────────────────────

const medCols = `
	e.id::text, e.company_id::text, e.employee_id::text,
	e.completed_date, e.expires_at,
	d.id::text, d.original_filename, d.mime_type, d.file_size, d.uploaded_at,
	e.created_at`

const medJoin = `FROM employee_medical_exams e LEFT JOIN employee_documents d ON d.id = e.document_id`

func scanMedical(rs rowScanner) (MedicalExamRow, error) {
	var m MedicalExamRow
	var dID, dName, dMime *string
	var dSz *int64
	var dUp *time.Time
	err := rs.Scan(&m.ID, &m.CompanyID, &m.EmployeeID, &m.CompletedDate, &m.ExpiresAt,
		&dID, &dName, &dMime, &dSz, &dUp, &m.CreatedAt)
	m.DocumentFile = scanDocFile(dID, dName, dMime, dSz, dUp)
	return m, err
}

func (r *DocumentationRepository) CreateMedicalExam(ctx context.Context, companyID, employeeID, createdBy, documentID string, completedDate, expiresAt time.Time) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO employee_medical_exams (company_id, employee_id, completed_date, expires_at, document_id, created_by)
		VALUES ($1::uuid, $2::uuid, $3::date, $4::date, $5::uuid, $6::uuid)
		RETURNING id::text
	`, companyID, employeeID, completedDate, expiresAt, documentID, createdBy).Scan(&id)
	return id, err
}

func (r *DocumentationRepository) GetLatestMedicalExam(ctx context.Context, companyID, employeeID string) (*MedicalExamRow, error) {
	m, err := scanMedical(r.db.QueryRow(ctx, `SELECT `+medCols+` `+medJoin+`
		WHERE e.company_id = $1::uuid AND e.employee_id = $2::uuid
		ORDER BY e.completed_date DESC LIMIT 1`, companyID, employeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetLatestMedicalExam: %w", err)
	}
	return &m, nil
}

func (r *DocumentationRepository) ListMedicalExams(ctx context.Context, companyID, employeeID string) ([]MedicalExamRow, error) {
	rows, err := r.db.Query(ctx, `SELECT `+medCols+` `+medJoin+`
		WHERE e.company_id = $1::uuid AND e.employee_id = $2::uuid
		ORDER BY e.completed_date DESC`, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListMedicalExams: %w", err)
	}
	defer rows.Close()
	result := []MedicalExamRow{}
	for rows.Next() {
		m, err := scanMedical(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *DocumentationRepository) ListLatestMedicalExams(ctx context.Context, companyID string) (map[string]*MedicalExamRow, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT ON (e.employee_id) `+medCols+` `+medJoin+`
		WHERE e.company_id = $1::uuid
		ORDER BY e.employee_id, e.completed_date DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListLatestMedicalExams: %w", err)
	}
	defer rows.Close()
	result := map[string]*MedicalExamRow{}
	for rows.Next() {
		m, err := scanMedical(rows)
		if err != nil {
			return nil, err
		}
		cp := m
		result[m.EmployeeID] = &cp
	}
	return result, rows.Err()
}

// ── Safety records ────────────────────────────────────────────────────────────

const safetyCols = `
	s.id::text, s.company_id::text, s.employee_id::text, s.status,
	d.id::text, d.original_filename, d.mime_type, d.file_size, d.uploaded_at,
	s.verified_at, s.created_at`

const safetyJoin = `FROM employee_safety_records s LEFT JOIN employee_documents d ON d.id = s.document_id`

func scanSafety(rs rowScanner) (SafetyRecordRow, error) {
	var s SafetyRecordRow
	var dID, dName, dMime *string
	var dSz *int64
	var dUp *time.Time
	err := rs.Scan(&s.ID, &s.CompanyID, &s.EmployeeID, &s.Status,
		&dID, &dName, &dMime, &dSz, &dUp, &s.VerifiedAt, &s.CreatedAt)
	s.DocFile = scanDocFile(dID, dName, dMime, dSz, dUp)
	return s, err
}

func (r *DocumentationRepository) GetSafetyRecord(ctx context.Context, companyID, employeeID string) (*SafetyRecordRow, error) {
	s, err := scanSafety(r.db.QueryRow(ctx, `SELECT `+safetyCols+` `+safetyJoin+`
		WHERE s.company_id = $1::uuid AND s.employee_id = $2::uuid`, companyID, employeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetSafetyRecord: %w", err)
	}
	return &s, nil
}

func (r *DocumentationRepository) UpsertSafetyUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*SafetyRecordRow, error) {
	_, err := r.db.Exec(ctx, `
		INSERT INTO employee_safety_records (company_id, employee_id, status, document_id, created_by)
		VALUES ($1::uuid, $2::uuid, 'uploaded', $3::uuid, $4::uuid)
		ON CONFLICT (company_id, employee_id) DO UPDATE
		SET status = 'uploaded', document_id = EXCLUDED.document_id
	`, companyID, employeeID, documentID, createdBy)
	if err != nil {
		return nil, fmt.Errorf("doc.UpsertSafetyUpload: %w", err)
	}
	return r.GetSafetyRecord(ctx, companyID, employeeID)
}

func (r *DocumentationRepository) VerifySafetyRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*SafetyRecordRow, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE employee_safety_records
		SET status = 'verified', verified_at = NOW(), verified_by = $3::uuid
		WHERE company_id = $1::uuid AND employee_id = $2::uuid AND status = 'uploaded'
	`, companyID, employeeID, verifiedBy)
	if err != nil {
		return nil, fmt.Errorf("doc.VerifySafetyRecord: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetSafetyRecord(ctx, companyID, employeeID)
}

func (r *DocumentationRepository) ListSafetyRecords(ctx context.Context, companyID string) (map[string]*SafetyRecordRow, error) {
	rows, err := r.db.Query(ctx, `SELECT `+safetyCols+` `+safetyJoin+`
		WHERE s.company_id = $1::uuid`, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListSafetyRecords: %w", err)
	}
	defer rows.Close()
	result := map[string]*SafetyRecordRow{}
	for rows.Next() {
		s, err := scanSafety(rows)
		if err != nil {
			return nil, err
		}
		cp := s
		result[s.EmployeeID] = &cp
	}
	return result, rows.Err()
}

// ── Employment contracts ──────────────────────────────────────────────────────

const contractCols = `
	c.id::text, c.company_id::text, c.employee_id::text, c.contract_type, c.expires_at,
	d.id::text,  d.original_filename,  d.mime_type,  d.file_size,  d.uploaded_at,
	c.terminated_at,
	td.id::text, td.original_filename, td.mime_type, td.file_size, td.uploaded_at,
	c.created_at`

const contractJoin = `FROM employee_contracts c
	LEFT JOIN employee_documents  d ON d.id  = c.document_id
	LEFT JOIN employee_documents td ON td.id = c.termination_document_id`

func scanContract(rs rowScanner) (ContractRow, error) {
	var c ContractRow
	var dID, dName, dMime, tdID, tdName, tdMime *string
	var dSz, tdSz *int64
	var dUp, tdUp *time.Time
	err := rs.Scan(&c.ID, &c.CompanyID, &c.EmployeeID, &c.ContractType, &c.ExpiresAt,
		&dID, &dName, &dMime, &dSz, &dUp,
		&c.TerminatedAt,
		&tdID, &tdName, &tdMime, &tdSz, &tdUp,
		&c.CreatedAt)
	c.DocFile = scanDocFile(dID, dName, dMime, dSz, dUp)
	c.TerminationDocFile = scanDocFile(tdID, tdName, tdMime, tdSz, tdUp)
	return c, err
}

func (r *DocumentationRepository) CreateContract(ctx context.Context, companyID, employeeID, contractType, createdBy, documentID string, expiresAt *time.Time) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO employee_contracts (company_id, employee_id, contract_type, expires_at, document_id, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4::date, $5::uuid, $6::uuid)
		RETURNING id::text
	`, companyID, employeeID, contractType, expiresAt, documentID, createdBy).Scan(&id)
	return id, err
}

func (r *DocumentationRepository) GetContract(ctx context.Context, companyID, contractID string) (*ContractRow, error) {
	c, err := scanContract(r.db.QueryRow(ctx, `SELECT `+contractCols+` `+contractJoin+`
		WHERE c.id = $1::uuid AND c.company_id = $2::uuid`, contractID, companyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetContract: %w", err)
	}
	return &c, nil
}

func (r *DocumentationRepository) TerminateContract(ctx context.Context, companyID, contractID, terminatedBy, terminationDocID string, terminatedAt time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE employee_contracts
		SET terminated_at = $3::date, termination_document_id = $4::uuid, terminated_by = $5::uuid
		WHERE id = $1::uuid AND company_id = $2::uuid AND terminated_at IS NULL
	`, contractID, companyID, terminatedAt, terminationDocID, terminatedBy)
	if err != nil {
		return fmt.Errorf("doc.TerminateContract: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DocumentationRepository) ListContracts(ctx context.Context, companyID, employeeID string) ([]ContractRow, error) {
	rows, err := r.db.Query(ctx, `SELECT `+contractCols+` `+contractJoin+`
		WHERE c.company_id = $1::uuid AND c.employee_id = $2::uuid
		ORDER BY c.created_at DESC`, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListContracts: %w", err)
	}
	defer rows.Close()
	result := []ContractRow{}
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *DocumentationRepository) GetLatestActiveContract(ctx context.Context, companyID, employeeID string) (*ContractRow, error) {
	c, err := scanContract(r.db.QueryRow(ctx, `SELECT `+contractCols+` `+contractJoin+`
		WHERE c.company_id = $1::uuid AND c.employee_id = $2::uuid AND c.terminated_at IS NULL
		ORDER BY c.created_at DESC LIMIT 1`, companyID, employeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetLatestActiveContract: %w", err)
	}
	return &c, nil
}

func (r *DocumentationRepository) ListLatestActiveContracts(ctx context.Context, companyID string) (map[string]*ContractRow, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT ON (c.employee_id) `+contractCols+` `+contractJoin+`
		WHERE c.company_id = $1::uuid AND c.terminated_at IS NULL
		ORDER BY c.employee_id, c.created_at DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListLatestActiveContracts: %w", err)
	}
	defer rows.Close()
	result := map[string]*ContractRow{}
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, err
		}
		cp := c
		result[c.EmployeeID] = &cp
	}
	return result, rows.Err()
}

// ── Monthly obligations ───────────────────────────────────────────────────────

const monthCols = `
	o.id::text, o.company_id::text, o.employee_id::text,
	o.year, o.month, o.obligation_type, o.status, o.hours,
	d.id::text, d.original_filename, d.mime_type, d.file_size, d.uploaded_at,
	o.completed_at, o.created_at`

const monthJoin = `FROM employee_monthly_obligations o LEFT JOIN employee_documents d ON d.id = o.document_id`

func scanMonthly(rs rowScanner) (MonthlyObligationRow, error) {
	var m MonthlyObligationRow
	var dID, dName, dMime *string
	var dSz *int64
	var dUp *time.Time
	err := rs.Scan(&m.ID, &m.CompanyID, &m.EmployeeID,
		&m.Year, &m.Month, &m.ObligationType, &m.Status, &m.Hours,
		&dID, &dName, &dMime, &dSz, &dUp,
		&m.CompletedAt, &m.CreatedAt)
	m.DocFile = scanDocFile(dID, dName, dMime, dSz, dUp)
	return m, err
}

func (r *DocumentationRepository) GetMonthlyObligation(ctx context.Context, companyID, employeeID string, year, month int, oblType string) (*MonthlyObligationRow, error) {
	m, err := scanMonthly(r.db.QueryRow(ctx, `SELECT `+monthCols+` `+monthJoin+`
		WHERE o.company_id = $1::uuid AND o.employee_id = $2::uuid
		  AND o.year = $3 AND o.month = $4 AND o.obligation_type = $5`,
		companyID, employeeID, year, month, oblType))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetMonthlyObligation: %w", err)
	}
	return &m, nil
}

func (r *DocumentationRepository) UpsertMonthlyCompletion(ctx context.Context, companyID, employeeID, oblType, completedBy string, year, month int, hours *float64, documentID *string) (*MonthlyObligationRow, error) {
	_, err := r.db.Exec(ctx, `
		INSERT INTO employee_monthly_obligations
			(company_id, employee_id, year, month, obligation_type, status, hours, document_id, completed_at, completed_by, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, 'completed', $6, $7::uuid, NOW(), $8::uuid, $8::uuid)
		ON CONFLICT (company_id, employee_id, year, month, obligation_type) DO UPDATE
		SET status = 'completed', hours = EXCLUDED.hours,
		    document_id = COALESCE(EXCLUDED.document_id, employee_monthly_obligations.document_id),
		    completed_at = NOW(), completed_by = EXCLUDED.completed_by
	`, companyID, employeeID, year, month, oblType, hours, documentID, completedBy)
	if err != nil {
		return nil, fmt.Errorf("doc.UpsertMonthlyCompletion: %w", err)
	}
	return r.GetMonthlyObligation(ctx, companyID, employeeID, year, month, oblType)
}

func (r *DocumentationRepository) ListMonthlyObligations(ctx context.Context, companyID, employeeID string) ([]MonthlyObligationRow, error) {
	rows, err := r.db.Query(ctx, `SELECT `+monthCols+` `+monthJoin+`
		WHERE o.company_id = $1::uuid AND o.employee_id = $2::uuid
		ORDER BY o.year DESC, o.month DESC, o.obligation_type`, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListMonthlyObligations: %w", err)
	}
	defer rows.Close()
	result := []MonthlyObligationRow{}
	for rows.Next() {
		m, err := scanMonthly(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// ListCompletedObligations returns completed obligations per employee since fromYear/fromMonth.
func (r *DocumentationRepository) ListCompletedObligations(ctx context.Context, companyID string, fromYear, fromMonth int) (map[string]map[MonthlyOblKey]bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT employee_id::text, year, month, obligation_type
		FROM employee_monthly_obligations
		WHERE company_id = $1::uuid AND status = 'completed'
		  AND (year > $2 OR (year = $2 AND month >= $3))
	`, companyID, fromYear, fromMonth)
	if err != nil {
		return nil, fmt.Errorf("doc.ListCompletedObligations: %w", err)
	}
	defer rows.Close()
	result := map[string]map[MonthlyOblKey]bool{}
	for rows.Next() {
		var empID, oblType string
		var yr, mo int
		if err := rows.Scan(&empID, &yr, &mo, &oblType); err != nil {
			return nil, err
		}
		if result[empID] == nil {
			result[empID] = map[MonthlyOblKey]bool{}
		}
		result[empID][MonthlyOblKey{Year: yr, Month: mo, Type: oblType}] = true
	}
	return result, rows.Err()
}

// SumMonthlyHoursForWorker totals worker_daily_hours for a given month.
func (r *DocumentationRepository) SumMonthlyHoursForWorker(ctx context.Context, companyID, employeeID string, year, month int) (float64, error) {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	var total float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(hours_worked), 0)
		FROM worker_daily_hours
		WHERE company_id = $1::uuid AND worker_id = $2::uuid
		  AND work_date >= $3::date AND work_date <= $4::date
	`, companyID, employeeID, first, last).Scan(&total)
	return total, err
}

// ── Annual leave ──────────────────────────────────────────────────────────────

func (r *DocumentationRepository) CreateAnnualLeave(ctx context.Context, companyID, employeeID, createdBy string, dateFrom, dateTo *time.Time, documentID *string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO employee_annual_leave (company_id, employee_id, date_from, date_to, document_id, created_by)
		VALUES ($1::uuid, $2::uuid, $3::date, $4::date, $5::uuid, $6::uuid)
		RETURNING id::text
	`, companyID, employeeID, dateFrom, dateTo, documentID, createdBy).Scan(&id)
	return id, err
}

func (r *DocumentationRepository) ListAnnualLeave(ctx context.Context, companyID, employeeID string) ([]AnnualLeaveRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, a.company_id::text, a.employee_id::text,
		       a.date_from, a.date_to,
		       d.id::text, d.original_filename, d.mime_type, d.file_size, d.uploaded_at,
		       a.created_at
		FROM employee_annual_leave a
		LEFT JOIN employee_documents d ON d.id = a.document_id
		WHERE a.company_id = $1::uuid AND a.employee_id = $2::uuid
		ORDER BY a.created_at DESC
	`, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListAnnualLeave: %w", err)
	}
	defer rows.Close()
	result := []AnnualLeaveRow{}
	for rows.Next() {
		var a AnnualLeaveRow
		var dID, dName, dMime *string
		var dSz *int64
		var dUp *time.Time
		if err := rows.Scan(&a.ID, &a.CompanyID, &a.EmployeeID, &a.DateFrom, &a.DateTo,
			&dID, &dName, &dMime, &dSz, &dUp, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.DocFile = scanDocFile(dID, dName, dMime, dSz, dUp)
		result = append(result, a)
	}
	return result, rows.Err()
}

// ── Work permits ──────────────────────────────────────────────────────────────

const permitCols = `
	w.id::text, w.company_id::text, w.employee_id::text, w.entered_at, w.expires_at,
	d.id::text, d.original_filename, d.mime_type, d.file_size, d.uploaded_at,
	w.created_at`

const permitJoin = `FROM employee_work_permits w LEFT JOIN employee_documents d ON d.id = w.document_id`

func scanPermit(rs rowScanner) (WorkPermitRow, error) {
	var w WorkPermitRow
	var dID, dName, dMime *string
	var dSz *int64
	var dUp *time.Time
	err := rs.Scan(&w.ID, &w.CompanyID, &w.EmployeeID, &w.EnteredAt, &w.ExpiresAt,
		&dID, &dName, &dMime, &dSz, &dUp, &w.CreatedAt)
	w.DocFile = scanDocFile(dID, dName, dMime, dSz, dUp)
	return w, err
}

func (r *DocumentationRepository) CreateWorkPermit(ctx context.Context, companyID, employeeID, createdBy, documentID string, enteredAt, expiresAt time.Time) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO employee_work_permits (company_id, employee_id, entered_at, expires_at, document_id, created_by)
		VALUES ($1::uuid, $2::uuid, $3::date, $4::date, $5::uuid, $6::uuid)
		RETURNING id::text
	`, companyID, employeeID, enteredAt, expiresAt, documentID, createdBy).Scan(&id)
	return id, err
}

func (r *DocumentationRepository) GetLatestWorkPermit(ctx context.Context, companyID, employeeID string) (*WorkPermitRow, error) {
	w, err := scanPermit(r.db.QueryRow(ctx, `SELECT `+permitCols+` `+permitJoin+`
		WHERE w.company_id = $1::uuid AND w.employee_id = $2::uuid
		ORDER BY w.expires_at DESC LIMIT 1`, companyID, employeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetLatestWorkPermit: %w", err)
	}
	return &w, nil
}

func (r *DocumentationRepository) ListWorkPermits(ctx context.Context, companyID, employeeID string) ([]WorkPermitRow, error) {
	rows, err := r.db.Query(ctx, `SELECT `+permitCols+` `+permitJoin+`
		WHERE w.company_id = $1::uuid AND w.employee_id = $2::uuid
		ORDER BY w.expires_at DESC`, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListWorkPermits: %w", err)
	}
	defer rows.Close()
	result := []WorkPermitRow{}
	for rows.Next() {
		w, err := scanPermit(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, rows.Err()
}

func (r *DocumentationRepository) ListLatestWorkPermits(ctx context.Context, companyID string) (map[string]*WorkPermitRow, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT ON (w.employee_id) `+permitCols+` `+permitJoin+`
		WHERE w.company_id = $1::uuid
		ORDER BY w.employee_id, w.expires_at DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListLatestWorkPermits: %w", err)
	}
	defer rows.Close()
	result := map[string]*WorkPermitRow{}
	for rows.Next() {
		w, err := scanPermit(rows)
		if err != nil {
			return nil, err
		}
		cp := w
		result[w.EmployeeID] = &cp
	}
	return result, rows.Err()
}

// ── Pension records ───────────────────────────────────────────────────────────

const pensionCols = `
	r.id::text, r.company_id::text, r.employee_id::text, r.status,
	d.id::text, d.original_filename, d.mime_type, d.file_size, d.uploaded_at,
	r.verified_at, r.created_at`

func scanPension(rs rowScanner) (PensionRecordRow, error) {
	var p PensionRecordRow
	var dID, dName, dMime *string
	var dSz *int64
	var dUp *time.Time
	err := rs.Scan(&p.ID, &p.CompanyID, &p.EmployeeID, &p.Status,
		&dID, &dName, &dMime, &dSz, &dUp, &p.VerifiedAt, &p.CreatedAt)
	p.DocFile = scanDocFile(dID, dName, dMime, dSz, dUp)
	return p, err
}

func (r *DocumentationRepository) GetPensionRecord(ctx context.Context, companyID, employeeID string) (*PensionRecordRow, error) {
	p, err := scanPension(r.db.QueryRow(ctx, `SELECT `+pensionCols+`
		FROM employee_pension_records r LEFT JOIN employee_documents d ON d.id = r.document_id
		WHERE r.company_id = $1::uuid AND r.employee_id = $2::uuid`, companyID, employeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetPensionRecord: %w", err)
	}
	return &p, nil
}

func (r *DocumentationRepository) UpsertPensionUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*PensionRecordRow, error) {
	_, err := r.db.Exec(ctx, `
		INSERT INTO employee_pension_records (company_id, employee_id, status, document_id, created_by)
		VALUES ($1::uuid, $2::uuid, 'uploaded', $3::uuid, $4::uuid)
		ON CONFLICT (company_id, employee_id) DO UPDATE
		SET status = 'uploaded', document_id = EXCLUDED.document_id
	`, companyID, employeeID, documentID, createdBy)
	if err != nil {
		return nil, fmt.Errorf("doc.UpsertPensionUpload: %w", err)
	}
	return r.GetPensionRecord(ctx, companyID, employeeID)
}

func (r *DocumentationRepository) VerifyPensionRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*PensionRecordRow, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE employee_pension_records
		SET status = 'verified', verified_at = NOW(), verified_by = $3::uuid
		WHERE company_id = $1::uuid AND employee_id = $2::uuid AND status = 'uploaded'
	`, companyID, employeeID, verifiedBy)
	if err != nil {
		return nil, fmt.Errorf("doc.VerifyPensionRecord: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetPensionRecord(ctx, companyID, employeeID)
}

func (r *DocumentationRepository) ListPensionRecords(ctx context.Context, companyID string) (map[string]*PensionRecordRow, error) {
	rows, err := r.db.Query(ctx, `SELECT `+pensionCols+`
		FROM employee_pension_records r LEFT JOIN employee_documents d ON d.id = r.document_id
		WHERE r.company_id = $1::uuid`, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListPensionRecords: %w", err)
	}
	defer rows.Close()
	result := map[string]*PensionRecordRow{}
	for rows.Next() {
		p, err := scanPension(rows)
		if err != nil {
			return nil, err
		}
		cp := p
		result[p.EmployeeID] = &cp
	}
	return result, rows.Err()
}

// ── Health records ────────────────────────────────────────────────────────────

const healthCols = `
	r.id::text, r.company_id::text, r.employee_id::text, r.status,
	d.id::text, d.original_filename, d.mime_type, d.file_size, d.uploaded_at,
	r.verified_at, r.created_at`

func scanHealth(rs rowScanner) (HealthRecordRow, error) {
	var h HealthRecordRow
	var dID, dName, dMime *string
	var dSz *int64
	var dUp *time.Time
	err := rs.Scan(&h.ID, &h.CompanyID, &h.EmployeeID, &h.Status,
		&dID, &dName, &dMime, &dSz, &dUp, &h.VerifiedAt, &h.CreatedAt)
	h.DocFile = scanDocFile(dID, dName, dMime, dSz, dUp)
	return h, err
}

func (r *DocumentationRepository) GetHealthRecord(ctx context.Context, companyID, employeeID string) (*HealthRecordRow, error) {
	h, err := scanHealth(r.db.QueryRow(ctx, `SELECT `+healthCols+`
		FROM employee_health_records r LEFT JOIN employee_documents d ON d.id = r.document_id
		WHERE r.company_id = $1::uuid AND r.employee_id = $2::uuid`, companyID, employeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("doc.GetHealthRecord: %w", err)
	}
	return &h, nil
}

func (r *DocumentationRepository) UpsertHealthUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*HealthRecordRow, error) {
	_, err := r.db.Exec(ctx, `
		INSERT INTO employee_health_records (company_id, employee_id, status, document_id, created_by)
		VALUES ($1::uuid, $2::uuid, 'uploaded', $3::uuid, $4::uuid)
		ON CONFLICT (company_id, employee_id) DO UPDATE
		SET status = 'uploaded', document_id = EXCLUDED.document_id
	`, companyID, employeeID, documentID, createdBy)
	if err != nil {
		return nil, fmt.Errorf("doc.UpsertHealthUpload: %w", err)
	}
	return r.GetHealthRecord(ctx, companyID, employeeID)
}

func (r *DocumentationRepository) VerifyHealthRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*HealthRecordRow, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE employee_health_records
		SET status = 'verified', verified_at = NOW(), verified_by = $3::uuid
		WHERE company_id = $1::uuid AND employee_id = $2::uuid AND status = 'uploaded'
	`, companyID, employeeID, verifiedBy)
	if err != nil {
		return nil, fmt.Errorf("doc.VerifyHealthRecord: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetHealthRecord(ctx, companyID, employeeID)
}

func (r *DocumentationRepository) ListHealthRecords(ctx context.Context, companyID string) (map[string]*HealthRecordRow, error) {
	rows, err := r.db.Query(ctx, `SELECT `+healthCols+`
		FROM employee_health_records r LEFT JOIN employee_documents d ON d.id = r.document_id
		WHERE r.company_id = $1::uuid`, companyID)
	if err != nil {
		return nil, fmt.Errorf("doc.ListHealthRecords: %w", err)
	}
	defer rows.Close()
	result := map[string]*HealthRecordRow{}
	for rows.Next() {
		h, err := scanHealth(rows)
		if err != nil {
			return nil, err
		}
		cp := h
		result[h.EmployeeID] = &cp
	}
	return result, rows.Err()
}
