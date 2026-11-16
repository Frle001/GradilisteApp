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

// ── Row types ─────────────────────────────────────────────────────────────────

type ProjectContractRow struct {
	ID           string
	ProjectID    string
	ProjectName  string
	ContractType string
	Reference    *string
	ContractDate *time.Time
	NetValue     float64
	Notes        *string
	CreatedBy    string
	CreatedAt    time.Time
}

type CreateContractParams struct {
	CompanyID    string
	ProjectID    string
	ContractType string
	Reference    *string
	ContractDate *time.Time
	NetValue     float64
	Notes        *string
	CreatedBy    string
}

type UpdateContractParams struct {
	CompanyID    string
	ContractID   string
	ContractType string
	Reference    *string
	ContractDate *time.Time
	NetValue     float64
	Notes        *string
}

type ProjectInvoiceRow struct {
	ID               string
	ProjectID        string
	ProjectName      string
	InvoiceNumber    string
	InvoiceDate      time.Time
	DueDate          *time.Time
	NetAmount        float64
	VatAmount        float64
	GrossAmount      float64
	PaidAmount       float64
	Status           string
	CancelsInvoiceID *string
	Notes            *string
	CreatedBy        string
	CreatedAt        time.Time
}

type CreateProjectInvoiceParams struct {
	CompanyID     string
	ProjectID     string
	InvoiceNumber string
	InvoiceDate   time.Time
	DueDate       *time.Time
	NetAmount     float64
	VatAmount     float64
	Notes         *string
	CreatedBy     string
}

type InvoicePaymentRow struct {
	ID          string
	InvoiceID   string
	PaymentDate time.Time
	Amount      float64
	Notes       *string
	CreatedBy   string
	CreatedAt   time.Time
}

type AddPaymentParams struct {
	CompanyID   string
	InvoiceID   string
	PaymentDate time.Time
	Amount      float64
	Notes       *string
	CreatedBy   string
}

type ProjectRevenueSummaryRow struct {
	ProjectID     string
	ProjectName   string
	ContractValue float64
	InvoicedNet   float64
	InvoicedGross float64
	Collected     float64
	Outstanding   float64
	InvoiceCount  int
	HasOverdue    bool
}

type MonthlyRevenueTrendRow struct {
	Month         int
	InvoicedNet   float64
	InvoicedGross float64
	Collected     float64
}

// ── Repository ────────────────────────────────────────────────────────────────

type RevenueRepository struct {
	db *pgxpool.Pool
}

func NewRevenueRepository(db *pgxpool.Pool) *RevenueRepository {
	return &RevenueRepository{db: db}
}

// ── Contracts ─────────────────────────────────────────────────────────────────

func (r *RevenueRepository) CreateContract(ctx context.Context, p CreateContractParams) (*ProjectContractRow, error) {
	var row ProjectContractRow
	err := r.db.QueryRow(ctx, `
		INSERT INTO project_contracts
			(company_id, project_id, contract_type, reference, contract_date, net_value, notes, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8::uuid)
		RETURNING id::text, project_id::text, ''::text, contract_type, reference, contract_date, net_value, notes, created_by::text, created_at`,
		p.CompanyID, p.ProjectID, p.ContractType, p.Reference, p.ContractDate, p.NetValue, p.Notes, p.CreatedBy,
	).Scan(&row.ID, &row.ProjectID, &row.ProjectName, &row.ContractType, &row.Reference, &row.ContractDate, &row.NetValue, &row.Notes, &row.CreatedBy, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("revenue.CreateContract: %w", err)
	}
	return &row, nil
}

func (r *RevenueRepository) GetContract(ctx context.Context, companyID, contractID string) (*ProjectContractRow, error) {
	var row ProjectContractRow
	err := r.db.QueryRow(ctx, `
		SELECT pc.id::text, pc.project_id::text, p.name, pc.contract_type, pc.reference,
		       pc.contract_date, pc.net_value, pc.notes, pc.created_by::text, pc.created_at
		FROM project_contracts pc
		JOIN projects p ON p.id = pc.project_id
		WHERE pc.id = $1::uuid AND pc.company_id = $2::uuid`,
		contractID, companyID,
	).Scan(&row.ID, &row.ProjectID, &row.ProjectName, &row.ContractType, &row.Reference, &row.ContractDate, &row.NetValue, &row.Notes, &row.CreatedBy, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetContract: %w", err)
	}
	return &row, nil
}

func (r *RevenueRepository) ListContractsForProject(ctx context.Context, companyID, projectID string) ([]ProjectContractRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pc.id::text, pc.project_id::text, p.name, pc.contract_type, pc.reference,
		       pc.contract_date, pc.net_value, pc.notes, pc.created_by::text, pc.created_at
		FROM project_contracts pc
		JOIN projects p ON p.id = pc.project_id
		WHERE pc.company_id = $1::uuid AND pc.project_id = $2::uuid
		ORDER BY pc.contract_date ASC NULLS LAST, pc.created_at ASC`,
		companyID, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("revenue.ListContractsForProject: %w", err)
	}
	defer rows.Close()
	return scanContracts(rows)
}

func (r *RevenueRepository) UpdateContract(ctx context.Context, p UpdateContractParams) (*ProjectContractRow, error) {
	var row ProjectContractRow
	err := r.db.QueryRow(ctx, `
		UPDATE project_contracts
		SET contract_type = $3, reference = $4, contract_date = $5, net_value = $6, notes = $7, updated_at = NOW()
		WHERE id = $1::uuid AND company_id = $2::uuid
		RETURNING id::text, project_id::text, ''::text, contract_type, reference, contract_date, net_value, notes, created_by::text, created_at`,
		p.ContractID, p.CompanyID, p.ContractType, p.Reference, p.ContractDate, p.NetValue, p.Notes,
	).Scan(&row.ID, &row.ProjectID, &row.ProjectName, &row.ContractType, &row.Reference, &row.ContractDate, &row.NetValue, &row.Notes, &row.CreatedBy, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("revenue.UpdateContract: %w", err)
	}
	return &row, nil
}

func (r *RevenueRepository) DeleteContract(ctx context.Context, companyID, contractID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM project_contracts WHERE id = $1::uuid AND company_id = $2::uuid`,
		contractID, companyID,
	)
	if err != nil {
		return fmt.Errorf("revenue.DeleteContract: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func scanContracts(rows pgx.Rows) ([]ProjectContractRow, error) {
	var out []ProjectContractRow
	for rows.Next() {
		var row ProjectContractRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.ProjectName, &row.ContractType, &row.Reference, &row.ContractDate, &row.NetValue, &row.Notes, &row.CreatedBy, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("revenue.scanContracts: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ── Invoices ──────────────────────────────────────────────────────────────────

const invoiceSelectCols = `
	pi.id::text, pi.project_id::text, p.name, pi.invoice_number,
	pi.invoice_date, pi.due_date, pi.net_amount, pi.vat_amount, pi.gross_amount,
	COALESCE((SELECT SUM(amount) FROM project_invoice_payments WHERE invoice_id = pi.id AND company_id = pi.company_id), 0) AS paid_amount,
	pi.status, pi.cancels_invoice_id::text, pi.notes, pi.created_by::text, pi.created_at`

func scanInvoiceRow(row pgx.Row) (*ProjectInvoiceRow, error) {
	var inv ProjectInvoiceRow
	if err := row.Scan(
		&inv.ID, &inv.ProjectID, &inv.ProjectName, &inv.InvoiceNumber,
		&inv.InvoiceDate, &inv.DueDate, &inv.NetAmount, &inv.VatAmount, &inv.GrossAmount,
		&inv.PaidAmount, &inv.Status, &inv.CancelsInvoiceID, &inv.Notes, &inv.CreatedBy, &inv.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &inv, nil
}

func scanInvoiceRows(rows pgx.Rows) ([]ProjectInvoiceRow, error) {
	var out []ProjectInvoiceRow
	for rows.Next() {
		var inv ProjectInvoiceRow
		if err := rows.Scan(
			&inv.ID, &inv.ProjectID, &inv.ProjectName, &inv.InvoiceNumber,
			&inv.InvoiceDate, &inv.DueDate, &inv.NetAmount, &inv.VatAmount, &inv.GrossAmount,
			&inv.PaidAmount, &inv.Status, &inv.CancelsInvoiceID, &inv.Notes, &inv.CreatedBy, &inv.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("revenue.scanInvoiceRows: %w", err)
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *RevenueRepository) CreateInvoice(ctx context.Context, p CreateProjectInvoiceParams) (*ProjectInvoiceRow, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO project_invoices
			(company_id, project_id, invoice_number, invoice_date, due_date, net_amount, vat_amount, notes, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9::uuid)
		RETURNING id::text, project_id::text, ''::text, invoice_number,
		          invoice_date, due_date, net_amount, vat_amount, gross_amount,
		          0::numeric, status, cancels_invoice_id::text, notes, created_by::text, created_at`,
		p.CompanyID, p.ProjectID, p.InvoiceNumber, p.InvoiceDate, p.DueDate,
		p.NetAmount, p.VatAmount, p.Notes, p.CreatedBy,
	)
	inv, err := scanInvoiceRow(row)
	if err != nil {
		return nil, fmt.Errorf("revenue.CreateInvoice: %w", err)
	}
	return inv, nil
}

func (r *RevenueRepository) GetInvoice(ctx context.Context, companyID, invoiceID string) (*ProjectInvoiceRow, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM project_invoices pi
		JOIN projects p ON p.id = pi.project_id
		WHERE pi.id = $1::uuid AND pi.company_id = $2::uuid`, invoiceSelectCols)
	row := r.db.QueryRow(ctx, q, invoiceID, companyID)
	inv, err := scanInvoiceRow(row)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetInvoice: %w", err)
	}
	return inv, nil
}

func (r *RevenueRepository) ListInvoices(ctx context.Context, companyID string, projectID, status *string, year, month int) ([]ProjectInvoiceRow, error) {
	args := []interface{}{companyID}
	clauses := []string{"pi.company_id = $1::uuid"}
	n := 2

	if projectID != nil {
		clauses = append(clauses, fmt.Sprintf("pi.project_id = $%d::uuid", n))
		args = append(args, *projectID)
		n++
	}
	if status != nil {
		clauses = append(clauses, fmt.Sprintf("pi.status = $%d", n))
		args = append(args, *status)
		n++
	}
	if year > 0 {
		clauses = append(clauses, fmt.Sprintf("EXTRACT(YEAR FROM pi.invoice_date)::int = $%d", n))
		args = append(args, year)
		n++
	}
	if month > 0 {
		clauses = append(clauses, fmt.Sprintf("EXTRACT(MONTH FROM pi.invoice_date)::int = $%d", n))
		args = append(args, month)
		n++
	}
	_ = n

	q := fmt.Sprintf(`
		SELECT %s
		FROM project_invoices pi
		JOIN projects p ON p.id = pi.project_id
		WHERE %s
		ORDER BY pi.invoice_date DESC, pi.created_at DESC`,
		invoiceSelectCols, strings.Join(clauses, " AND "))

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("revenue.ListInvoices: %w", err)
	}
	defer rows.Close()
	out, err := scanInvoiceRows(rows)
	if err != nil {
		return nil, fmt.Errorf("revenue.ListInvoices: %w", err)
	}
	return out, nil
}

func (r *RevenueRepository) CancelInvoice(ctx context.Context, companyID, invoiceID string, notes *string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE project_invoices
		SET status = 'cancelled',
		    notes = CASE WHEN $3::text IS NOT NULL THEN COALESCE(notes || E'\n', '') || $3::text ELSE notes END,
		    updated_at = NOW()
		WHERE id = $1::uuid AND company_id = $2::uuid AND status != 'cancelled'`,
		invoiceID, companyID, notes,
	)
	if err != nil {
		return fmt.Errorf("revenue.CancelInvoice: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ── Payments ──────────────────────────────────────────────────────────────────

func (r *RevenueRepository) AddPayment(ctx context.Context, p AddPaymentParams) (*InvoicePaymentRow, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("revenue.AddPayment begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Verify invoice exists, belongs to company, and isn't cancelled.
	var status string
	if err := tx.QueryRow(ctx, `
		SELECT status FROM project_invoices
		WHERE id = $1::uuid AND company_id = $2::uuid`,
		p.InvoiceID, p.CompanyID,
	).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("revenue.AddPayment check: %w", err)
	}
	if status == "cancelled" {
		return nil, fmt.Errorf("račun je otkazan")
	}

	var pay InvoicePaymentRow
	if err := tx.QueryRow(ctx, `
		INSERT INTO project_invoice_payments (company_id, invoice_id, payment_date, amount, notes, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid)
		RETURNING id::text, invoice_id::text, payment_date, amount, notes, created_by::text, created_at`,
		p.CompanyID, p.InvoiceID, p.PaymentDate, p.Amount, p.Notes, p.CreatedBy,
	).Scan(&pay.ID, &pay.InvoiceID, &pay.PaymentDate, &pay.Amount, &pay.Notes, &pay.CreatedBy, &pay.CreatedAt); err != nil {
		return nil, fmt.Errorf("revenue.AddPayment insert: %w", err)
	}

	// Update invoice status based on total paid vs gross_amount.
	if _, err := tx.Exec(ctx, `
		UPDATE project_invoices
		SET status = CASE
		    WHEN (SELECT COALESCE(SUM(amount), 0) FROM project_invoice_payments WHERE invoice_id = $1::uuid) >= gross_amount
		    THEN 'paid'
		    ELSE 'partial'
		END,
		updated_at = NOW()
		WHERE id = $1::uuid AND company_id = $2::uuid AND status != 'cancelled'`,
		p.InvoiceID, p.CompanyID,
	); err != nil {
		return nil, fmt.Errorf("revenue.AddPayment update status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("revenue.AddPayment commit: %w", err)
	}
	return &pay, nil
}

func (r *RevenueRepository) ListPayments(ctx context.Context, companyID, invoiceID string) ([]InvoicePaymentRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pip.id::text, pip.invoice_id::text, pip.payment_date, pip.amount, pip.notes, pip.created_by::text, pip.created_at
		FROM project_invoice_payments pip
		WHERE pip.invoice_id = $1::uuid AND pip.company_id = $2::uuid
		ORDER BY pip.payment_date DESC, pip.created_at DESC`,
		invoiceID, companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("revenue.ListPayments: %w", err)
	}
	defer rows.Close()

	var out []InvoicePaymentRow
	for rows.Next() {
		var pay InvoicePaymentRow
		if err := rows.Scan(&pay.ID, &pay.InvoiceID, &pay.PaymentDate, &pay.Amount, &pay.Notes, &pay.CreatedBy, &pay.CreatedAt); err != nil {
			return nil, fmt.Errorf("revenue.ListPayments scan: %w", err)
		}
		out = append(out, pay)
	}
	return out, rows.Err()
}

// ── Analytics ─────────────────────────────────────────────────────────────────

func (r *RevenueRepository) GetProjectRevenueSummaries(ctx context.Context, companyID string, year, month int) ([]ProjectRevenueSummaryRow, error) {
	q := `
		WITH invoice_agg AS (
		    SELECT
		        pi.project_id,
		        COALESCE(SUM(CASE WHEN pi.status != 'cancelled' THEN pi.net_amount   ELSE 0 END), 0) AS invoiced_net,
		        COALESCE(SUM(CASE WHEN pi.status != 'cancelled' THEN pi.gross_amount ELSE 0 END), 0) AS invoiced_gross,
		        COUNT(CASE WHEN pi.status != 'cancelled' THEN 1 END)::int AS invoice_count,
		        COALESCE(bool_or(pi.status IN ('issued', 'partial') AND pi.due_date IS NOT NULL AND pi.due_date < NOW()::date), false) AS has_overdue
		    FROM project_invoices pi
		    WHERE pi.company_id = $1::uuid
		      AND EXTRACT(YEAR FROM pi.invoice_date)::int = $2
		      AND ($3::int = 0 OR EXTRACT(MONTH FROM pi.invoice_date)::int = $3::int)
		    GROUP BY pi.project_id
		),
		payment_agg AS (
		    SELECT pi.project_id, COALESCE(SUM(pip.amount), 0) AS collected
		    FROM project_invoice_payments pip
		    JOIN project_invoices pi ON pi.id = pip.invoice_id
		    WHERE pip.company_id = $1::uuid
		      AND pi.status != 'cancelled'
		      AND EXTRACT(YEAR FROM pip.payment_date)::int = $2
		      AND ($3::int = 0 OR EXTRACT(MONTH FROM pip.payment_date)::int = $3::int)
		    GROUP BY pi.project_id
		),
		contract_agg AS (
		    SELECT project_id, COALESCE(SUM(net_value), 0) AS contract_value
		    FROM project_contracts
		    WHERE company_id = $1::uuid
		    GROUP BY project_id
		)
		SELECT
		    p.id::text,
		    p.name,
		    COALESCE(ca.contract_value, 0),
		    COALESCE(ia.invoiced_net, 0),
		    COALESCE(ia.invoiced_gross, 0),
		    COALESCE(pa.collected, 0),
		    COALESCE(ia.invoiced_gross, 0) - COALESCE(pa.collected, 0),
		    COALESCE(ia.invoice_count, 0),
		    COALESCE(ia.has_overdue, false)
		FROM projects p
		LEFT JOIN invoice_agg ia ON ia.project_id = p.id
		LEFT JOIN payment_agg pa ON pa.project_id = p.id
		LEFT JOIN contract_agg ca ON ca.project_id = p.id
		WHERE p.company_id = $1::uuid
		  AND (ia.project_id IS NOT NULL OR ca.project_id IS NOT NULL)
		ORDER BY p.name`

	rows, err := r.db.Query(ctx, q, companyID, year, month)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetProjectRevenueSummaries: %w", err)
	}
	defer rows.Close()

	var out []ProjectRevenueSummaryRow
	for rows.Next() {
		var row ProjectRevenueSummaryRow
		if err := rows.Scan(
			&row.ProjectID, &row.ProjectName,
			&row.ContractValue, &row.InvoicedNet, &row.InvoicedGross,
			&row.Collected, &row.Outstanding,
			&row.InvoiceCount, &row.HasOverdue,
		); err != nil {
			return nil, fmt.Errorf("revenue.GetProjectRevenueSummaries scan: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RevenueRepository) GetCompanyMonthlyRevenueTrend(ctx context.Context, companyID string, year int) ([]MonthlyRevenueTrendRow, error) {
	rows, err := r.db.Query(ctx, `
		WITH invoiced AS (
		    SELECT EXTRACT(MONTH FROM invoice_date)::int AS month,
		           COALESCE(SUM(CASE WHEN status != 'cancelled' THEN net_amount   ELSE 0 END), 0) AS inv_net,
		           COALESCE(SUM(CASE WHEN status != 'cancelled' THEN gross_amount ELSE 0 END), 0) AS inv_gross
		    FROM project_invoices
		    WHERE company_id = $1::uuid AND EXTRACT(YEAR FROM invoice_date)::int = $2
		    GROUP BY month
		),
		collected AS (
		    SELECT EXTRACT(MONTH FROM pip.payment_date)::int AS month,
		           COALESCE(SUM(pip.amount), 0) AS coll
		    FROM project_invoice_payments pip
		    JOIN project_invoices pi ON pi.id = pip.invoice_id
		    WHERE pip.company_id = $1::uuid
		      AND pi.status != 'cancelled'
		      AND EXTRACT(YEAR FROM pip.payment_date)::int = $2
		    GROUP BY month
		)
		SELECT m, COALESCE(i.inv_net, 0), COALESCE(i.inv_gross, 0), COALESCE(c.coll, 0)
		FROM generate_series(1, 12) m
		LEFT JOIN invoiced i ON i.month = m
		LEFT JOIN collected c ON c.month = m
		ORDER BY m`,
		companyID, year,
	)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetCompanyMonthlyRevenueTrend: %w", err)
	}
	defer rows.Close()

	var out []MonthlyRevenueTrendRow
	for rows.Next() {
		var row MonthlyRevenueTrendRow
		if err := rows.Scan(&row.Month, &row.InvoicedNet, &row.InvoicedGross, &row.Collected); err != nil {
			return nil, fmt.Errorf("revenue.GetCompanyMonthlyRevenueTrend scan: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RevenueRepository) GetProjectMonthlyRevenueTrend(ctx context.Context, companyID, projectID string, year int) ([]MonthlyRevenueTrendRow, error) {
	rows, err := r.db.Query(ctx, `
		WITH invoiced AS (
		    SELECT EXTRACT(MONTH FROM invoice_date)::int AS month,
		           COALESCE(SUM(CASE WHEN status != 'cancelled' THEN net_amount   ELSE 0 END), 0) AS inv_net,
		           COALESCE(SUM(CASE WHEN status != 'cancelled' THEN gross_amount ELSE 0 END), 0) AS inv_gross
		    FROM project_invoices
		    WHERE company_id = $1::uuid AND project_id = $2::uuid AND EXTRACT(YEAR FROM invoice_date)::int = $3
		    GROUP BY month
		),
		collected AS (
		    SELECT EXTRACT(MONTH FROM pip.payment_date)::int AS month,
		           COALESCE(SUM(pip.amount), 0) AS coll
		    FROM project_invoice_payments pip
		    JOIN project_invoices pi ON pi.id = pip.invoice_id
		    WHERE pip.company_id = $1::uuid
		      AND pi.project_id = $2::uuid
		      AND pi.status != 'cancelled'
		      AND EXTRACT(YEAR FROM pip.payment_date)::int = $3
		    GROUP BY month
		)
		SELECT m, COALESCE(i.inv_net, 0), COALESCE(i.inv_gross, 0), COALESCE(c.coll, 0)
		FROM generate_series(1, 12) m
		LEFT JOIN invoiced i ON i.month = m
		LEFT JOIN collected c ON c.month = m
		ORDER BY m`,
		companyID, projectID, year,
	)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetProjectMonthlyRevenueTrend: %w", err)
	}
	defer rows.Close()

	var out []MonthlyRevenueTrendRow
	for rows.Next() {
		var row MonthlyRevenueTrendRow
		if err := rows.Scan(&row.Month, &row.InvoicedNet, &row.InvoicedGross, &row.Collected); err != nil {
			return nil, fmt.Errorf("revenue.GetProjectMonthlyRevenueTrend scan: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *RevenueRepository) GetContractTotalForProject(ctx context.Context, companyID, projectID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(net_value), 0) FROM project_contracts
		WHERE company_id = $1::uuid AND project_id = $2::uuid`,
		companyID, projectID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("revenue.GetContractTotalForProject: %w", err)
	}
	return total, nil
}
