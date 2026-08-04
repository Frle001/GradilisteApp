package repositories

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

// InventoryConflictError is returned by Update when the edit would violate an
// inventory constraint. Message is a user-facing Croatian description.
type InventoryConflictError struct{ Message string }

func (e *InventoryConflictError) Error() string { return e.Message }

type MaterialPurchasesRepository struct {
	db *pgxpool.Pool
}

func NewMaterialPurchasesRepository(db *pgxpool.Pool) *MaterialPurchasesRepository {
	return &MaterialPurchasesRepository{db: db}
}

// ── WHERE builder ─────────────────────────────────────────────────────────────

type purchaseWhere struct {
	conds []string
	args  []interface{}
	idx   int
}

func newPurchaseWhere(companyID string) *purchaseWhere {
	return &purchaseWhere{
		conds: []string{"mps.company_id = $1::uuid"},
		args:  []interface{}{companyID},
		idx:   2,
	}
}

func (w *purchaseWhere) add(cond string, arg interface{}) {
	w.conds = append(w.conds, fmt.Sprintf(cond, w.idx))
	w.args = append(w.args, arg)
	w.idx++
}

func (w *purchaseWhere) sql() string {
	return "WHERE " + strings.Join(w.conds, " AND ")
}

func (w *purchaseWhere) clone() *purchaseWhere {
	c := &purchaseWhere{
		conds: make([]string, len(w.conds)),
		args:  make([]interface{}, len(w.args)),
		idx:   w.idx,
	}
	copy(c.conds, w.conds)
	copy(c.args, w.args)
	return c
}

func buildPurchaseWhere(companyID string, f dto.PurchaseFilter) *purchaseWhere {
	w := newPurchaseWhere(companyID)
	if f.ProjectID != nil {
		w.add("mps.project_id = $%d::uuid", *f.ProjectID)
	}
	if f.BuyerID != nil {
		w.add("mps.buyer_id = $%d::uuid", *f.BuyerID)
	}
	if f.DateFrom != nil {
		w.add("mps.purchased_at >= $%d::date", *f.DateFrom)
	}
	if f.DateTo != nil {
		w.add("mps.purchased_at < ($%d::date + interval '1 day')", *f.DateTo)
	}
	if f.Search != nil && strings.TrimSpace(*f.Search) != "" {
		like := "%" + *f.Search + "%"
		n := w.idx
		w.conds = append(w.conds, fmt.Sprintf(
			"(p.name ILIKE $%d OR (e.first_name||' '||e.last_name) ILIKE $%d)",
			n, n,
		))
		w.args = append(w.args, like)
		w.idx++
	}
	return w
}

// ── Form data ─────────────────────────────────────────────────────────────────

func (r *MaterialPurchasesRepository) GetFormProjects(ctx context.Context, companyID, role, employeeID string) ([]dto.PurchaseFormProject, error) {
	var query string
	var args []interface{}

	if role == "poslovoda" {
		query = `
			SELECT DISTINCT p.id::text, p.name
			FROM projects p
			JOIN project_assignments pa ON pa.project_id = p.id AND pa.company_id = p.company_id
			WHERE p.company_id = $1::uuid
			  AND p.status = 'active'
			  AND pa.employee_id = $2::uuid
			  AND pa.active = true
			ORDER BY p.name`
		args = []interface{}{companyID, employeeID}
	} else {
		query = `
			SELECT id::text, name FROM projects
			WHERE company_id = $1::uuid AND status = 'active'
			ORDER BY name`
		args = []interface{}{companyID}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dto.PurchaseFormProject
	for rows.Next() {
		var p dto.PurchaseFormProject
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if out == nil {
		out = []dto.PurchaseFormProject{}
	}
	return out, rows.Err()
}

func (r *MaterialPurchasesRepository) GetFormMaterials(ctx context.Context, companyID, projectID string) ([]dto.PurchaseFormMaterial, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, material_name, material_code, available_quantity, unit, tracking_type
		FROM project_materials
		WHERE company_id = $1::uuid AND project_id = $2::uuid AND active = true
		  AND tracking_type = 'stock'
		ORDER BY material_name
	`, companyID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dto.PurchaseFormMaterial
	for rows.Next() {
		var m dto.PurchaseFormMaterial
		if err := rows.Scan(&m.ID, &m.MaterialName, &m.MaterialCode, &m.AvailableQuantity, &m.Unit, &m.TrackingType); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if out == nil {
		out = []dto.PurchaseFormMaterial{}
	}
	return out, rows.Err()
}

// ── List / Count ──────────────────────────────────────────────────────────────

func (r *MaterialPurchasesRepository) Count(ctx context.Context, companyID string, f dto.PurchaseFilter) (int, error) {
	w := buildPurchaseWhere(companyID, f)
	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT mps.id)::int
		FROM material_purchase_sessions mps
		JOIN projects p ON p.id = mps.project_id
		JOIN employees e ON e.id = mps.buyer_id
		%s
	`, w.sql())

	var total int
	if err := r.db.QueryRow(ctx, query, w.args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *MaterialPurchasesRepository) List(ctx context.Context, companyID string, f dto.PurchaseFilter, limit, offset int) ([]dto.PurchaseListItem, error) {
	w := buildPurchaseWhere(companyID, f)
	args := append(w.args, limit, offset)
	limitIdx := w.idx
	offsetIdx := w.idx + 1

	query := fmt.Sprintf(`
		SELECT
			mps.id::text,
			mps.project_id::text,
			p.name,
			mps.buyer_id::text,
			e.first_name||' '||e.last_name,
			mps.purchased_at,
			COUNT(mpi.id)::int,
			CASE WHEN mps.receipt_file_key IS NOT NULL
				THEN '/api/material-purchases/' || mps.id::text || '/receipt'
				ELSE NULL END,
			mps.receipt_original_filename,
			mps.notes,
			mps.created_at
		FROM material_purchase_sessions mps
		JOIN projects p ON p.id = mps.project_id
		JOIN employees e ON e.id = mps.buyer_id
		LEFT JOIN material_purchase_items mpi ON mpi.purchase_session_id = mps.id
		%s
		GROUP BY mps.id, p.name, e.first_name, e.last_name
		ORDER BY mps.purchased_at DESC
		LIMIT $%d OFFSET $%d
	`, w.sql(), limitIdx, offsetIdx)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dto.PurchaseListItem
	for rows.Next() {
		var item dto.PurchaseListItem
		if err := rows.Scan(
			&item.ID, &item.ProjectID, &item.ProjectName,
			&item.BuyerID, &item.BuyerName,
			&item.PurchasedAt, &item.ItemsCount,
			&item.ReceiptFileURL, &item.ReceiptOriginalFilename,
			&item.Notes, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if out == nil {
		out = []dto.PurchaseListItem{}
	}
	return out, rows.Err()
}

// ── Detail ────────────────────────────────────────────────────────────────────

func (r *MaterialPurchasesRepository) GetByID(ctx context.Context, companyID, id string) (*dto.PurchaseDetail, error) {
	var d dto.PurchaseDetail
	err := r.db.QueryRow(ctx, `
		SELECT
			mps.id::text,
			mps.project_id::text, p.name, p.address,
			mps.buyer_id::text, e.first_name||' '||e.last_name,
			mps.purchased_at,
			CASE WHEN mps.receipt_file_key IS NOT NULL
				THEN '/api/material-purchases/' || mps.id::text || '/receipt'
				ELSE NULL END,
			mps.receipt_original_filename,
			mps.notes,
			mps.created_at
		FROM material_purchase_sessions mps
		JOIN projects p ON p.id = mps.project_id
		JOIN employees e ON e.id = mps.buyer_id
		WHERE mps.id = $1::uuid AND mps.company_id = $2::uuid
	`, id, companyID).Scan(
		&d.ID,
		&d.Project.ID, &d.Project.Name, &d.Project.Address,
		&d.Buyer.ID, &d.Buyer.FullName,
		&d.PurchasedAt,
		&d.ReceiptFileURL, &d.ReceiptOriginalFilename,
		&d.Notes, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Load items
	rows, err := r.db.Query(ctx, `
		SELECT
			mpi.id::text,
			mpi.project_material_id::text,
			pm.material_name,
			pm.material_code,
			mpi.quantity,
			mpi.unit
		FROM material_purchase_items mpi
		JOIN project_materials pm ON pm.id = mpi.project_material_id
		WHERE mpi.purchase_session_id = $1::uuid AND mpi.company_id = $2::uuid
		ORDER BY pm.material_name
	`, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	d.Items = []dto.PurchaseItemDetail{}
	for rows.Next() {
		var item dto.PurchaseItemDetail
		if err := rows.Scan(&item.ID, &item.ProjectMaterialID, &item.MaterialName, &item.MaterialCode, &item.Quantity, &item.Unit); err != nil {
			return nil, err
		}
		d.Items = append(d.Items, item)
	}
	return &d, rows.Err()
}

// GetFileKey returns the storage key and original filename for a purchase's receipt.
func (r *MaterialPurchasesRepository) GetFileKey(ctx context.Context, companyID, id string) (fileKey, origFilename string, err error) {
	var fk, fn *string
	err = r.db.QueryRow(ctx, `
		SELECT receipt_file_key, receipt_original_filename
		FROM material_purchase_sessions
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, id, companyID).Scan(&fk, &fn)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if fk != nil {
		fileKey = *fk
	}
	if fn != nil {
		origFilename = *fn
	}
	return
}

// ── Validation ────────────────────────────────────────────────────────────────

// GetActiveProject confirms the project exists, is active, and belongs to company. Returns project name.
func (r *MaterialPurchasesRepository) GetActiveProject(ctx context.Context, companyID, projectID string) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `
		SELECT name FROM projects
		WHERE id = $1::uuid AND company_id = $2::uuid AND status = 'active'
	`, projectID, companyID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

// IsProjectAssigned checks if an employee has an active assignment on the project.
func (r *MaterialPurchasesRepository) IsProjectAssigned(ctx context.Context, companyID, projectID, employeeID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM project_assignments
		WHERE project_id = $1::uuid AND employee_id = $2::uuid AND company_id = $3::uuid AND active = true
	`, projectID, employeeID, companyID).Scan(&count)
	return count > 0, err
}

// ValidateMaterials checks all materialIDs belong to the project and are active.
// Returns map[materialID]unit for valid materials.
func (r *MaterialPurchasesRepository) ValidateMaterials(ctx context.Context, companyID, projectID string, materialIDs []string) (map[string]string, error) {
	if len(materialIDs) == 0 {
		return nil, nil
	}

	// Build dynamic IN clause
	placeholders := make([]string, len(materialIDs))
	args := []interface{}{companyID, projectID}
	for i, id := range materialIDs {
		args = append(args, id)
		placeholders[i] = fmt.Sprintf("$%d::uuid", len(args))
	}
	query := fmt.Sprintf(`
		SELECT id::text, unit FROM project_materials
		WHERE company_id = $1::uuid AND project_id = $2::uuid AND active = true
		  AND id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var id, unit string
		if err := rows.Scan(&id, &unit); err != nil {
			return nil, err
		}
		result[id] = unit
	}
	return result, rows.Err()
}

// ── Create (transaction) ──────────────────────────────────────────────────────

type CreatePurchaseParams struct {
	CompanyID    string
	ProjectID    string
	BuyerID      string
	UserID       string
	PurchasedAt  time.Time
	Notes        *string
	FileKey      *string
	OrigFilename *string
	Items        []dto.CreatePurchaseItemRequest
}

func (r *MaterialPurchasesRepository) Create(ctx context.Context, p CreatePurchaseParams) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Insert purchase session
	var sessionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO material_purchase_sessions
			(company_id, project_id, buyer_id, purchased_at, notes, receipt_file_key, receipt_original_filename, created_by)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8::uuid)
		RETURNING id::text
	`, p.CompanyID, p.ProjectID, p.BuyerID, p.PurchasedAt, p.Notes, p.FileKey, p.OrigFilename, p.UserID).Scan(&sessionID)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}

	// 2. For each item: insert item, update available_quantity, upsert responsibility
	for _, item := range p.Items {
		// Insert purchase item
		if _, err = tx.Exec(ctx, `
			INSERT INTO material_purchase_items (company_id, purchase_session_id, project_material_id, quantity, unit)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
		`, p.CompanyID, sessionID, item.ProjectMaterialID, item.Quantity, item.Unit); err != nil {
			return "", fmt.Errorf("insert purchase item %s: %w", item.ProjectMaterialID, err)
		}

		// Increase project_materials.available_quantity
		if _, err = tx.Exec(ctx, `
			UPDATE project_materials
			SET available_quantity = available_quantity + $1
			WHERE id = $2::uuid AND company_id = $3::uuid
		`, item.Quantity, item.ProjectMaterialID, p.CompanyID); err != nil {
			return "", fmt.Errorf("update available_quantity %s: %w", item.ProjectMaterialID, err)
		}

		// Upsert employee_material_responsibility
		var respID string
		err = tx.QueryRow(ctx, `
			SELECT id::text FROM employee_material_responsibility
			WHERE company_id = $1::uuid AND employee_id = $2::uuid AND project_id = $3::uuid
			  AND project_material_id = $4::uuid AND active = true
			FOR UPDATE
		`, p.CompanyID, p.BuyerID, p.ProjectID, item.ProjectMaterialID).Scan(&respID)

		if errors.Is(err, pgx.ErrNoRows) {
			_, err = tx.Exec(ctx, `
				INSERT INTO employee_material_responsibility
					(company_id, employee_id, project_id, project_material_id, quantity, unit, source_purchase_session_id, active)
				VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7::uuid, true)
			`, p.CompanyID, p.BuyerID, p.ProjectID, item.ProjectMaterialID, item.Quantity, item.Unit, sessionID)
		} else if err == nil {
			_, err = tx.Exec(ctx, `
				UPDATE employee_material_responsibility
				SET quantity = quantity + $1, updated_at = NOW()
				WHERE id = $2::uuid
			`, item.Quantity, respID)
		}
		if err != nil {
			return "", fmt.Errorf("upsert responsibility %s: %w", item.ProjectMaterialID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}
	return sessionID, nil
}

// ── Update (transaction) ──────────────────────────────────────────────────────

type UpdatePurchaseParams struct {
	CompanyID   string
	SessionID   string
	PurchasedAt time.Time
	Notes       *string
	NewItems    []dto.UpdatePurchaseItemRequest
}

// Update replaces a purchase session's items and metadata inside a single transaction.
// It reverses the inventory effects of the original items and applies the effects of
// the new items using per-material net deltas. Locks are acquired in sorted UUID order
// to avoid deadlocks with concurrent approvals.
func (r *MaterialPurchasesRepository) Update(ctx context.Context, p UpdatePurchaseParams) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Lock the session row and fetch project/buyer identifiers.
	var projectID, buyerID string
	err = tx.QueryRow(ctx, `
		SELECT project_id::text, buyer_id::text
		FROM material_purchase_sessions
		WHERE id = $1::uuid AND company_id = $2::uuid
		FOR UPDATE
	`, p.SessionID, p.CompanyID).Scan(&projectID, &buyerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock session: %w", err)
	}

	// 2. Load original items.
	rows, err := tx.Query(ctx, `
		SELECT project_material_id::text, quantity
		FROM material_purchase_items
		WHERE purchase_session_id = $1::uuid AND company_id = $2::uuid
	`, p.SessionID, p.CompanyID)
	if err != nil {
		return fmt.Errorf("load original items: %w", err)
	}
	oldMap := map[string]float64{}
	for rows.Next() {
		var matID string
		var qty float64
		if err := rows.Scan(&matID, &qty); err != nil {
			rows.Close()
			return fmt.Errorf("scan original item: %w", err)
		}
		oldMap[matID] += qty
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate original items: %w", err)
	}

	// 3. Build new-quantity and unit maps from the replacement items.
	newMap := map[string]float64{}
	newUnitMap := map[string]string{}
	for _, item := range p.NewItems {
		newMap[item.ProjectMaterialID] += item.Quantity
		newUnitMap[item.ProjectMaterialID] = item.Unit
	}

	// 4. Collect all affected material IDs and sort them for consistent lock ordering
	//    to prevent deadlocks with concurrent report-approval transactions.
	allMatIDs := make([]string, 0, len(oldMap)+len(newMap))
	seen := map[string]struct{}{}
	for id := range oldMap {
		seen[id] = struct{}{}
		allMatIDs = append(allMatIDs, id)
	}
	for id := range newMap {
		if _, ok := seen[id]; !ok {
			allMatIDs = append(allMatIDs, id)
		}
	}
	sort.Strings(allMatIDs)

	// 5. Apply net deltas for each affected material.
	for _, matID := range allMatIDs {
		delta := newMap[matID] - oldMap[matID]
		if delta == 0 {
			continue
		}

		// --- project_materials ---
		var avail float64
		err = tx.QueryRow(ctx, `
			SELECT available_quantity
			FROM project_materials
			WHERE id = $1::uuid AND company_id = $2::uuid
			FOR UPDATE
		`, matID, p.CompanyID).Scan(&avail)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("materijal %s nije pronađen u projektu", matID)
		}
		if err != nil {
			return fmt.Errorf("lock project_materials %s: %w", matID, err)
		}
		if avail+delta < 0 {
			return &InventoryConflictError{
				Message: "Nije moguće urediti upis — dostupne zalihe za jedan od materijala postale bi negativne (materijal je već dijelom ugrađen)",
			}
		}
		if _, err = tx.Exec(ctx, `
			UPDATE project_materials
			SET available_quantity = available_quantity + $1
			WHERE id = $2::uuid AND company_id = $3::uuid
		`, delta, matID, p.CompanyID); err != nil {
			return fmt.Errorf("update project_materials %s: %w", matID, err)
		}

		// --- employee_material_responsibility ---
		var emrID string
		var emrQty float64
		err = tx.QueryRow(ctx, `
			SELECT id::text, quantity
			FROM employee_material_responsibility
			WHERE employee_id = $1::uuid AND project_material_id = $2::uuid
			  AND company_id = $3::uuid AND active = true
			FOR UPDATE
		`, buyerID, matID, p.CompanyID).Scan(&emrID, &emrQty)

		if errors.Is(err, pgx.ErrNoRows) {
			// No active responsibility row exists.
			if delta > 0 {
				unit := newUnitMap[matID]
				if _, err = tx.Exec(ctx, `
					INSERT INTO employee_material_responsibility
						(company_id, employee_id, project_id, project_material_id, quantity, unit, source_purchase_session_id, active)
					VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7::uuid, true)
				`, p.CompanyID, buyerID, projectID, matID, delta, unit, p.SessionID); err != nil {
					return fmt.Errorf("insert emr %s: %w", matID, err)
				}
			}
			// delta < 0 with no EMR row: responsibility was already removed; skip.
		} else if err != nil {
			return fmt.Errorf("lock emr %s: %w", matID, err)
		} else {
			newEMRQty := emrQty + delta
			if newEMRQty < 0 {
				return &InventoryConflictError{
					Message: "Nije moguće urediti upis — odgovornost za materijal je djelomično ili potpuno bila prenijeta",
				}
			}
			if _, err = tx.Exec(ctx, `
				UPDATE employee_material_responsibility
				SET quantity = $1, updated_at = NOW()
				WHERE id = $2::uuid
			`, newEMRQty, emrID); err != nil {
				return fmt.Errorf("update emr %s: %w", matID, err)
			}
		}
	}

	// 6. Replace items: delete old, insert new.
	if _, err = tx.Exec(ctx, `
		DELETE FROM material_purchase_items
		WHERE purchase_session_id = $1::uuid AND company_id = $2::uuid
	`, p.SessionID, p.CompanyID); err != nil {
		return fmt.Errorf("delete old items: %w", err)
	}
	for _, item := range p.NewItems {
		if _, err = tx.Exec(ctx, `
			INSERT INTO material_purchase_items (company_id, purchase_session_id, project_material_id, quantity, unit)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
		`, p.CompanyID, p.SessionID, item.ProjectMaterialID, item.Quantity, item.Unit); err != nil {
			return fmt.Errorf("insert item %s: %w", item.ProjectMaterialID, err)
		}
	}

	// 7. Update session metadata.
	if _, err = tx.Exec(ctx, `
		UPDATE material_purchase_sessions
		SET purchased_at = $1, notes = $2
		WHERE id = $3::uuid AND company_id = $4::uuid
	`, p.PurchasedAt, p.Notes, p.SessionID, p.CompanyID); err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	return tx.Commit(ctx)
}
