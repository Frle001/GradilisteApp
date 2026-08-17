package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicateLeasingPayment is returned when a leasing payment already exists for the given month.
var ErrDuplicateLeasingPayment = errors.New("leasing payment already recorded for this month")

// ── Row types ─────────────────────────────────────────────────────────────────

type CompanyAssetRow struct {
	ID                    string
	CompanyID             string
	AssetType             string
	Name                  string
	Status                string
	Notes                 *string
	AssignedEmployeeID    *string
	AssignedFirstName     *string
	AssignedLastName      *string
	AssignedRole          *string
	PurchasedAt           *time.Time
	WarrantyExpiresAt     *time.Time
	Size                  *string
	RegistrationPlate     *string
	RegistrationDate      *time.Time
	RegistrationExpiresAt *time.Time
	// Leasing (vozilo only)
	IsLeasing      bool
	LeasingCompany *string
	LeasingEndDate *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AssetEmployeeRow struct {
	ID        string
	FirstName string
	LastName  string
	Role      string
}

type LeasingPaymentRow struct {
	ID          string
	CompanyID   string
	AssetID     string
	PeriodMonth time.Time
	CompletedAt time.Time
	CompletedBy string
	CreatedAt   time.Time
}

type CompanyAssetsFilter struct {
	AssetType  string
	EmployeeID string
	Status     string
	Search     string
}

// ── Repository ────────────────────────────────────────────────────────────────

type CompanyAssetsRepository struct {
	db *pgxpool.Pool
}

func NewCompanyAssetsRepository(db *pgxpool.Pool) *CompanyAssetsRepository {
	return &CompanyAssetsRepository{db: db}
}

const assetCols = `
    a.id, a.company_id, a.asset_type, a.name, a.status, a.notes,
    a.assigned_employee_id,
    e.first_name, e.last_name, e.role,
    a.purchased_at, a.warranty_expires_at,
    a.size,
    a.registration_plate, a.registration_date, a.registration_expires_at,
    a.is_leasing, a.leasing_company, a.leasing_end_date,
    a.created_at, a.updated_at`

func scanAssetCols(s interface {
	Scan(...any) error
}) (CompanyAssetRow, error) {
	var r CompanyAssetRow
	err := s.Scan(
		&r.ID, &r.CompanyID, &r.AssetType, &r.Name, &r.Status, &r.Notes,
		&r.AssignedEmployeeID,
		&r.AssignedFirstName, &r.AssignedLastName, &r.AssignedRole,
		&r.PurchasedAt, &r.WarrantyExpiresAt,
		&r.Size,
		&r.RegistrationPlate, &r.RegistrationDate, &r.RegistrationExpiresAt,
		&r.IsLeasing, &r.LeasingCompany, &r.LeasingEndDate,
		&r.CreatedAt, &r.UpdatedAt,
	)
	return r, err
}

func (r *CompanyAssetsRepository) List(ctx context.Context, companyID string, f CompanyAssetsFilter) ([]CompanyAssetRow, error) {
	conditions := []string{"a.company_id = $1::uuid"}
	args := []interface{}{companyID}
	i := 2

	if f.AssetType != "" {
		conditions = append(conditions, fmt.Sprintf("a.asset_type = $%d", i))
		args = append(args, f.AssetType)
		i++
	}
	if f.EmployeeID != "" {
		conditions = append(conditions, fmt.Sprintf("a.assigned_employee_id = $%d::uuid", i))
		args = append(args, f.EmployeeID)
		i++
	}
	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", i))
		args = append(args, f.Status)
		i++
	}
	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(a.name ILIKE $%d OR a.registration_plate ILIKE $%d)", i, i,
		))
		args = append(args, "%"+f.Search+"%")
		i++
	}
	_ = i

	q := fmt.Sprintf(`
		SELECT %s
		FROM company_assets a
		LEFT JOIN employees e ON e.id = a.assigned_employee_id
		WHERE %s
		ORDER BY a.created_at DESC
	`, assetCols, strings.Join(conditions, " AND "))

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompanyAssetRow
	for rows.Next() {
		row, err := scanAssetCols(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *CompanyAssetsRepository) GetByID(ctx context.Context, companyID, id string) (*CompanyAssetRow, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM company_assets a
		LEFT JOIN employees e ON e.id = a.assigned_employee_id
		WHERE a.company_id = $1::uuid AND a.id = $2::uuid
	`, assetCols)

	row, err := scanAssetCols(r.db.QueryRow(ctx, q, companyID, id))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CompanyAssetsRepository) Create(ctx context.Context, companyID string, row CompanyAssetRow) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO company_assets (
			company_id, assigned_employee_id, asset_type, name, status, notes,
			purchased_at, warranty_expires_at, size,
			registration_plate, registration_date, registration_expires_at,
			is_leasing, leasing_company, leasing_end_date
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11, $12,
			$13, $14, $15
		)
		RETURNING id
	`,
		companyID, row.AssignedEmployeeID, row.AssetType, row.Name, row.Status, row.Notes,
		row.PurchasedAt, row.WarrantyExpiresAt, row.Size,
		row.RegistrationPlate, row.RegistrationDate, row.RegistrationExpiresAt,
		row.IsLeasing, row.LeasingCompany, row.LeasingEndDate,
	).Scan(&id)
	return id, err
}

func (r *CompanyAssetsRepository) Update(ctx context.Context, companyID, id string, row CompanyAssetRow) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE company_assets SET
			assigned_employee_id    = $3::uuid,
			name                    = $4,
			status                  = $5,
			notes                   = $6,
			purchased_at            = $7,
			warranty_expires_at     = $8,
			size                    = $9,
			registration_plate      = $10,
			registration_date       = $11,
			registration_expires_at = $12,
			is_leasing              = $13,
			leasing_company         = $14,
			leasing_end_date        = $15
		WHERE company_id = $1::uuid AND id = $2::uuid
	`,
		companyID, id,
		row.AssignedEmployeeID, row.Name, row.Status, row.Notes,
		row.PurchasedAt, row.WarrantyExpiresAt, row.Size,
		row.RegistrationPlate, row.RegistrationDate, row.RegistrationExpiresAt,
		row.IsLeasing, row.LeasingCompany, row.LeasingEndDate,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CompanyAssetsRepository) Deactivate(ctx context.Context, companyID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE company_assets SET status = 'inactive'
		WHERE company_id = $1::uuid AND id = $2::uuid
	`, companyID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CompanyAssetsRepository) ListActiveEmployees(ctx context.Context, companyID string) ([]AssetEmployeeRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, first_name, last_name, role
		FROM employees
		WHERE company_id = $1::uuid AND active = true
		ORDER BY last_name, first_name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AssetEmployeeRow
	for rows.Next() {
		var e AssetEmployeeRow
		if err := rows.Scan(&e.ID, &e.FirstName, &e.LastName, &e.Role); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *CompanyAssetsRepository) EmployeeBelongsToCompany(ctx context.Context, companyID, employeeID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM employees
			WHERE id = $1::uuid AND company_id = $2::uuid AND active = true
		)
	`, employeeID, companyID).Scan(&exists)
	return exists, err
}

// ── Leasing payments ──────────────────────────────────────────────────────────

// CreateLeasingPayment inserts a new monthly leasing completion record.
// Returns ErrDuplicateLeasingPayment if a record for (asset, period_month) already exists.
func (r *CompanyAssetsRepository) CreateLeasingPayment(ctx context.Context, row LeasingPaymentRow) (*LeasingPaymentRow, error) {
	var id string
	var completedAt, createdAt time.Time

	err := r.db.QueryRow(ctx, `
		INSERT INTO company_asset_leasing_payments
			(company_id, company_asset_id, period_month, completed_by)
		VALUES ($1::uuid, $2::uuid, $3, $4::uuid)
		ON CONFLICT (company_asset_id, period_month) DO NOTHING
		RETURNING id, completed_at, created_at
	`, row.CompanyID, row.AssetID, row.PeriodMonth, row.CompletedBy).Scan(&id, &completedAt, &createdAt)

	if err == pgx.ErrNoRows {
		return nil, ErrDuplicateLeasingPayment
	}
	if err != nil {
		return nil, err
	}

	return &LeasingPaymentRow{
		ID:          id,
		CompanyID:   row.CompanyID,
		AssetID:     row.AssetID,
		PeriodMonth: row.PeriodMonth,
		CompletedAt: completedAt,
		CompletedBy: row.CompletedBy,
		CreatedAt:   createdAt,
	}, nil
}

// ListCompletedLeasingForPeriod returns a set of asset IDs that have a completed
// payment for the given period_month within the company.
func (r *CompanyAssetsRepository) ListCompletedLeasingForPeriod(ctx context.Context, companyID string, periodMonth time.Time) (map[string]bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT company_asset_id
		FROM company_asset_leasing_payments
		WHERE company_id = $1::uuid AND period_month = $2
	`, companyID, periodMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var assetID string
		if err := rows.Scan(&assetID); err != nil {
			return nil, err
		}
		out[assetID] = true
	}
	return out, rows.Err()
}

// GetLeasingPaymentsByAsset returns all leasing payment records for an asset, newest first.
func (r *CompanyAssetsRepository) GetLeasingPaymentsByAsset(ctx context.Context, companyID, assetID string) ([]LeasingPaymentRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, company_id, company_asset_id, period_month, completed_at, completed_by, created_at
		FROM company_asset_leasing_payments
		WHERE company_id = $1::uuid AND company_asset_id = $2::uuid
		ORDER BY period_month DESC
	`, companyID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LeasingPaymentRow
	for rows.Next() {
		var p LeasingPaymentRow
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.AssetID, &p.PeriodMonth, &p.CompletedAt, &p.CompletedBy, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
