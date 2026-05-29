package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmployeeAsset struct {
	ID           string
	CompanyID    string
	EmployeeID   string
	AssetType    string
	Name         string
	Quantity     float64
	Unit         *string
	SerialNumber *string
	Notes        *string
	Active       bool
	AssignedAt   time.Time
	AssignedBy   *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type EmployeeAssetRepository struct {
	db *pgxpool.Pool
}

func NewEmployeeAssetRepository(db *pgxpool.Pool) *EmployeeAssetRepository {
	return &EmployeeAssetRepository{db: db}
}

func (r *EmployeeAssetRepository) ListByEmployee(ctx context.Context, companyID, employeeID string) ([]EmployeeAsset, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id::text, company_id::text, employee_id::text,
			asset_type, name, quantity, unit,
			serial_number, notes, active,
			assigned_at, assigned_by::text,
			created_at, updated_at
		FROM employee_assets
		WHERE employee_id = $1::uuid AND company_id = $2::uuid
		ORDER BY active DESC, assigned_at DESC
	`, employeeID, companyID)
	if err != nil {
		return nil, fmt.Errorf("employee_assets.List: %w", err)
	}
	defer rows.Close()

	var result []EmployeeAsset
	for rows.Next() {
		var a EmployeeAsset
		if err := rows.Scan(
			&a.ID, &a.CompanyID, &a.EmployeeID,
			&a.AssetType, &a.Name, &a.Quantity, &a.Unit,
			&a.SerialNumber, &a.Notes, &a.Active,
			&a.AssignedAt, &a.AssignedBy,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("employee_assets.List scan: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *EmployeeAssetRepository) GetByID(ctx context.Context, companyID, id string) (*EmployeeAsset, error) {
	var a EmployeeAsset
	err := r.db.QueryRow(ctx, `
		SELECT
			id::text, company_id::text, employee_id::text,
			asset_type, name, quantity, unit,
			serial_number, notes, active,
			assigned_at, assigned_by::text,
			created_at, updated_at
		FROM employee_assets
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, id, companyID).Scan(
		&a.ID, &a.CompanyID, &a.EmployeeID,
		&a.AssetType, &a.Name, &a.Quantity, &a.Unit,
		&a.SerialNumber, &a.Notes, &a.Active,
		&a.AssignedAt, &a.AssignedBy,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("employee_assets.GetByID: %w", err)
	}
	return &a, nil
}

func (r *EmployeeAssetRepository) Create(ctx context.Context, companyID, employeeID, assetType, name string, quantity float64, unit, serialNumber, notes, assignedByUserID *string) (*EmployeeAsset, error) {
	var a EmployeeAsset
	err := r.db.QueryRow(ctx, `
		INSERT INTO employee_assets
			(company_id, employee_id, asset_type, name, quantity, unit, serial_number, notes, assigned_by)
		VALUES
			($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9::uuid)
		RETURNING
			id::text, company_id::text, employee_id::text,
			asset_type, name, quantity, unit,
			serial_number, notes, active,
			assigned_at, assigned_by::text,
			created_at, updated_at
	`, companyID, employeeID, assetType, name, quantity, unit, serialNumber, notes, assignedByUserID).Scan(
		&a.ID, &a.CompanyID, &a.EmployeeID,
		&a.AssetType, &a.Name, &a.Quantity, &a.Unit,
		&a.SerialNumber, &a.Notes, &a.Active,
		&a.AssignedAt, &a.AssignedBy,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("employee_assets.Create: %w", err)
	}
	return &a, nil
}

func (r *EmployeeAssetRepository) Update(ctx context.Context, companyID, id, assetType, name string, quantity float64, unit, serialNumber, notes *string) (*EmployeeAsset, error) {
	var a EmployeeAsset
	err := r.db.QueryRow(ctx, `
		UPDATE employee_assets
		SET asset_type = $3, name = $4, quantity = $5,
		    unit = $6, serial_number = $7, notes = $8
		WHERE id = $1::uuid AND company_id = $2::uuid
		RETURNING
			id::text, company_id::text, employee_id::text,
			asset_type, name, quantity, unit,
			serial_number, notes, active,
			assigned_at, assigned_by::text,
			created_at, updated_at
	`, id, companyID, assetType, name, quantity, unit, serialNumber, notes).Scan(
		&a.ID, &a.CompanyID, &a.EmployeeID,
		&a.AssetType, &a.Name, &a.Quantity, &a.Unit,
		&a.SerialNumber, &a.Notes, &a.Active,
		&a.AssignedAt, &a.AssignedBy,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("employee_assets.Update: %w", err)
	}
	return &a, nil
}

func (r *EmployeeAssetRepository) SetActive(ctx context.Context, companyID, id string, active bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE employee_assets SET active = $3
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, id, companyID, active)
	if err != nil {
		return fmt.Errorf("employee_assets.SetActive: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
