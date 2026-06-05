package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

var ErrInsufficientQuantity = errors.New("insufficient quantity for this operation")

type InventoryRepository struct {
	db *pgxpool.Pool
}

func NewInventoryRepository(db *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// ── Employee summary ───────────────────────────────────────────────────────────

func (r *InventoryRepository) GetEmployee(ctx context.Context, companyID, employeeID string) (*dto.InventoryEmployeeSummary, error) {
	var s dto.InventoryEmployeeSummary
	err := r.db.QueryRow(ctx,
		`SELECT id, first_name, last_name, role
		 FROM employees
		 WHERE id = $1::uuid AND company_id = $2::uuid AND active = true`,
		employeeID, companyID,
	).Scan(&s.ID, &s.FirstName, &s.LastName, &s.Role)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get employee: %w", err)
	}
	return &s, nil
}

// ── Assets ────────────────────────────────────────────────────────────────────

func (r *InventoryRepository) GetAssets(ctx context.Context, companyID, employeeID string) ([]dto.InventoryAssetItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, asset_type, name, quantity, unit, serial_number, notes, assigned_at
		 FROM employee_assets
		 WHERE company_id = $1::uuid AND employee_id = $2::uuid AND active = true
		 ORDER BY assigned_at DESC`,
		companyID, employeeID,
	)
	if err != nil {
		return nil, fmt.Errorf("get assets: %w", err)
	}
	defer rows.Close()

	var items []dto.InventoryAssetItem
	for rows.Next() {
		var a dto.InventoryAssetItem
		if err := rows.Scan(&a.ID, &a.AssetType, &a.Name, &a.Quantity, &a.Unit, &a.SerialNumber, &a.Notes, &a.AssignedAt); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// ── Materials ─────────────────────────────────────────────────────────────────

func (r *InventoryRepository) GetMaterials(ctx context.Context, companyID, employeeID string) ([]dto.InventoryMaterialItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
			emr.project_material_id,
			pm.material_name,
			pm.material_code,
			emr.project_id,
			p.name  AS project_name,
			SUM(emr.quantity) AS total_quantity,
			emr.unit
		 FROM employee_material_responsibility emr
		 JOIN project_materials pm ON pm.id = emr.project_material_id
		 JOIN projects p ON p.id = emr.project_id
		 WHERE emr.company_id = $1::uuid AND emr.employee_id = $2::uuid AND emr.active = true
		 GROUP BY emr.project_material_id, pm.material_name, pm.material_code,
		          emr.project_id, p.name, emr.unit
		 ORDER BY p.name, pm.material_name`,
		companyID, employeeID,
	)
	if err != nil {
		return nil, fmt.Errorf("get materials: %w", err)
	}
	defer rows.Close()

	var items []dto.InventoryMaterialItem
	for rows.Next() {
		var m dto.InventoryMaterialItem
		if err := rows.Scan(&m.ProjectMaterialID, &m.MaterialName, &m.MaterialCode,
			&m.ProjectID, &m.ProjectName, &m.TotalQuantity, &m.Unit); err != nil {
			return nil, fmt.Errorf("scan material: %w", err)
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// ── Transfer targets ───────────────────────────────────────────────────────────

func (r *InventoryRepository) GetTransferTargets(ctx context.Context, companyID, callerEmployeeID, callerRole, transferType string) ([]dto.TransferTarget, error) {
	var query string
	var args []interface{}

	if callerRole == "poslovoda" && transferType == "material" {
		// Material transfers: any active poslovoda in the company except self
		query = `SELECT id, first_name, last_name, role
				 FROM employees
				 WHERE company_id = $1::uuid
				   AND id <> $2::uuid
				   AND role = 'poslovoda'
				   AND active = true
				 ORDER BY last_name, first_name`
		args = []interface{}{companyID, callerEmployeeID}
	} else if callerRole == "poslovoda" {
		// Asset transfers: only directly supervised radnici
		query = `SELECT id, first_name, last_name, role
				 FROM employees
				 WHERE company_id = $1::uuid
				   AND supervisor_id = $2::uuid
				   AND active = true
				 ORDER BY last_name, first_name`
		args = []interface{}{companyID, callerEmployeeID}
	} else {
		// direktor, inzenjer, administracija: any active employee in company except self
		query = `SELECT id, first_name, last_name, role
				 FROM employees
				 WHERE company_id = $1::uuid
				   AND id <> $2::uuid
				   AND active = true
				 ORDER BY last_name, first_name`
		args = []interface{}{companyID, callerEmployeeID}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get transfer targets: %w", err)
	}
	defer rows.Close()

	var targets []dto.TransferTarget
	for rows.Next() {
		var t dto.TransferTarget
		if err := rows.Scan(&t.ID, &t.FirstName, &t.LastName, &t.Role); err != nil {
			return nil, fmt.Errorf("scan transfer target: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

// GetEmployeeActiveProjects returns projects with an active assignment for the given employee.
func (r *InventoryRepository) GetEmployeeActiveProjects(ctx context.Context, companyID, employeeID string) ([]dto.InventoryProject, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT p.id, p.name
		 FROM projects p
		 JOIN project_assignments pa ON pa.project_id = p.id AND pa.active = true
		 WHERE p.company_id = $1::uuid AND pa.employee_id = $2::uuid AND p.status = 'active'
		 ORDER BY p.name`,
		companyID, employeeID,
	)
	if err != nil {
		return nil, fmt.Errorf("get employee active projects: %w", err)
	}
	defer rows.Close()

	var projects []dto.InventoryProject
	for rows.Next() {
		var p dto.InventoryProject
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// ── Internal transfer helpers ─────────────────────────────────────────────────

// materialResponsibilityRow holds a single row from employee_material_responsibility.
type materialResponsibilityRow struct {
	ID        string
	ProjectID string
	Unit      string
	Quantity  float64
}

// projectMaterialInfo holds identifying metadata from a project_materials row.
type projectMaterialInfo struct {
	ID           string
	ProjectID    string
	MaterialName string
	MaterialCode *string
	Unit         string
}

// GetProjectMaterialInfo fetches name/code/unit for a project_material within a transaction.
func (r *InventoryRepository) GetProjectMaterialInfo(ctx context.Context, tx pgx.Tx, companyID, projectMaterialID string) (*projectMaterialInfo, error) {
	var m projectMaterialInfo
	err := tx.QueryRow(ctx,
		`SELECT id, project_id, material_name, material_code, unit
		 FROM project_materials
		 WHERE id = $1::uuid AND company_id = $2::uuid AND active = true`,
		projectMaterialID, companyID,
	).Scan(&m.ID, &m.ProjectID, &m.MaterialName, &m.MaterialCode, &m.Unit)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get project material info: %w", err)
	}
	return &m, nil
}

// FindOrCreateDestProjectMaterial ensures the destination project has a project_materials row
// for the transferred material. It matches first by material_code, then by normalized name+unit.
// If no match is found it inserts a new row with planned/used/available all zero.
// Returns the dest project_material_id to use for the responsibility row.
func (r *InventoryRepository) FindOrCreateDestProjectMaterial(
	ctx context.Context, tx pgx.Tx,
	companyID, destProjectID, materialName string,
	materialCode *string, unit string,
) (string, error) {
	var existingID string

	// Prefer match by material_code when available
	if materialCode != nil && *materialCode != "" {
		err := tx.QueryRow(ctx,
			`SELECT id FROM project_materials
			 WHERE project_id = $1::uuid AND company_id = $2::uuid
			   AND material_code = $3 AND active = true
			 LIMIT 1`,
			destProjectID, companyID, *materialCode,
		).Scan(&existingID)
		if err != nil && !isNoRows(err) {
			return "", fmt.Errorf("find dest material by code: %w", err)
		}
	}

	// Fall back to normalized name + unit match
	if existingID == "" {
		err := tx.QueryRow(ctx,
			`SELECT id FROM project_materials
			 WHERE project_id = $1::uuid AND company_id = $2::uuid
			   AND LOWER(TRIM(material_name)) = LOWER(TRIM($3))
			   AND unit = $4 AND active = true
			 LIMIT 1`,
			destProjectID, companyID, materialName, unit,
		).Scan(&existingID)
		if err != nil && !isNoRows(err) {
			return "", fmt.Errorf("find dest material by name: %w", err)
		}
	}

	if existingID != "" {
		return existingID, nil
	}

	// Insert new row. ON CONFLICT handles the rare case where a concurrent tx beat us
	// to the INSERT; DO UPDATE SET active=true reactivates any inactive row and returns id.
	var newID string
	err := tx.QueryRow(ctx,
		`INSERT INTO project_materials
		   (project_id, company_id, material_name, material_code,
		    planned_quantity, used_quantity, available_quantity, unit, source)
		 VALUES ($1::uuid, $2::uuid, $3, $4, 0, 0, 0, $5, 'transfer')
		 ON CONFLICT (project_id, company_id, LOWER(material_name), unit)
		   DO UPDATE SET active = true
		 RETURNING id`,
		destProjectID, companyID, materialName, materialCode, unit,
	).Scan(&newID)
	if err != nil {
		return "", fmt.Errorf("create dest project material: %w", err)
	}
	return newID, nil
}

// DeductProjectMaterialAvailable reduces available_quantity on a project_materials row.
// The WHERE guard prevents negative values; returns ErrInsufficientQuantity when it fires.
func (r *InventoryRepository) DeductProjectMaterialAvailable(ctx context.Context, tx pgx.Tx, companyID, projectMaterialID string, qty float64) error {
	tag, err := tx.Exec(ctx,
		`UPDATE project_materials
		 SET available_quantity = available_quantity - $1
		 WHERE id = $2::uuid AND company_id = $3::uuid AND available_quantity >= $1`,
		qty, projectMaterialID, companyID,
	)
	if err != nil {
		return fmt.Errorf("deduct project material available: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInsufficientQuantity
	}
	return nil
}

// AddProjectMaterialAvailable increases available_quantity on a project_materials row.
func (r *InventoryRepository) AddProjectMaterialAvailable(ctx context.Context, tx pgx.Tx, companyID, projectMaterialID string, qty float64) error {
	_, err := tx.Exec(ctx,
		`UPDATE project_materials
		 SET available_quantity = available_quantity + $1
		 WHERE id = $2::uuid AND company_id = $3::uuid`,
		qty, projectMaterialID, companyID,
	)
	return err
}

// LockAsset selects an employee_asset FOR UPDATE and verifies ownership.
// Returns asset_type for use in the transfer record.
func (r *InventoryRepository) LockAsset(ctx context.Context, tx pgx.Tx, companyID, assetID, fromEmployeeID string) (assetType string, err error) {
	err = tx.QueryRow(ctx,
		`SELECT asset_type FROM employee_assets
		 WHERE id = $1::uuid AND company_id = $2::uuid AND employee_id = $3::uuid AND active = true
		 FOR UPDATE`,
		assetID, companyID, fromEmployeeID,
	).Scan(&assetType)
	if err != nil {
		if isNoRows(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("lock asset: %w", err)
	}
	return assetType, nil
}

// AssignAsset updates the employee_id on an employee_assets row to the new owner.
func (r *InventoryRepository) AssignAsset(ctx context.Context, tx pgx.Tx, assetID, toEmployeeID string) error {
	_, err := tx.Exec(ctx,
		`UPDATE employee_assets SET employee_id = $1::uuid WHERE id = $2::uuid`,
		toEmployeeID, assetID,
	)
	return err
}

// LockMaterialRows selects all active responsibility rows for a given employee+material FOR UPDATE.
// Returns the rows and total available quantity.
func (r *InventoryRepository) LockMaterialRows(ctx context.Context, tx pgx.Tx, companyID, employeeID, projectMaterialID string) ([]materialResponsibilityRow, float64, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, project_id, unit, quantity
		 FROM employee_material_responsibility
		 WHERE company_id = $1::uuid AND employee_id = $2::uuid AND project_material_id = $3::uuid AND active = true
		 ORDER BY created_at ASC
		 FOR UPDATE`,
		companyID, employeeID, projectMaterialID,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("lock material rows: %w", err)
	}
	defer rows.Close()

	var result []materialResponsibilityRow
	var total float64
	for rows.Next() {
		var row materialResponsibilityRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Unit, &row.Quantity); err != nil {
			return nil, 0, fmt.Errorf("scan responsibility row: %w", err)
		}
		total += row.Quantity
		result = append(result, row)
	}
	return result, total, rows.Err()
}

// DeductMaterialRow reduces quantity on one responsibility row.
// When remaining == 0 the row is fully consumed: we set active=false but deliberately
// leave quantity at its current value because CHECK (quantity > 0) disallows 0.
// The guarded WHERE quantity >= $newQty prevents any accidental double-deduction.
func (r *InventoryRepository) DeductMaterialRow(ctx context.Context, tx pgx.Tx, rowID string, remaining float64) error {
	if remaining <= 0 {
		tag, err := tx.Exec(ctx,
			`UPDATE employee_material_responsibility SET active = false WHERE id = $1::uuid`,
			rowID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}
	tag, err := tx.Exec(ctx,
		`UPDATE employee_material_responsibility
		 SET quantity = $1
		 WHERE id = $2::uuid AND quantity >= $1`,
		remaining, rowID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInsufficientQuantity
	}
	return nil
}

// UpsertDestMaterialRow finds an existing active responsibility row for the destination
// employee+material and adds to it, or inserts a new row. Returns the row ID.
func (r *InventoryRepository) UpsertDestMaterialRow(ctx context.Context, tx pgx.Tx, companyID, toEmployeeID, projectID, projectMaterialID, unit string, addQty float64) (string, error) {
	// Try to find an existing row for the same employee, material, AND destination project
	var existingID string
	err := tx.QueryRow(ctx,
		`SELECT id FROM employee_material_responsibility
		 WHERE company_id = $1::uuid AND employee_id = $2::uuid AND project_material_id = $3::uuid AND project_id = $4::uuid AND active = true
		 LIMIT 1
		 FOR UPDATE`,
		companyID, toEmployeeID, projectMaterialID, projectID,
	).Scan(&existingID)

	if err != nil && !isNoRows(err) {
		return "", fmt.Errorf("find dest responsibility: %w", err)
	}

	if existingID != "" {
		_, err = tx.Exec(ctx,
			`UPDATE employee_material_responsibility SET quantity = quantity + $1 WHERE id = $2::uuid`,
			addQty, existingID,
		)
		return existingID, err
	}

	// Insert new row
	var newID string
	err = tx.QueryRow(ctx,
		`INSERT INTO employee_material_responsibility
		 (company_id, employee_id, project_id, project_material_id, quantity, unit, active)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, true)
		 RETURNING id`,
		companyID, toEmployeeID, projectID, projectMaterialID, addQty, unit,
	).Scan(&newID)
	if err != nil {
		return "", fmt.Errorf("insert dest responsibility: %w", err)
	}
	return newID, nil
}

// InsertTransferRecord creates the audit row in asset_transfers.
func (r *InventoryRepository) InsertTransferRecord(
	ctx context.Context, tx pgx.Tx,
	companyID, fromEmployeeID, toEmployeeID, assetType string,
	employeeAssetID *string,
	responsibilityID *string,
	quantity float64,
	projectID *string,
	transferredBy, notes string,
) (string, error) {
	var id string
	err := tx.QueryRow(ctx,
		`INSERT INTO asset_transfers
		 (company_id, from_employee_id, to_employee_id, asset_type,
		  employee_asset_id, employee_material_responsibility_id,
		  quantity, project_id, transferred_by, transferred_at, notes)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4,
		         $5, $6,
		         $7, $8, $9::uuid, $10, $11)
		 RETURNING id`,
		companyID, fromEmployeeID, toEmployeeID, assetType,
		uuidOrNil(employeeAssetID), uuidOrNil(responsibilityID),
		quantity, uuidOrNil(projectID), transferredBy, time.Now().UTC(), nullableString(notes),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert transfer record: %w", err)
	}
	return id, nil
}

// ── Transfer list ─────────────────────────────────────────────────────────────

func (r *InventoryRepository) CountTransfers(ctx context.Context, companyID string, scopeEmployeeID *string) (int, error) {
	base := `SELECT COUNT(*)::int FROM asset_transfers at WHERE at.company_id = $1::uuid`
	args := []interface{}{companyID}

	if scopeEmployeeID != nil {
		base += ` AND (at.from_employee_id = $2::uuid OR at.to_employee_id = $2::uuid)`
		args = append(args, *scopeEmployeeID)
	}

	var count int
	err := r.db.QueryRow(ctx, base, args...).Scan(&count)
	return count, err
}

func (r *InventoryRepository) ListTransfers(ctx context.Context, companyID string, scopeEmployeeID *string, limit, offset int) ([]dto.TransferListItem, error) {
	base := `
		SELECT
			at.id,
			at.from_employee_id,
			(fe.first_name || ' ' || fe.last_name) AS from_employee_name,
			at.to_employee_id,
			(te.first_name || ' ' || te.last_name) AS to_employee_name,
			at.asset_type,
			COALESCE(ea.name, pm.material_name, '') AS item_name,
			at.quantity,
			at.transferred_at,
			at.notes,
			COALESCE(tbu.first_name || ' ' || tbu.last_name, '') AS transferred_by_name
		FROM asset_transfers at
		JOIN employees fe ON fe.id = at.from_employee_id
		JOIN employees te ON te.id = at.to_employee_id
		LEFT JOIN employee_assets ea ON ea.id = at.employee_asset_id
		LEFT JOIN employee_material_responsibility emr ON emr.id = at.employee_material_responsibility_id
		LEFT JOIN project_materials pm ON pm.id = emr.project_material_id
		LEFT JOIN users u ON u.id = at.transferred_by
		LEFT JOIN employees tbu ON tbu.id = u.employee_id
		WHERE at.company_id = $1::uuid`
	args := []interface{}{companyID}

	if scopeEmployeeID != nil {
		base += ` AND (at.from_employee_id = $2::uuid OR at.to_employee_id = $2::uuid)`
		args = append(args, *scopeEmployeeID)
		args = append(args, limit, offset)
		base += fmt.Sprintf(` ORDER BY at.transferred_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	} else {
		args = append(args, limit, offset)
		base += fmt.Sprintf(` ORDER BY at.transferred_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	}

	rows, err := r.db.Query(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}
	defer rows.Close()

	var items []dto.TransferListItem
	for rows.Next() {
		var t dto.TransferListItem
		if err := rows.Scan(
			&t.ID, &t.FromEmployeeID, &t.FromEmployeeName,
			&t.ToEmployeeID, &t.ToEmployeeName,
			&t.AssetType, &t.ItemName, &t.Quantity,
			&t.TransferredAt, &t.Notes, &t.TransferredByName,
		); err != nil {
			return nil, fmt.Errorf("scan transfer: %w", err)
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// uuidOrNil returns nil interface{} when s is nil or empty, otherwise a string.
// pgx handles string → uuid casting via $n::uuid in the SQL.
func uuidOrNil(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
