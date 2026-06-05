package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

type ProjectMaterialRepository struct {
	db *pgxpool.Pool
}

func NewProjectMaterialRepository(db *pgxpool.Pool) *ProjectMaterialRepository {
	return &ProjectMaterialRepository{db: db}
}

var ErrMaterialNotFound = errors.New("material not found")
var ErrMaterialDuplicate = errors.New("material with this name and unit already exists")

func isDuplicateKey(err error) bool {
	type pgError interface{ SQLState() string }
	var pge pgError
	if errors.As(err, &pge) {
		return pge.SQLState() == "23505"
	}
	return false
}

type MaterialFilter struct {
	Search     string
	ActiveOnly bool
}

func (r *ProjectMaterialRepository) List(ctx context.Context, projectID, companyID string, f MaterialFilter) ([]dto.MaterialListItem, error) {
	conditions := []string{"project_id = $1", "company_id = $2"}
	args := []interface{}{projectID, companyID}
	idx := 3

	if f.ActiveOnly {
		conditions = append(conditions, fmt.Sprintf("active = $%d", idx))
		args = append(args, true)
		idx++
	}
	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(material_name ILIKE $%d OR material_code ILIKE $%d)", idx, idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	_ = idx

	where := "WHERE " + strings.Join(conditions, " AND ")
	q := `SELECT id, material_name, material_code, planned_quantity, used_quantity, available_quantity,
	             unit, source, active, created_at, updated_at
	      FROM project_materials ` + where + ` ORDER BY material_name ASC`

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.MaterialListItem
	for rows.Next() {
		var m dto.MaterialListItem
		if err := rows.Scan(
			&m.ID, &m.MaterialName, &m.MaterialCode,
			&m.PlannedQuantity, &m.UsedQuantity, &m.AvailableQuantity,
			&m.Unit, &m.Source, &m.Active, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *ProjectMaterialRepository) GetByID(ctx context.Context, id, projectID, companyID string) (*dto.MaterialListItem, error) {
	q := `SELECT id, material_name, material_code, planned_quantity, used_quantity, available_quantity,
	             unit, source, active, created_at, updated_at
	      FROM project_materials
	      WHERE id = $1 AND project_id = $2 AND company_id = $3`

	var m dto.MaterialListItem
	err := r.db.QueryRow(ctx, q, id, projectID, companyID).Scan(
		&m.ID, &m.MaterialName, &m.MaterialCode,
		&m.PlannedQuantity, &m.UsedQuantity, &m.AvailableQuantity,
		&m.Unit, &m.Source, &m.Active, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMaterialNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ProjectMaterialRepository) Create(ctx context.Context, projectID, companyID string, req dto.CreateMaterialRequest) (*dto.MaterialListItem, error) {
	q := `INSERT INTO project_materials
	        (project_id, company_id, material_name, material_code, planned_quantity, used_quantity, available_quantity, unit, source)
	      VALUES ($1, $2, $3, $4, $5, 0, 0, $6, 'manual')
	      RETURNING id, material_name, material_code, planned_quantity, used_quantity, available_quantity,
	                unit, source, active, created_at, updated_at`

	var m dto.MaterialListItem
	if err := r.db.QueryRow(ctx, q,
		projectID, companyID,
		req.MaterialName, req.MaterialCode,
		req.PlannedQuantity, req.Unit,
	).Scan(
		&m.ID, &m.MaterialName, &m.MaterialCode,
		&m.PlannedQuantity, &m.UsedQuantity, &m.AvailableQuantity,
		&m.Unit, &m.Source, &m.Active, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrMaterialDuplicate
		}
		return nil, err
	}
	return &m, nil
}

func (r *ProjectMaterialRepository) Update(ctx context.Context, id, projectID, companyID string, req dto.UpdateMaterialRequest) (*dto.MaterialListItem, error) {
	q := `UPDATE project_materials
	      SET material_name = $1, material_code = $2, planned_quantity = $3, unit = $4
	      WHERE id = $5 AND project_id = $6 AND company_id = $7
	      RETURNING id, material_name, material_code, planned_quantity, used_quantity, available_quantity,
	                unit, source, active, created_at, updated_at`

	var m dto.MaterialListItem
	err := r.db.QueryRow(ctx, q,
		req.MaterialName, req.MaterialCode, req.PlannedQuantity, req.Unit,
		id, projectID, companyID,
	).Scan(
		&m.ID, &m.MaterialName, &m.MaterialCode,
		&m.PlannedQuantity, &m.UsedQuantity, &m.AvailableQuantity,
		&m.Unit, &m.Source, &m.Active, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMaterialNotFound
	}
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrMaterialDuplicate
		}
		return nil, err
	}
	return &m, nil
}

func (r *ProjectMaterialRepository) Deactivate(ctx context.Context, id, projectID, companyID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE project_materials SET active = false WHERE id = $1 AND project_id = $2 AND company_id = $3`,
		id, projectID, companyID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMaterialNotFound
	}
	return nil
}

// UpsertConfirmRowWithTx inserts or updates a project_material from a WizardConfirmRow
// (which may have been edited by the director before confirming).
func (r *ProjectMaterialRepository) UpsertConfirmRowWithTx(ctx context.Context, tx pgx.Tx, projectID, companyID string, row dto.WizardConfirmRow) error {
	q := `INSERT INTO project_materials
	        (project_id, company_id, material_name, planned_quantity, used_quantity, available_quantity, unit, source)
	      VALUES ($1, $2, $3, $4, 0, 0, $5, 'excel')
	      ON CONFLICT (project_id, company_id, LOWER(material_name), unit)
	      DO UPDATE SET
	        material_name    = EXCLUDED.material_name,
	        planned_quantity = EXCLUDED.planned_quantity,
	        active           = true`

	_, err := tx.Exec(ctx, q,
		projectID, companyID,
		row.MaterialName, row.Quantity, row.Unit,
	)
	return err
}

// UpsertWizardRowWithTx inserts or updates a project_material from a WizardPreviewRow.
func (r *ProjectMaterialRepository) UpsertWizardRowWithTx(ctx context.Context, tx pgx.Tx, projectID, companyID string, row dto.WizardPreviewRow) error {
	q := `INSERT INTO project_materials
	        (project_id, company_id, material_name, planned_quantity, used_quantity, available_quantity, unit, source)
	      VALUES ($1, $2, $3, $4, 0, 0, $5, 'excel')
	      ON CONFLICT (project_id, company_id, LOWER(material_name), unit)
	      DO UPDATE SET
	        material_name    = EXCLUDED.material_name,
	        planned_quantity = EXCLUDED.planned_quantity,
	        active           = true`

	_, err := tx.Exec(ctx, q,
		projectID, companyID,
		row.MaterialName, row.Quantity, row.Unit,
	)
	return err
}

// UpsertWithTx inserts or updates a material within a transaction.
// Returns (inserted bool, err).
func (r *ProjectMaterialRepository) UpsertWithTx(ctx context.Context, tx pgx.Tx, projectID, companyID string, row dto.ImportPreviewRow) (inserted bool, err error) {
	q := `INSERT INTO project_materials
	        (project_id, company_id, material_name, material_code, planned_quantity, used_quantity, available_quantity, unit, source)
	      VALUES ($1, $2, $3, $4, $5, 0, 0, $6, 'excel')
	      ON CONFLICT (project_id, company_id, LOWER(material_name), unit)
	      DO UPDATE SET
	        material_name    = EXCLUDED.material_name,
	        material_code    = COALESCE(EXCLUDED.material_code, project_materials.material_code),
	        planned_quantity = EXCLUDED.planned_quantity,
	        active           = true
	      RETURNING (xmax = 0) AS is_insert`

	var isInsert bool
	err = tx.QueryRow(ctx, q,
		projectID, companyID,
		row.MaterialName, row.MaterialCode,
		row.Quantity, row.Unit,
	).Scan(&isInsert)
	if err != nil {
		return false, err
	}
	return isInsert, nil
}
