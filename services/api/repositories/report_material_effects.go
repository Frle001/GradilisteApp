package repositories

import (
	"context"
	"fmt"
	"log"

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

// ValidateAndApply applies material quantity changes for each activity within the given transaction.
// poslovodaEmpID is the employee ID of the report's poslovoda (used for demontaža responsibility rows).
//
// montaža (non-VTK): subtracts qty from project available_quantity and from responsibility rows.
// demontaža (non-VTK): adds qty to project available_quantity and upserts poslovoda responsibility.
// montaža (VTK): looks up material by name+unit, same dual update.
// demontaža (VTK): upserts project_material by (project_id, name, unit), same dual update.
func (r *ReportMaterialEffectsRepository) ValidateAndApply(
	ctx context.Context,
	tx pgx.Tx,
	companyID, reportID, projectID, poslovodaEmpID, appliedByUserID string,
	acts []ActivityForApproval,
) error {
	for _, a := range acts {
		switch {
		case a.ActivityType == "montaza" && !a.IsVTK:
			if err := r.applyMontaza(ctx, tx, companyID, reportID, projectID, poslovodaEmpID, appliedByUserID, a); err != nil {
				return err
			}
		case a.ActivityType == "montaza" && a.IsVTK:
			if err := r.applyMontazaVTK(ctx, tx, companyID, reportID, projectID, poslovodaEmpID, appliedByUserID, a); err != nil {
				return err
			}
		case a.ActivityType == "demontaza" && !a.IsVTK:
			if err := r.applyDemontaza(ctx, tx, companyID, reportID, projectID, poslovodaEmpID, appliedByUserID, a); err != nil {
				return err
			}
		case a.ActivityType == "demontaza" && a.IsVTK:
			if err := r.applyDemontazaVTK(ctx, tx, companyID, reportID, projectID, poslovodaEmpID, appliedByUserID, a); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ReportMaterialEffectsRepository) applyMontaza(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, poslovodaEmpID, appliedByUserID string,
	a ActivityForApproval,
) error {
	// Atomic subtract: only succeeds if available_quantity >= quantity.
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE project_materials
		SET available_quantity = available_quantity - $1,
		    used_quantity      = used_quantity      + $1,
		    updated_at = NOW()
		WHERE id = $2::uuid
		  AND company_id = $3::uuid
		  AND available_quantity >= $1
		RETURNING id::text
	`, a.Quantity, *a.ProjectMaterialID, companyID).Scan(&updatedID)

	if err == pgx.ErrNoRows {
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

	// Keep responsibility in sync: deduct the same quantity from active EMR rows.
	r.deductResponsibility(ctx, tx, companyID, *a.ProjectMaterialID, a.Quantity)

	return r.insertEffect(ctx, tx, companyID, reportID, projectID, appliedByUserID, a, a.ProjectMaterialID)
}

func (r *ReportMaterialEffectsRepository) applyMontazaVTK(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, poslovodaEmpID, appliedByUserID string,
	a ActivityForApproval,
) error {
	if a.CustomMaterialName == nil {
		return fmt.Errorf("VTR montaža aktivnost nema naziv materijala")
	}

	// Step 1: Resolve-or-create the material at zero stock.
	// If the material was never on this project, create it so subsequent logic
	// can run consistently and produce a clear stock-check error rather than a
	// generic "not found" message.
	var materialID string
	err := tx.QueryRow(ctx, `
		INSERT INTO project_materials
		    (project_id, company_id, material_name, unit, planned_quantity, used_quantity, available_quantity, source)
		VALUES ($1::uuid, $2::uuid, $3, $4, 0, 0, 0, 'report')
		ON CONFLICT (project_id, company_id, LOWER(material_name), unit)
		DO UPDATE SET active = true
		RETURNING id::text
	`, projectID, companyID, *a.CustomMaterialName, a.Unit).Scan(&materialID)
	if err != nil {
		return fmt.Errorf("greška pri pronalasku VTR materijala za montažu: %w", err)
	}

	// Step 2: Apply the montaža effect — subtract from available_quantity.
	// A newly created zero-stock material will always fail the stock check here,
	// producing the correct "not available" message.
	var updatedID string
	err = tx.QueryRow(ctx, `
		UPDATE project_materials
		SET available_quantity = available_quantity - $1,
		    used_quantity      = used_quantity      + $1,
		    updated_at = NOW()
		WHERE id = $2::uuid
		  AND company_id = $3::uuid
		  AND available_quantity >= $1
		RETURNING id::text
	`, a.Quantity, materialID, companyID).Scan(&updatedID)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("Materijal nije dostupan na projektu za ovu VTR montažu.")
	}
	if err != nil {
		return fmt.Errorf("greška pri umanjenju stanja VTR materijala: %w", err)
	}

	r.deductResponsibility(ctx, tx, companyID, updatedID, a.Quantity)

	return r.insertEffect(ctx, tx, companyID, reportID, projectID, appliedByUserID, a, &updatedID)
}

func (r *ReportMaterialEffectsRepository) applyDemontaza(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, poslovodaEmpID, appliedByUserID string,
	a ActivityForApproval,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE project_materials
		SET available_quantity = available_quantity + $1,
		    used_quantity      = GREATEST(used_quantity - $1, 0),
		    updated_at = NOW()
		WHERE id = $2::uuid
		  AND company_id = $3::uuid
	`, a.Quantity, *a.ProjectMaterialID, companyID)
	if err != nil {
		return fmt.Errorf("greška pri povećanju stanja materijala: %w", err)
	}

	// Keep responsibility in sync: add the returned quantity to the poslovoda's EMR row.
	if err := r.upsertResponsibility(ctx, tx, companyID, poslovodaEmpID, projectID, *a.ProjectMaterialID, a.Unit, a.Quantity); err != nil {
		return fmt.Errorf("greška pri ažuriranju odgovornosti za materijal: %w", err)
	}

	return r.insertEffect(ctx, tx, companyID, reportID, projectID, appliedByUserID, a, a.ProjectMaterialID)
}

func (r *ReportMaterialEffectsRepository) applyDemontazaVTK(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, poslovodaEmpID, appliedByUserID string,
	a ActivityForApproval,
) error {
	if a.CustomMaterialName == nil {
		return fmt.Errorf("VTR demontaža aktivnost nema naziv materijala")
	}

	// Step 1: Resolve-or-create the material at zero stock.
	// Baking the quantity into the INSERT VALUES would double-count on an idempotency
	// retry, because ON CONFLICT would add qty again. Keep INSERT at zero and apply
	// the effect in a separate UPDATE so each step is independently idempotent.
	var materialID string
	err := tx.QueryRow(ctx, `
		INSERT INTO project_materials
		    (project_id, company_id, material_name, unit, planned_quantity, used_quantity, available_quantity, source)
		VALUES ($1::uuid, $2::uuid, $3, $4, 0, 0, 0, 'demontaza')
		ON CONFLICT (project_id, company_id, LOWER(material_name), unit)
		DO UPDATE SET active = true
		RETURNING id::text
	`, projectID, companyID, *a.CustomMaterialName, a.Unit).Scan(&materialID)
	if err != nil {
		return fmt.Errorf("greška pri pronalasku VTR materijala za demontažu: %w", err)
	}

	// Step 2: Apply the demontaža effect — add to available_quantity.
	_, err = tx.Exec(ctx, `
		UPDATE project_materials
		SET available_quantity = available_quantity + $1,
		    used_quantity      = GREATEST(used_quantity - $1, 0),
		    updated_at = NOW()
		WHERE id = $2::uuid
		  AND company_id = $3::uuid
	`, a.Quantity, materialID, companyID)
	if err != nil {
		return fmt.Errorf("greška pri povećanju stanja VTR materijala: %w", err)
	}

	if err := r.upsertResponsibility(ctx, tx, companyID, poslovodaEmpID, projectID, materialID, a.Unit, a.Quantity); err != nil {
		return fmt.Errorf("greška pri ažuriranju odgovornosti za VTR materijal: %w", err)
	}

	return r.insertEffect(ctx, tx, companyID, reportID, projectID, appliedByUserID, a, &materialID)
}

// deductResponsibility greedily deducts qty from all active EMR rows for the given
// project_material_id, oldest first, within the current transaction.
// The rows are locked with FOR UPDATE to prevent concurrent modification.
// If the total responsibility is less than qty (pre-existing data inconsistency),
// all available responsibility is deducted and the remainder is logged but not returned as an error,
// so the approval itself is not blocked.
func (r *ReportMaterialEffectsRepository) deductResponsibility(
	ctx context.Context, tx pgx.Tx,
	companyID, projectMaterialID string,
	qty float64,
) {
	rows, err := tx.Query(ctx, `
		SELECT id, quantity
		FROM employee_material_responsibility
		WHERE company_id = $1::uuid
		  AND project_material_id = $2::uuid
		  AND active = true
		ORDER BY created_at ASC
		FOR UPDATE
	`, companyID, projectMaterialID)
	if err != nil {
		log.Printf("[report_effects] deductResponsibility: query error for material %s: %v", projectMaterialID, err)
		return
	}
	defer rows.Close()

	type emrRow struct {
		ID       string
		Quantity float64
	}
	var candidates []emrRow
	for rows.Next() {
		var row emrRow
		if err := rows.Scan(&row.ID, &row.Quantity); err != nil {
			log.Printf("[report_effects] deductResponsibility: scan error: %v", err)
			return
		}
		candidates = append(candidates, row)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[report_effects] deductResponsibility: rows error: %v", err)
		return
	}

	remaining := qty
	for _, row := range candidates {
		if remaining <= 0 {
			break
		}
		deduct := remaining
		if deduct > row.Quantity {
			deduct = row.Quantity
		}
		newQty := row.Quantity - deduct

		var execErr error
		if newQty <= 0 {
			// CHECK (quantity > 0) disallows 0; set active=false to retire the row.
			_, execErr = tx.Exec(ctx,
				`UPDATE employee_material_responsibility SET active = false WHERE id = $1::uuid`,
				row.ID,
			)
		} else {
			_, execErr = tx.Exec(ctx,
				`UPDATE employee_material_responsibility SET quantity = $1 WHERE id = $2::uuid`,
				newQty, row.ID,
			)
		}
		if execErr != nil {
			log.Printf("[report_effects] deductResponsibility: update error for row %s: %v", row.ID, execErr)
			return
		}
		remaining -= deduct
	}

	if remaining > 0 {
		// The total responsibility was less than the quantity consumed.
		// This indicates a pre-existing data inconsistency — log it for investigation.
		log.Printf("[report_effects] deductResponsibility: responsibility underflow %.4f for material %s — pre-existing inconsistency", remaining, projectMaterialID)
	}
}

// upsertResponsibility adds qty to the poslovoda's active EMR row for the given material,
// creating a new row if none exists. The row is locked with FOR UPDATE.
func (r *ReportMaterialEffectsRepository) upsertResponsibility(
	ctx context.Context, tx pgx.Tx,
	companyID, employeeID, projectID, projectMaterialID, unit string,
	qty float64,
) error {
	// Try to find an existing active row for this (employee, project, material) triple.
	var existingID string
	err := tx.QueryRow(ctx, `
		SELECT id FROM employee_material_responsibility
		WHERE company_id      = $1::uuid
		  AND employee_id     = $2::uuid
		  AND project_id      = $3::uuid
		  AND project_material_id = $4::uuid
		  AND active = true
		LIMIT 1
		FOR UPDATE
	`, companyID, employeeID, projectID, projectMaterialID).Scan(&existingID)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("find responsibility row: %w", err)
	}

	if existingID != "" {
		_, err = tx.Exec(ctx,
			`UPDATE employee_material_responsibility SET quantity = quantity + $1 WHERE id = $2::uuid`,
			qty, existingID,
		)
		return err
	}

	// Insert a new responsibility row for this poslovoda.
	_, err = tx.Exec(ctx, `
		INSERT INTO employee_material_responsibility
		    (company_id, employee_id, project_id, project_material_id, quantity, unit, active)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, true)
	`, companyID, employeeID, projectID, projectMaterialID, qty, unit)
	return err
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
