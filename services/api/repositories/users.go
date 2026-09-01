package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthState holds the live session-validity fields for a user.
type AuthState struct {
	Active      bool
	AuthVersion int
}

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

// ResetPasswordAndInvalidateSessions atomically updates the user's password hash,
// sets must_change_password=true, increments auth_version (invalidating all existing
// access tokens), and revokes all active refresh tokens — in a single PostgreSQL
// transaction. If any step fails the entire transaction rolls back.
// Returns ErrNotFound if the employee has no login account in this company.
func (r *UserRepository) ResetPasswordAndInvalidateSessions(ctx context.Context, companyID, employeeID, newHash string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("users.ResetPasswordAndInvalidateSessions begin: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `
		UPDATE users
		SET password_hash = $1, must_change_password = true, auth_version = auth_version + 1
		WHERE company_id = $2::uuid AND employee_id = $3::uuid
	`, newHash, companyID, employeeID)
	if err != nil {
		return fmt.Errorf("users.ResetPasswordAndInvalidateSessions update: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = (
			SELECT id FROM users
			WHERE company_id = $1::uuid AND employee_id = $2::uuid
		) AND revoked_at IS NULL
	`, companyID, employeeID); err != nil {
		return fmt.Errorf("users.ResetPasswordAndInvalidateSessions revoke: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("users.ResetPasswordAndInvalidateSessions commit: %w", err)
	}
	return nil
}

// GetAuthState returns active status and auth_version for the given user.
// Returns ErrNotFound if no user row exists.
func (r *UserRepository) GetAuthState(ctx context.Context, userID string) (*AuthState, error) {
	var s AuthState
	err := r.db.QueryRow(ctx, `
		SELECT active, auth_version FROM users WHERE id = $1::uuid
	`, userID).Scan(&s.Active, &s.AuthVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("users.GetAuthState: %w", err)
	}
	return &s, nil
}

// InvalidateSessions atomically increments auth_version and revokes all active
// refresh tokens for the given user. Any JWT issued before this call will be
// rejected by the middleware on the next request.
func (r *UserRepository) InvalidateSessions(ctx context.Context, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("users.InvalidateSessions begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE users SET auth_version = auth_version + 1 WHERE id = $1::uuid
	`, userID); err != nil {
		return fmt.Errorf("users.InvalidateSessions increment: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE user_id = $1::uuid AND revoked_at IS NULL
	`, userID); err != nil {
		return fmt.Errorf("users.InvalidateSessions revoke: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("users.InvalidateSessions commit: %w", err)
	}
	return nil
}

// InvalidateSessionsByEmployeeID finds the user account linked to the given employee
// and calls InvalidateSessions. No-op if the employee has no login account.
func (r *UserRepository) InvalidateSessionsByEmployeeID(ctx context.Context, companyID, employeeID string) error {
	var userID string
	err := r.db.QueryRow(ctx, `
		SELECT id::text FROM users WHERE company_id = $1::uuid AND employee_id = $2::uuid
	`, companyID, employeeID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("users.InvalidateSessionsByEmployeeID lookup: %w", err)
	}
	return r.InvalidateSessions(ctx, userID)
}

// DeactivateEmployeeWithInvalidation atomically sets employees.active=false,
// sets users.active=false, increments users.auth_version, and revokes all
// refresh tokens — all in a single PostgreSQL transaction.
//
// Security guarantee: either every step succeeds or none do.
// A partial state (employee inactive but sessions still valid) is impossible.
// Returns ErrNotFound if the employee does not exist in this company.
func (r *UserRepository) DeactivateEmployeeWithInvalidation(ctx context.Context, companyID, employeeID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("users.DeactivateEmployeeWithInvalidation begin: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE employees SET active = false
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, employeeID, companyID)
	if err != nil {
		return fmt.Errorf("users.DeactivateEmployeeWithInvalidation employees: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Mark user account inactive and bump auth_version so all existing JWTs are immediately
	// rejected by the middleware. No-op if this employee has no login account.
	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET active = false, auth_version = auth_version + 1
		WHERE employee_id = $1::uuid AND company_id = $2::uuid
	`, employeeID, companyID); err != nil {
		return fmt.Errorf("users.DeactivateEmployeeWithInvalidation users: %w", err)
	}

	// Revoke all refresh tokens so they cannot be used to obtain new access tokens.
	if _, err := tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = (
			SELECT id FROM users
			WHERE employee_id = $1::uuid AND company_id = $2::uuid
		) AND revoked_at IS NULL
	`, employeeID, companyID); err != nil {
		return fmt.Errorf("users.DeactivateEmployeeWithInvalidation tokens: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("users.DeactivateEmployeeWithInvalidation commit: %w", err)
	}
	return nil
}

// ActivateEmployeeAccount restores employees.active=true and users.active=true
// inside a single transaction. auth_version is deliberately left unchanged:
// tokens issued before the employee was deactivated remain invalid because
// auth_version was incremented during deactivation and is never decremented.
// Returns ErrNotFound if the employee does not exist in this company.
func (r *UserRepository) ActivateEmployeeAccount(ctx context.Context, companyID, employeeID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("users.ActivateEmployeeAccount begin: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE employees SET active = true
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, employeeID, companyID)
	if err != nil {
		return fmt.Errorf("users.ActivateEmployeeAccount employees: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Re-enable the login account. auth_version stays at its current value.
	if _, err := tx.Exec(ctx, `
		UPDATE users SET active = true
		WHERE employee_id = $1::uuid AND company_id = $2::uuid
	`, employeeID, companyID); err != nil {
		return fmt.Errorf("users.ActivateEmployeeAccount users: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("users.ActivateEmployeeAccount commit: %w", err)
	}
	return nil
}
