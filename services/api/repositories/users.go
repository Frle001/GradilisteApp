package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// CreateWithTx inserts a new user login account inside an existing transaction.
// must_change_password is always set to true because a temporary password is assigned.
func (r *UserRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, companyID, employeeID, email, passwordHash, role string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO users (company_id, employee_id, email, password_hash, role, active, must_change_password)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, true, true)
	`, companyID, employeeID, email, passwordHash, role)
	if err != nil {
		return fmt.Errorf("users.CreateWithTx: %w", err)
	}
	return nil
}

// UpdatePassword updates the password hash and clears the must_change_password flag.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, newHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $1, must_change_password = false
		WHERE id = $2::uuid
	`, newHash, userID)
	if err != nil {
		return fmt.Errorf("users.UpdatePassword: %w", err)
	}
	return nil
}

// DeleteByEmployeeIDWithTx permanently deletes the user account linked to an employee.
// No-op (no error) if the employee has no linked user.
func (r *UserRepository) DeleteByEmployeeIDWithTx(ctx context.Context, tx pgx.Tx, companyID, employeeID string) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM users WHERE employee_id = $1::uuid AND company_id = $2::uuid
	`, employeeID, companyID)
	return err
}

// ResetPasswordByEmployeeID replaces the password hash for the user linked to a given employee
// within the company and sets must_change_password = true so the employee is forced to change it.
// Returns ErrNotFound if the employee has no login account in this company.
func (r *UserRepository) ResetPasswordByEmployeeID(ctx context.Context, companyID, employeeID, newHash string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $1, must_change_password = true
		WHERE company_id = $2::uuid AND employee_id = $3::uuid
	`, newHash, companyID, employeeID)
	if err != nil {
		return fmt.Errorf("users.ResetPasswordByEmployeeID: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
