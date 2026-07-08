package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportMaterialEffectsRepository struct {
	db *pgxpool.Pool
}

func NewReportMaterialEffectsRepository(db *pgxpool.Pool) *ReportMaterialEffectsRepository {
	return &ReportMaterialEffectsRepository{db: db}
}

// EffectsAlreadyApplied returns true if material effects are already recorded for this report.
func (r *ReportMaterialEffectsRepository) EffectsAlreadyApplied(ctx context.Context, tx pgx.Tx, reportID string) (bool, error) {
	var count int
	err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM report_material_effects WHERE report_id = $1::uuid`,
		reportID,
	).Scan(&count)
	return count > 0, err
}

// ValidateAndApply applies material quantity changes for each activity, within the given transaction.
//
// montaža (non-VTK): subtracts quantity from available_quantity; returns error if insufficient.
// demontaža (non-VTK): adds quantity to available_quantity.
// demontaža (VTK): upserts project_material by (project_id, name, unit), then adds quantity.
// montaža (VTK): skipped — no project_material_id to target.
func (r *ReportMaterialEffectsRepository) ValidateAndApply(
	ctx context.Context,
	tx pgx.Tx,
	companyID, reportID, projectID, appliedByUserID string,
	acts []ActivityForApproval,
) error {
	for _, a := range acts {
		switch {
		case a.ActivityType == "montaza" && !a.IsVTK:
			if err := r.applyMontaza(ctx, tx, companyID, reportID, projectID, appliedByUserID, a); err != nil {
				return err
			}
		case a.ActivityType == "demontaza" && !a.IsVTK:
			if err := r.applyDemontaza(ctx, tx, companyID, reportID, projectID, appliedByUserID, a); err != nil {
				return err
			}
		case a.ActivityType == "demontaza" && a.IsVTK:
			if err := r.applyDemontazaVTK(ctx, tx, companyID, reportID, projectID, appliedByUserID, a); err != nil {
				return err
			}
		// montaža VTK: no project_material_id — skip, no material effect recorded
		}
	}
	return nil
}

func (r *ReportMaterialEffectsRepository) applyMontaza(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, appliedByUserID string,
	a ActivityForApproval,
) error {
	// Atomic subtract: only succeeds if available_quantity >= quantity.
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE project_materials
		SET available_quantity = available_quantity - $1,
		    updated_at = NOW()
		WHERE id = $2::uuid
		  AND company_id = $3::uuid
		  AND available_quantity >= $1
		RETURNING id::text
	`, a.Quantity, *a.ProjectMaterialID, companyID).Scan(&updatedID)

	if err == pgx.ErrNoRows {
		// Distinguish "not found" from "insufficient stock"
		var exists bool
		_ = tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM project_materials WHERE id = $1::uuid AND company_id = $2::uuid)`,
			*a.ProjectMaterialID, companyID,
		).Scan(&exists)
		if exists {
			return fmt.Errorf("nedovoljno dostupnih zaliha za materijal (potrebno %.2f %s, ali dostupno je manje)", a.Quantity, a.Unit)
		}
		return fmt.Errorf("materijal nije pronađen u projektu")
	}
	if err != nil {
		return fmt.Errorf("greška pri umanjenju stanja materijala: %w", err)
	}

	return r.insertEffect(ctx, tx, companyID, reportID, projectID, appliedByUserID, a, a.ProjectMaterialID)
}

func (r *ReportMaterialEffectsRepository) applyDemontaza(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, appliedByUserID string,
	a ActivityForApproval,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE project_materials
		SET available_quantity = available_quantity + $1,
		    updated_at = NOW()
		WHERE id = $2::uuid
		  AND company_id = $3::uuid
	`, a.Quantity, *a.ProjectMaterialID, companyID)
	if err != nil {
		return fmt.Errorf("greška pri povećanju stanja materijala: %w", err)
	}
	return r.insertEffect(ctx, tx, companyID, reportID, projectID, appliedByUserID, a, a.ProjectMaterialID)
}

func (r *ReportMaterialEffectsRepository) applyDemontazaVTK(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, appliedByUserID string,
	a ActivityForApproval,
) error {
	if a.CustomMaterialName == nil {
		return fmt.Errorf("VTK demontaža aktivnost nema naziv materijala")
	}

	// Upsert by (project_id, company_id, LOWER(material_name), unit) — the unique index on project_materials.
	// On insert: sets available_quantity = delta. On conflict: adds delta to existing available_quantity.
	var materialID string
	err := tx.QueryRow(ctx, `
		INSERT INTO project_materials
		    (project_id, company_id, material_name, planned_quantity, used_quantity, available_quantity, unit, source)
		VALUES ($1::uuid, $2::uuid, $3, 0, 0, $4, $5, 'demontaza')
		ON CONFLICT (project_id, company_id, LOWER(material_name), unit)
		DO UPDATE SET
		    available_quantity = project_materials.available_quantity + EXCLUDED.available_quantity,
		    active = true,
		    updated_at = NOW()
		RETURNING id::text
	`, projectID, companyID, *a.CustomMaterialName, a.Quantity, a.Unit).Scan(&materialID)
	if err != nil {
		return fmt.Errorf("greška pri upsertu materijala za demontažu: %w", err)
	}

	return r.insertEffect(ctx, tx, companyID, reportID, projectID, appliedByUserID, a, &materialID)
}

func (r *ReportMaterialEffectsRepository) insertEffect(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, appliedByUserID string,
	a ActivityForApproval,
	materialID *string,
) error {
	var appliedBy *string
	if appliedByUserID != "" {
		appliedBy = &appliedByUserID
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO report_material_effects
		    (company_id, report_id, activity_id, project_id, project_material_id, effect_type, quantity_delta, unit, applied_by)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6, $7, $8, $9::uuid)
		ON CONFLICT (report_id, activity_id) DO NOTHING
	`,
		companyID, reportID, a.ID, projectID,
		materialID,
		a.ActivityType, a.Quantity, a.Unit,
		appliedBy,
	)
	return err
}
