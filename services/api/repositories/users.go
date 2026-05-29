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
