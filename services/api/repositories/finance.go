package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Row types ─────────────────────────────────────────────────────────────────

type CompanyInvoiceRow struct {
	ID               string
	InvoiceType      string
	Supplier         *string
	LeasingCompany   *string
	StorageKey       string
	OriginalFilename string
	MimeType         string
	FileSize         int64
	CreatedBy        string
	CreatedByName    string
	CreatedAt        time.Time
}

type R1ReceiptRow struct {
	ID               string
	SubmittedBy      string
	SubmitterName    string
	Price            float64
	StorageKey       string
	OriginalFilename string
	MimeType         string
	FileSize         int64
	CreatedAt        time.Time
}

// ── Params ────────────────────────────────────────────────────────────────────

type CreateInvoiceParams struct {
	CompanyID        string
	InvoiceType      string
	Supplier         *string
	LeasingCompany   *string
	StorageKey       string
	OriginalFilename string
	MimeType         string
	FileSize         int64
	CreatedBy        string
}

type CreateR1Params struct {
	CompanyID        string
	SubmittedBy      string
	Price            float64
	StorageKey       string
	OriginalFilename string
	MimeType         string
	FileSize         int64
}

// ── Repository ────────────────────────────────────────────────────────────────

type FinanceRepository struct {
	db *pgxpool.Pool
}

func NewFinanceRepository(db *pgxpool.Pool) *FinanceRepository {
	return &FinanceRepository{db: db}
}

// ── Company Invoices ──────────────────────────────────────────────────────────

func (r *FinanceRepository) CreateInvoice(ctx context.Context, p CreateInvoiceParams) (*CompanyInvoiceRow, error) {
	const q = `
		INSERT INTO company_invoices
			(company_id, invoice_type, supplier, leasing_company,
			 storage_key, original_filename, mime_type, file_size, created_by)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9::uuid)
		RETURNING
			id::text, invoice_type, supplier, leasing_company,
			storage_key, original_filename, mime_type, file_size,
			created_by::text, created_at`

	row := r.db.QueryRow(ctx, q,
		p.CompanyID, p.InvoiceType, p.Supplier, p.LeasingCompany,
		p.StorageKey, p.OriginalFilename, p.MimeType, p.FileSize, p.CreatedBy,
	)
	var inv CompanyInvoiceRow
	if err := row.Scan(
		&inv.ID, &inv.InvoiceType, &inv.Supplier, &inv.LeasingCompany,
		&inv.StorageKey, &inv.OriginalFilename, &inv.MimeType, &inv.FileSize,
		&inv.CreatedBy, &inv.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("finance.CreateInvoice: %w", err)
	}
	inv.CreatedByName = inv.CreatedBy
	return &inv, nil
}

func (r *FinanceRepository) ListInvoices(ctx context.Context, companyID string) ([]CompanyInvoiceRow, error) {
	const q = `
		SELECT ci.id::text, ci.invoice_type, ci.supplier, ci.leasing_company,
		       ci.storage_key, ci.original_filename, ci.mime_type, ci.file_size,
		       ci.created_by::text, ci.created_at,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS created_by_name
		FROM company_invoices ci
		JOIN users u ON u.id = ci.created_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE ci.company_id = $1::uuid
		ORDER BY ci.created_at DESC`

	rows, err := r.db.Query(ctx, q, companyID)
	if err != nil {
		return nil, fmt.Errorf("finance.ListInvoices: %w", err)
	}
	defer rows.Close()

	var out []CompanyInvoiceRow
	for rows.Next() {
		var inv CompanyInvoiceRow
		if err := rows.Scan(
			&inv.ID, &inv.InvoiceType, &inv.Supplier, &inv.LeasingCompany,
			&inv.StorageKey, &inv.OriginalFilename, &inv.MimeType, &inv.FileSize,
			&inv.CreatedBy, &inv.CreatedAt, &inv.CreatedByName,
		); err != nil {
			return nil, fmt.Errorf("finance.ListInvoices scan: %w", err)
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *FinanceRepository) GetInvoice(ctx context.Context, companyID, invoiceID string) (*CompanyInvoiceRow, error) {
	const q = `
		SELECT ci.id::text, ci.invoice_type, ci.supplier, ci.leasing_company,
		       ci.storage_key, ci.original_filename, ci.mime_type, ci.file_size,
		       ci.created_by::text, ci.created_at,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS created_by_name
		FROM company_invoices ci
		JOIN users u ON u.id = ci.created_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE ci.id = $1::uuid AND ci.company_id = $2::uuid`

	row := r.db.QueryRow(ctx, q, invoiceID, companyID)
	var inv CompanyInvoiceRow
	if err := row.Scan(
		&inv.ID, &inv.InvoiceType, &inv.Supplier, &inv.LeasingCompany,
		&inv.StorageKey, &inv.OriginalFilename, &inv.MimeType, &inv.FileSize,
		&inv.CreatedBy, &inv.CreatedAt, &inv.CreatedByName,
	); err != nil {
		return nil, fmt.Errorf("finance.GetInvoice: %w", err)
	}
	return &inv, nil
}

// ── R1 Receipts ───────────────────────────────────────────────────────────────

func (r *FinanceRepository) CreateR1Receipt(ctx context.Context, p CreateR1Params) (*R1ReceiptRow, error) {
	const q = `
		INSERT INTO r1_receipts
			(company_id, submitted_by, price, storage_key, original_filename, mime_type, file_size)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
		RETURNING id::text, submitted_by::text, price, storage_key, original_filename, mime_type, file_size, created_at`

	row := r.db.QueryRow(ctx, q,
		p.CompanyID, p.SubmittedBy, p.Price,
		p.StorageKey, p.OriginalFilename, p.MimeType, p.FileSize,
	)
	var rec R1ReceiptRow
	if err := row.Scan(
		&rec.ID, &rec.SubmittedBy, &rec.Price,
		&rec.StorageKey, &rec.OriginalFilename, &rec.MimeType, &rec.FileSize,
		&rec.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("finance.CreateR1Receipt: %w", err)
	}
	rec.SubmitterName = rec.SubmittedBy
	return &rec, nil
}

// ListR1Receipts returns receipts scoped to the company. When submittedBy is
// non-nil, only that user's receipts are returned (for non-director roles).
func (r *FinanceRepository) ListR1Receipts(ctx context.Context, companyID string, submittedBy *string) ([]R1ReceiptRow, error) {
	const qAll = `
		SELECT rr.id::text, rr.submitted_by::text,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS submitter_name,
		       rr.price, rr.storage_key, rr.original_filename, rr.mime_type, rr.file_size, rr.created_at
		FROM r1_receipts rr
		JOIN users u ON u.id = rr.submitted_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE rr.company_id = $1::uuid
		ORDER BY rr.created_at DESC`

	const qOwn = `
		SELECT rr.id::text, rr.submitted_by::text,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS submitter_name,
		       rr.price, rr.storage_key, rr.original_filename, rr.mime_type, rr.file_size, rr.created_at
		FROM r1_receipts rr
		JOIN users u ON u.id = rr.submitted_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE rr.company_id = $1::uuid AND rr.submitted_by = $2::uuid
		ORDER BY rr.created_at DESC`

	var (
		rows pgx.Rows
		err  error
	)
	if submittedBy == nil {
		rows, err = r.db.Query(ctx, qAll, companyID)
	} else {
		rows, err = r.db.Query(ctx, qOwn, companyID, *submittedBy)
	}
	if err != nil {
		return nil, fmt.Errorf("finance.ListR1Receipts: %w", err)
	}
	defer rows.Close()

	var out []R1ReceiptRow
	for rows.Next() {
		var rec R1ReceiptRow
		if err := rows.Scan(
			&rec.ID, &rec.SubmittedBy, &rec.SubmitterName,
			&rec.Price, &rec.StorageKey, &rec.OriginalFilename, &rec.MimeType, &rec.FileSize,
			&rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("finance.ListR1Receipts scan: %w", err)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *FinanceRepository) GetR1Receipt(ctx context.Context, companyID, receiptID string) (*R1ReceiptRow, error) {
	const q = `
		SELECT rr.id::text, rr.submitted_by::text,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS submitter_name,
		       rr.price, rr.storage_key, rr.original_filename, rr.mime_type, rr.file_size, rr.created_at
		FROM r1_receipts rr
		JOIN users u ON u.id = rr.submitted_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE rr.id = $1::uuid AND rr.company_id = $2::uuid`

	row := r.db.QueryRow(ctx, q, receiptID, companyID)
	var rec R1ReceiptRow
	if err := row.Scan(
		&rec.ID, &rec.SubmittedBy, &rec.SubmitterName,
		&rec.Price, &rec.StorageKey, &rec.OriginalFilename, &rec.MimeType, &rec.FileSize,
		&rec.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("finance.GetR1Receipt: %w", err)
	}
	return &rec, nil
}

// ── Fuel Receipts ─────────────────────────────────────────────────────────────

type FuelReceiptRow struct {
	ID                  string
	SubmittedBy         string
	SubmitterName       string
	ReceiptDate         time.Time
	AmountEur           float64
	StorageKey          string
	OriginalFilename    string
	MimeType            string
	FileSize            int64
	Notes               *string
	VehicleRegistration *string
	FuelLiters          *float64
	FuelType            *string
	CreatedAt           time.Time
}

type CreateFuelReceiptParams struct {
	CompanyID           string
	SubmittedBy         string
	ReceiptDate         time.Time
	AmountEur           float64
	StorageKey          string
	OriginalFilename    string
	MimeType            string
	FileSize            int64
	Notes               *string
	VehicleRegistration *string
	FuelLiters          *float64
	FuelType            *string
}

func (r *FinanceRepository) CreateFuelReceipt(ctx context.Context, p CreateFuelReceiptParams) (*FuelReceiptRow, error) {
	const q = `
		INSERT INTO fuel_receipts
			(company_id, submitted_by, receipt_date, amount_eur,
			 storage_key, original_filename, mime_type, file_size,
			 notes, vehicle_registration, fuel_liters, fuel_type)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING
			id::text, submitted_by::text, receipt_date, amount_eur,
			storage_key, original_filename, mime_type, file_size,
			notes, vehicle_registration, fuel_liters, fuel_type, created_at`

	row := r.db.QueryRow(ctx, q,
		p.CompanyID, p.SubmittedBy, p.ReceiptDate, p.AmountEur,
		p.StorageKey, p.OriginalFilename, p.MimeType, p.FileSize,
		p.Notes, p.VehicleRegistration, p.FuelLiters, p.FuelType,
	)
	var rec FuelReceiptRow
	if err := row.Scan(
		&rec.ID, &rec.SubmittedBy, &rec.ReceiptDate, &rec.AmountEur,
		&rec.StorageKey, &rec.OriginalFilename, &rec.MimeType, &rec.FileSize,
		&rec.Notes, &rec.VehicleRegistration, &rec.FuelLiters, &rec.FuelType,
		&rec.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("finance.CreateFuelReceipt: %w", err)
	}
	rec.SubmitterName = rec.SubmittedBy
	return &rec, nil
}

// ListFuelReceipts returns receipts scoped to the company. When submittedBy is
// non-nil, only that user's receipts are returned (for non-admin roles).
func (r *FinanceRepository) ListFuelReceipts(ctx context.Context, companyID string, submittedBy *string) ([]FuelReceiptRow, error) {
	const qAll = `
		SELECT fr.id::text, fr.submitted_by::text,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS submitter_name,
		       fr.receipt_date, fr.amount_eur,
		       fr.storage_key, fr.original_filename, fr.mime_type, fr.file_size,
		       fr.notes, fr.vehicle_registration, fr.fuel_liters, fr.fuel_type, fr.created_at
		FROM fuel_receipts fr
		JOIN users u ON u.id = fr.submitted_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE fr.company_id = $1::uuid
		ORDER BY fr.created_at DESC`

	const qOwn = `
		SELECT fr.id::text, fr.submitted_by::text,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS submitter_name,
		       fr.receipt_date, fr.amount_eur,
		       fr.storage_key, fr.original_filename, fr.mime_type, fr.file_size,
		       fr.notes, fr.vehicle_registration, fr.fuel_liters, fr.fuel_type, fr.created_at
		FROM fuel_receipts fr
		JOIN users u ON u.id = fr.submitted_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE fr.company_id = $1::uuid AND fr.submitted_by = $2::uuid
		ORDER BY fr.created_at DESC`

	var (
		rows pgx.Rows
		err  error
	)
	if submittedBy == nil {
		rows, err = r.db.Query(ctx, qAll, companyID)
	} else {
		rows, err = r.db.Query(ctx, qOwn, companyID, *submittedBy)
	}
	if err != nil {
		return nil, fmt.Errorf("finance.ListFuelReceipts: %w", err)
	}
	defer rows.Close()

	var out []FuelReceiptRow
	for rows.Next() {
		var rec FuelReceiptRow
		if err := rows.Scan(
			&rec.ID, &rec.SubmittedBy, &rec.SubmitterName,
			&rec.ReceiptDate, &rec.AmountEur,
			&rec.StorageKey, &rec.OriginalFilename, &rec.MimeType, &rec.FileSize,
			&rec.Notes, &rec.VehicleRegistration, &rec.FuelLiters, &rec.FuelType,
			&rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("finance.ListFuelReceipts scan: %w", err)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *FinanceRepository) GetFuelReceipt(ctx context.Context, companyID, receiptID string) (*FuelReceiptRow, error) {
	const q = `
		SELECT fr.id::text, fr.submitted_by::text,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS submitter_name,
		       fr.receipt_date, fr.amount_eur,
		       fr.storage_key, fr.original_filename, fr.mime_type, fr.file_size,
		       fr.notes, fr.vehicle_registration, fr.fuel_liters, fr.fuel_type, fr.created_at
		FROM fuel_receipts fr
		JOIN users u ON u.id = fr.submitted_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE fr.id = $1::uuid AND fr.company_id = $2::uuid`

	row := r.db.QueryRow(ctx, q, receiptID, companyID)
	var rec FuelReceiptRow
	if err := row.Scan(
		&rec.ID, &rec.SubmittedBy, &rec.SubmitterName,
		&rec.ReceiptDate, &rec.AmountEur,
		&rec.StorageKey, &rec.OriginalFilename, &rec.MimeType, &rec.FileSize,
		&rec.Notes, &rec.VehicleRegistration, &rec.FuelLiters, &rec.FuelType,
		&rec.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("finance.GetFuelReceipt: %w", err)
	}
	return &rec, nil
}

func (r *FinanceRepository) DeleteFuelReceipt(ctx context.Context, companyID, receiptID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM fuel_receipts WHERE id = $1::uuid AND company_id = $2::uuid
	`, receiptID, companyID)
	if err != nil {
		return fmt.Errorf("finance.DeleteFuelReceipt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
