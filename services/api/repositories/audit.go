package repositories

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

type AuditParams struct {
	CompanyID  string
	UserID     *string
	EmployeeID *string
	Action     string
	EntityType string
	EntityID   *string
	OldData    interface{}
	NewData    interface{}
	IPAddress  string
	UserAgent  string
}

// Log inserts an audit_logs record. Safe to call in a goroutine — errors are logged, never returned.
func (r *AuditRepository) Log(ctx context.Context, p AuditParams) {
	oldJSON, err := auditMarshal(p.OldData)
	if err != nil {
		log.Printf("audit: marshal old_data: %v", err)
		return
	}
	newJSON, err := auditMarshal(p.NewData)
	if err != nil {
		log.Printf("audit: marshal new_data: %v", err)
		return
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_logs (
			company_id, user_id, employee_id,
			action, entity_type, entity_id,
			old_data, new_data,
			ip_address, user_agent
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid,
			$4, $5, $6::uuid,
			$7::jsonb, $8::jsonb,
			$9, $10
		)
	`,
		p.CompanyID,
		p.UserID,
		p.EmployeeID,
		p.Action,
		p.EntityType,
		p.EntityID,
		oldJSON,
		newJSON,
		auditNullString(p.IPAddress),
		auditNullString(p.UserAgent),
	)
	if err != nil {
		log.Printf("audit: insert failed (action=%s): %v", p.Action, err)
	}
}

func auditMarshal(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func auditNullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
