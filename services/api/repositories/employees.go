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

var ErrNotFound = errors.New("not found")

// HardDeleteBlockedError is returned when a hard delete is rejected due to existing history.
type HardDeleteBlockedError struct{ Reason string }

func (e *HardDeleteBlockedError) Error() string { return e.Reason }

// Employee is the internal DB model used within the repositories package.
type Employee struct {
	ID           string
	CompanyID    string
	FirstName    string
	LastName     string
	Role         string
	Email        *string
	Phone        *string
	Active       bool
	SupervisorID *string
	// Populated only by GetByID (via LEFT JOIN)
	SupervisorFirstName *string
	SupervisorLastName  *string
	SupervisorRole      *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type EmployeeListFilter struct {
	Search string  // substring match on first_name, last_name, email
	Role   *string // nil = no filter
	Active *bool   // nil = no filter
}

type EmployeeRepository struct {
	db *pgxpool.Pool
}

func NewEmployeeRepository(db *pgxpool.Pool) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

func (r *EmployeeRepository) List(ctx context.Context, companyID string, f EmployeeListFilter) ([]Employee, error) {
	args := []interface{}{companyID}
	conditions := []string{"e.company_id = $1::uuid"}

	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		n := len(args)
		conditions = append(conditions, fmt.Sprintf(
			"(e.first_name ILIKE $%d OR e.last_name ILIKE $%d OR e.email ILIKE $%d)",
			n, n, n,
		))
	}

	if f.Role != nil {
		args = append(args, *f.Role)
		conditions = append(conditions, fmt.Sprintf("e.role = $%d", len(args)))
	}

	if f.Active != nil {
		args = append(args, *f.Active)
		conditions = append(conditions, fmt.Sprintf("e.active = $%d", len(args)))
	}

	query := `
		SELECT
			e.id::text,
			e.company_id::text,
			e.first_name,
			e.last_name,
			e.role,
			e.email,
			e.phone,
			e.active,
			e.supervisor_id::text,
			e.created_at,
			e.updated_at
		FROM employees e
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY e.last_name ASC, e.first_name ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("employees.List query: %w", err)
	}
	defer rows.Close()

	var result []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(
			&e.ID, &e.CompanyID,
			&e.FirstName, &e.LastName, &e.Role,
			&e.Email, &e.Phone, &e.Active,
			&e.SupervisorID,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("employees.List scan: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *EmployeeRepository) GetByID(ctx context.Context, companyID, id string) (*Employee, error) {
	var e Employee
	err := r.db.QueryRow(ctx, `
		SELECT
			e.id::text,
			e.company_id::text,
			e.first_name,
			e.last_name,
			e.role,
			e.email,
			e.phone,
			e.active,
			e.supervisor_id::text,
			s.first_name,
			s.last_name,
			s.role,
			e.created_at,
			e.updated_at
		FROM employees e
		LEFT JOIN employees s ON s.id = e.supervisor_id AND s.company_id = e.company_id
		WHERE e.id = $1::uuid AND e.company_id = $2::uuid
	`, id, companyID).Scan(
		&e.ID, &e.CompanyID,
		&e.FirstName, &e.LastName, &e.Role,
		&e.Email, &e.Phone, &e.Active,
		&e.SupervisorID,
		&e.SupervisorFirstName, &e.SupervisorLastName, &e.SupervisorRole,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("employees.GetByID: %w", err)
	}
	return &e, nil
}

// CreateWithTx inserts an employee row inside an existing transaction.
func (r *EmployeeRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, companyID, firstName, lastName, role string, email, phone, supervisorID *string) (*Employee, error) {
	var e Employee
	err := tx.QueryRow(ctx, `
		INSERT INTO employees (company_id, first_name, last_name, role, email, phone, supervisor_id)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7::uuid)
		RETURNING
			id::text, company_id::text,
			first_name, last_name, role,
			email, phone, active,
			supervisor_id::text,
			created_at, updated_at
	`, companyID, firstName, lastName, role, email, phone, supervisorID).Scan(
		&e.ID, &e.CompanyID,
		&e.FirstName, &e.LastName, &e.Role,
		&e.Email, &e.Phone, &e.Active,
		&e.SupervisorID,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("employees.CreateWithTx: %w", err)
	}
	return &e, nil
}

func (r *EmployeeRepository) Create(ctx context.Context, companyID, firstName, lastName, role string, email, phone, supervisorID *string) (*Employee, error) {
	var e Employee
	err := r.db.QueryRow(ctx, `
		INSERT INTO employees (company_id, first_name, last_name, role, email, phone, supervisor_id)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7::uuid)
		RETURNING
			id::text, company_id::text,
			first_name, last_name, role,
			email, phone, active,
			supervisor_id::text,
			created_at, updated_at
	`, companyID, firstName, lastName, role, email, phone, supervisorID).Scan(
		&e.ID, &e.CompanyID,
		&e.FirstName, &e.LastName, &e.Role,
		&e.Email, &e.Phone, &e.Active,
		&e.SupervisorID,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("employees.Create: %w", err)
	}
	return &e, nil
}

func (r *EmployeeRepository) Update(ctx context.Context, companyID, id, firstName, lastName, role string, email, phone, supervisorID *string) (*Employee, error) {
	var e Employee
	err := r.db.QueryRow(ctx, `
		UPDATE employees
		SET first_name = $3, last_name = $4, role = $5,
		    email = $6, phone = $7, supervisor_id = $8::uuid
		WHERE id = $1::uuid AND company_id = $2::uuid
		RETURNING
			id::text, company_id::text,
			first_name, last_name, role,
			email, phone, active,
			supervisor_id::text,
			created_at, updated_at
	`, id, companyID, firstName, lastName, role, email, phone, supervisorID).Scan(
		&e.ID, &e.CompanyID,
		&e.FirstName, &e.LastName, &e.Role,
		&e.Email, &e.Phone, &e.Active,
		&e.SupervisorID,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("employees.Update: %w", err)
	}
	return &e, nil
}

func (r *EmployeeRepository) SetActive(ctx context.Context, companyID, id string, active bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE employees SET active = $3
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, id, companyID, active)
	if err != nil {
		return fmt.Errorf("employees.SetActive: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetRoleByID returns the role of an employee that belongs to the given company.
// Used by the service to validate supervisor assignments.
func (r *EmployeeRepository) GetRoleByID(ctx context.Context, companyID, id string) (string, error) {
	var role string
	err := r.db.QueryRow(ctx, `
		SELECT role FROM employees WHERE id = $1::uuid AND company_id = $2::uuid
	`, id, companyID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("employees.GetRoleByID: %w", err)
	}
	return role, nil
}

// ── Hard delete ───────────────────────────────────────────────────────────────

// GetLinkedUserID returns the users.id linked to an employee, or ("", false, nil) if none.
func (r *EmployeeRepository) GetLinkedUserID(ctx context.Context, companyID, employeeID string) (string, bool, error) {
	var userID string
	err := r.db.QueryRow(ctx, `
		SELECT id::text FROM users
		WHERE employee_id = $1::uuid AND company_id = $2::uuid
		LIMIT 1
	`, employeeID, companyID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("employees.GetLinkedUserID: %w", err)
	}
	return userID, true, nil
}

// CountActiveDirectors counts active direktor employees who have an active user account.
func (r *EmployeeRepository) CountActiveDirectors(ctx context.Context, companyID string) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM employees e
		JOIN users u ON u.employee_id = e.id AND u.company_id = e.company_id AND u.active = true
		WHERE e.company_id = $1::uuid
		  AND e.role       = 'direktor'
		  AND e.active     = true
	`, companyID).Scan(&count)
	return count, err
}

// HardDeleteCheck returns a HardDeleteBlockedError if the employee has any meaningful history.
// A nil return means it is safe to proceed with hard delete.
func (r *EmployeeRepository) HardDeleteCheck(ctx context.Context, companyID, employeeID string) error {
	var (
		dailyReports        int64
		reportWorkerHours   int64
		workerDailyHours    int64
		projectAssignments  int64
		materialPurchases   int64
		materialResponsib   int64
		assetTransfers      int64
		assets              int64
		supervisedEmployees int64
	)
	err := r.db.QueryRow(ctx, `
		SELECT
		    (SELECT COUNT(*) FROM daily_reports                    WHERE poslovoda_id   = $1::uuid AND company_id = $2::uuid),
		    (SELECT COUNT(*) FROM daily_report_worker_hours        WHERE worker_id      = $1::uuid AND company_id = $2::uuid),
		    (SELECT COUNT(*) FROM worker_daily_hours               WHERE worker_id      = $1::uuid AND company_id = $2::uuid),
		    (SELECT COUNT(*) FROM project_assignments              WHERE employee_id    = $1::uuid AND company_id = $2::uuid),
		    (SELECT COUNT(*) FROM material_purchase_sessions       WHERE buyer_id       = $1::uuid AND company_id = $2::uuid),
		    (SELECT COUNT(*) FROM employee_material_responsibility WHERE employee_id    = $1::uuid AND company_id = $2::uuid),
		    (SELECT COUNT(*) FROM asset_transfers                  WHERE (from_employee_id = $1::uuid OR to_employee_id = $1::uuid) AND company_id = $2::uuid),
		    (SELECT COUNT(*) FROM employee_assets                  WHERE employee_id    = $1::uuid AND company_id = $2::uuid),
		    (SELECT COUNT(*) FROM employees                        WHERE supervisor_id  = $1::uuid AND company_id = $2::uuid)
	`, employeeID, companyID).Scan(
		&dailyReports, &reportWorkerHours, &workerDailyHours,
		&projectAssignments, &materialPurchases, &materialResponsib,
		&assetTransfers, &assets, &supervisedEmployees,
	)
	if err != nil {
		return fmt.Errorf("employees.HardDeleteCheck: %w", err)
	}

	switch {
	case dailyReports > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik je poslovođa na %d dnevnih izvještaja. Koristite deaktivaciju.", dailyReports)}
	case reportWorkerHours > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik ima evidentiranih sati u %d dnevnih izvještaja. Koristite deaktivaciju.", reportWorkerHours)}
	case workerDailyHours > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik ima %d zapisa vlastitih radnih sati. Koristite deaktivaciju.", workerDailyHours)}
	case projectAssignments > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik je dodijeljen na %d projekt(a). Koristite deaktivaciju.", projectAssignments)}
	case materialPurchases > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik je evidentiran kao kupac materijala u %d zapisa. Koristite deaktivaciju.", materialPurchases)}
	case materialResponsib > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik ima %d zapisa odgovornosti za materijal. Koristite deaktivaciju.", materialResponsib)}
	case assetTransfers > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik je uključen u %d prijenosa imovine/materijala. Koristite deaktivaciju.", assetTransfers)}
	case assets > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik ima %d dodijeljenih sredstava (alati, vozila). Koristite deaktivaciju.", assets)}
	case supervisedEmployees > 0:
		return &HardDeleteBlockedError{Reason: fmt.Sprintf("Trajno brisanje nije moguće: zaposlenik je nadzornik za %d zaposlenik(a). Najprije promijenite nadzornika. Koristite deaktivaciju.", supervisedEmployees)}
	}
	return nil
}

// HardDeleteWithTx permanently removes an employee row within the given transaction.
// Call only after HardDeleteCheck passes.
func (r *EmployeeRepository) HardDeleteWithTx(ctx context.Context, tx pgx.Tx, companyID, employeeID string) error {
	tag, err := tx.Exec(ctx, `
		DELETE FROM employees WHERE id = $1::uuid AND company_id = $2::uuid
	`, employeeID, companyID)
	if err != nil {
		return fmt.Errorf("employees.HardDelete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
