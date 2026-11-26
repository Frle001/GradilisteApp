package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

type WarehouseRepository struct {
	db *pgxpool.Pool
}

func NewWarehouseRepository(db *pgxpool.Pool) *WarehouseRepository {
	return &WarehouseRepository{db: db}
}

// ── Overview ──────────────────────────────────────────────────────────────────

func (r *WarehouseRepository) GetOverview(ctx context.Context, companyID string) (*dto.WarehouseOverview, error) {
	var o dto.WarehouseOverview

	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*)::int                                                               AS total_items,
			COALESCE(SUM(quantity * avg_cost), 0)                                       AS total_value,
			COUNT(*) FILTER (WHERE quantity > 0 AND quantity < 5)::int                 AS low_stock_items
		FROM warehouse_balances
		WHERE company_id = $1::uuid AND active = true`,
		companyID,
	).Scan(&o.TotalItems, &o.TotalValue, &o.LowStockItems)
	if err != nil {
		return nil, fmt.Errorf("warehouse overview: %w", err)
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM warehouse_receipts
		 WHERE company_id = $1::uuid AND created_at >= NOW() - INTERVAL '30 days'`,
		companyID,
	).Scan(&o.RecentReceipts)
	if err != nil {
		return nil, fmt.Errorf("warehouse overview receipts: %w", err)
	}

	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM warehouse_movements
		 WHERE company_id = $1::uuid AND created_at >= NOW() - INTERVAL '30 days'`,
		companyID,
	).Scan(&o.RecentMovements)
	if err != nil {
		return nil, fmt.Errorf("warehouse overview movements: %w", err)
	}

	return &o, nil
}

// ── Stock ─────────────────────────────────────────────────────────────────────

func (r *WarehouseRepository) ListStock(ctx context.Context, companyID, search string, activeOnly bool) ([]dto.WarehouseStockItem, error) {
	q := `
		SELECT id, material_name, material_code, unit, quantity, avg_cost,
		       quantity * avg_cost AS total_value, active, updated_at
		FROM warehouse_balances
		WHERE company_id = $1::uuid`
	args := []interface{}{companyID}
	idx := 2

	if activeOnly {
		q += fmt.Sprintf(` AND active = $%d`, idx)
		args = append(args, true)
		idx++
	}
	if search != "" {
		q += fmt.Sprintf(` AND (material_name ILIKE $%d OR material_code ILIKE $%d)`, idx, idx)
		args = append(args, "%"+search+"%")
		idx++ //nolint:ineffassign
	}
	q += ` ORDER BY material_name ASC`

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list warehouse stock: %w", err)
	}
	defer rows.Close()

	var items []dto.WarehouseStockItem
	for rows.Next() {
		var s dto.WarehouseStockItem
		if err := rows.Scan(&s.ID, &s.MaterialName, &s.MaterialCode, &s.Unit,
			&s.Quantity, &s.AvgCost, &s.TotalValue, &s.Active, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *WarehouseRepository) GetStockByID(ctx context.Context, companyID, balanceID string) (*dto.WarehouseStockDetail, error) {
	var d dto.WarehouseStockDetail
	err := r.db.QueryRow(ctx, `
		SELECT id, material_name, material_code, unit, quantity, avg_cost,
		       quantity * avg_cost AS total_value, active, updated_at
		FROM warehouse_balances
		WHERE id = $1::uuid AND company_id = $2::uuid`,
		balanceID, companyID,
	).Scan(&d.ID, &d.MaterialName, &d.MaterialCode, &d.Unit,
		&d.Quantity, &d.AvgCost, &d.TotalValue, &d.Active, &d.UpdatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get stock: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT wm.id, wm.warehouse_balance_id, wb.material_name, wb.unit,
		       wm.movement_type, wm.quantity_delta, wm.quantity_before, wm.quantity_after,
		       wm.unit_price, wm.avg_cost_before, wm.avg_cost_after,
		       wm.project_id::text, p.name, wm.notes,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS created_by_name,
		       wm.created_at
		FROM warehouse_movements wm
		JOIN warehouse_balances wb ON wb.id = wm.warehouse_balance_id
		LEFT JOIN projects p ON p.id = wm.project_id
		JOIN users u ON u.id = wm.created_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE wm.warehouse_balance_id = $1::uuid AND wm.company_id = $2::uuid
		ORDER BY wm.created_at DESC
		LIMIT 50`,
		balanceID, companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list movements for stock: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m dto.WarehouseMovementItem
		if err := rows.Scan(
			&m.ID, &m.WarehouseBalID, &m.MaterialName, &m.Unit,
			&m.MovementType, &m.QuantityDelta, &m.QuantityBefore, &m.QuantityAfter,
			&m.UnitPrice, &m.AvgCostBefore, &m.AvgCostAfter,
			&m.ProjectID, &m.ProjectName, &m.Notes,
			&m.CreatedByName, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		d.Movements = append(d.Movements, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if d.Movements == nil {
		d.Movements = []dto.WarehouseMovementItem{}
	}
	return &d, nil
}

// IsProjectAssigned returns true if the employee is actively assigned to the project.
func (r *WarehouseRepository) IsProjectAssigned(ctx context.Context, companyID, projectID, employeeID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_assignments
			WHERE project_id  = $1::uuid
			  AND company_id  = $2::uuid
			  AND employee_id = $3::uuid
			  AND active      = true
		)`, projectID, companyID, employeeID,
	).Scan(&exists)
	return exists, err
}

// ── Balance operations (Tx) ───────────────────────────────────────────────────

type BalanceSnapshot struct {
	ID           string
	Quantity     float64
	AvgCost      float64
	MaterialName string
	MaterialCode *string
	Unit         string
}

// FindOrCreateBalance ensures a warehouse_balances row exists and returns a locked snapshot.
func (r *WarehouseRepository) FindOrCreateBalance(ctx context.Context, tx pgx.Tx, companyID, materialName, unit string, materialCode *string) (*BalanceSnapshot, error) {
	var snap BalanceSnapshot

	// Try to lock an existing row first.
	err := tx.QueryRow(ctx, `
		SELECT id, quantity, avg_cost
		FROM warehouse_balances
		WHERE company_id = $1::uuid AND LOWER(material_name) = LOWER($2) AND unit = $3
		FOR UPDATE`,
		companyID, materialName, unit,
	).Scan(&snap.ID, &snap.Quantity, &snap.AvgCost)

	if err != nil && !isNoRows(err) {
		return nil, fmt.Errorf("lock balance: %w", err)
	}

	if isNoRows(err) {
		// Insert new balance row.
		err = tx.QueryRow(ctx, `
			INSERT INTO warehouse_balances (company_id, material_name, material_code, unit)
			VALUES ($1::uuid, $2, $3, $4)
			ON CONFLICT (company_id, LOWER(material_name), unit)
			  DO UPDATE SET active = true, updated_at = now()
			RETURNING id, quantity, avg_cost`,
			companyID, materialName, materialCode, unit,
		).Scan(&snap.ID, &snap.Quantity, &snap.AvgCost)
		if err != nil {
			return nil, fmt.Errorf("create balance: %w", err)
		}
	}
	return &snap, nil
}

// LockBalance locks an existing balance row FOR UPDATE and returns its snapshot including material metadata.
func (r *WarehouseRepository) LockBalance(ctx context.Context, tx pgx.Tx, companyID, balanceID string) (*BalanceSnapshot, error) {
	var snap BalanceSnapshot
	err := tx.QueryRow(ctx, `
		SELECT id, quantity, avg_cost, material_name, material_code, unit
		FROM warehouse_balances
		WHERE id = $1::uuid AND company_id = $2::uuid AND active = true
		FOR UPDATE`,
		balanceID, companyID,
	).Scan(&snap.ID, &snap.Quantity, &snap.AvgCost, &snap.MaterialName, &snap.MaterialCode, &snap.Unit)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock balance: %w", err)
	}
	return &snap, nil
}

// UpdateBalance sets the new quantity and avg_cost on a balance row.
func (r *WarehouseRepository) UpdateBalance(ctx context.Context, tx pgx.Tx, balanceID string, newQty, newAvgCost float64) error {
	_, err := tx.Exec(ctx, `
		UPDATE warehouse_balances
		SET quantity = $1, avg_cost = $2, updated_at = now()
		WHERE id = $3::uuid`,
		newQty, newAvgCost, balanceID,
	)
	return err
}

// ── Receipt persistence ───────────────────────────────────────────────────────

func (r *WarehouseRepository) InsertReceipt(ctx context.Context, tx pgx.Tx, companyID, createdBy string, receiptNumber, supplier, notes *string, receiptDate time.Time) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO warehouse_receipts (company_id, receipt_number, supplier, receipt_date, notes, created_by)
		VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid)
		RETURNING id`,
		companyID, receiptNumber, supplier, receiptDate, notes, createdBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert receipt: %w", err)
	}
	return id, nil
}

func (r *WarehouseRepository) InsertReceiptLine(ctx context.Context, tx pgx.Tx, receiptID, balanceID string, qty float64, unitPrice *float64) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO warehouse_receipt_lines (receipt_id, warehouse_balance_id, quantity, unit_price)
		VALUES ($1::uuid, $2::uuid, $3, $4)
		RETURNING id`,
		receiptID, balanceID, qty, unitPrice,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert receipt line: %w", err)
	}
	return id, nil
}

// ── Movement log ──────────────────────────────────────────────────────────────

type InsertMovementParams struct {
	CompanyID      string
	BalanceID      string
	MovementType   string
	QuantityDelta  float64
	QuantityBefore float64
	QuantityAfter  float64
	UnitPrice      float64
	AvgCostBefore  float64
	AvgCostAfter   float64
	ReceiptLineID  *string
	ProjectID      *string
	ProjectMatID   *string
	Notes          *string
	CreatedBy      string
}

func (r *WarehouseRepository) InsertMovement(ctx context.Context, tx pgx.Tx, p InsertMovementParams) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO warehouse_movements
		  (company_id, warehouse_balance_id, movement_type,
		   quantity_delta, quantity_before, quantity_after,
		   unit_price, avg_cost_before, avg_cost_after,
		   receipt_line_id, project_id, project_material_id, notes, created_by)
		VALUES
		  ($1::uuid, $2::uuid, $3,
		   $4, $5, $6,
		   $7, $8, $9,
		   $10, $11, $12, $13, $14::uuid)
		RETURNING id`,
		p.CompanyID, p.BalanceID, p.MovementType,
		p.QuantityDelta, p.QuantityBefore, p.QuantityAfter,
		p.UnitPrice, p.AvgCostBefore, p.AvgCostAfter,
		uuidOrNil(p.ReceiptLineID), uuidOrNil(p.ProjectID), uuidOrNil(p.ProjectMatID), p.Notes, p.CreatedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert movement: %w", err)
	}
	return id, nil
}

// ── Movement list ─────────────────────────────────────────────────────────────

func (r *WarehouseRepository) CountMovements(ctx context.Context, companyID string, movementType, balanceID *string) (int, error) {
	q := `SELECT COUNT(*)::int FROM warehouse_movements WHERE company_id = $1::uuid`
	args := []interface{}{companyID}
	idx := 2

	if movementType != nil {
		q += fmt.Sprintf(` AND movement_type = $%d`, idx)
		args = append(args, *movementType)
		idx++
	}
	if balanceID != nil {
		q += fmt.Sprintf(` AND warehouse_balance_id = $%d::uuid`, idx)
		args = append(args, *balanceID)
		idx++ //nolint:ineffassign
	}

	var count int
	return count, r.db.QueryRow(ctx, q, args...).Scan(&count)
}

func (r *WarehouseRepository) ListMovements(ctx context.Context, companyID string, movementType, balanceID *string, limit, offset int) ([]dto.WarehouseMovementItem, error) {
	q := `
		SELECT wm.id, wm.warehouse_balance_id::text, wb.material_name, wb.unit,
		       wm.movement_type, wm.quantity_delta, wm.quantity_before, wm.quantity_after,
		       wm.unit_price, wm.avg_cost_before, wm.avg_cost_after,
		       wm.project_id::text, p.name, wm.notes,
		       COALESCE(e.first_name || ' ' || e.last_name, u.email) AS created_by_name,
		       wm.created_at
		FROM warehouse_movements wm
		JOIN warehouse_balances wb ON wb.id = wm.warehouse_balance_id
		LEFT JOIN projects p ON p.id = wm.project_id
		JOIN users u ON u.id = wm.created_by
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE wm.company_id = $1::uuid`
	args := []interface{}{companyID}
	idx := 2

	if movementType != nil {
		q += fmt.Sprintf(` AND wm.movement_type = $%d`, idx)
		args = append(args, *movementType)
		idx++
	}
	if balanceID != nil {
		q += fmt.Sprintf(` AND wm.warehouse_balance_id = $%d::uuid`, idx)
		args = append(args, *balanceID)
		idx++
	}
	args = append(args, limit, offset)
	q += fmt.Sprintf(` ORDER BY wm.created_at DESC LIMIT $%d OFFSET $%d`, idx, idx+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	var items []dto.WarehouseMovementItem
	for rows.Next() {
		var m dto.WarehouseMovementItem
		if err := rows.Scan(
			&m.ID, &m.WarehouseBalID, &m.MaterialName, &m.Unit,
			&m.MovementType, &m.QuantityDelta, &m.QuantityBefore, &m.QuantityAfter,
			&m.UnitPrice, &m.AvgCostBefore, &m.AvgCostAfter,
			&m.ProjectID, &m.ProjectName, &m.Notes,
			&m.CreatedByName, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// ── Idempotency ───────────────────────────────────────────────────────────────

type IdempotencyRecord struct {
	PayloadHash string
	ResultID    *string
}

// TryInsertIdempotencyKey atomically inserts a new idempotency key inside the
// caller's transaction. Returns (record, true, nil) for a new key; returns
// (existingRecord, false, nil) when the key was already committed by a prior
// request. Concurrent requests with the same key block until the first commits,
// then see the committed record — PostgreSQL's UNIQUE constraint serialises them.
func (r *WarehouseRepository) TryInsertIdempotencyKey(
	ctx context.Context, tx pgx.Tx,
	companyID, idempotencyKey, opType, payloadHash string,
) (*IdempotencyRecord, bool, error) {
	var rec IdempotencyRecord
	var isNew bool
	err := tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO warehouse_idempotency_keys
				(company_id, idempotency_key, operation_type, payload_hash)
			VALUES ($1::uuid, $2, $3, $4)
			ON CONFLICT (company_id, idempotency_key) DO NOTHING
			RETURNING payload_hash, result_id, true AS is_new
		)
		SELECT is_new, payload_hash, result_id FROM ins
		UNION ALL
		SELECT false::bool AS is_new, payload_hash, result_id
		FROM warehouse_idempotency_keys
		WHERE company_id = $1::uuid AND idempotency_key = $2
		  AND NOT EXISTS (SELECT 1 FROM ins)
	`, companyID, idempotencyKey, opType, payloadHash,
	).Scan(&isNew, &rec.PayloadHash, &rec.ResultID)
	if err != nil {
		return nil, false, fmt.Errorf("idempotency insert: %w", err)
	}
	return &rec, isNew, nil
}

// UpdateIdempotencyResult sets the result_id once the operation has succeeded.
func (r *WarehouseRepository) UpdateIdempotencyResult(
	ctx context.Context, tx pgx.Tx,
	companyID, idempotencyKey, resultID string,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE warehouse_idempotency_keys
		SET result_id = $1
		WHERE company_id = $2::uuid AND idempotency_key = $3
	`, resultID, companyID, idempotencyKey)
	return err
}
