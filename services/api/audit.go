package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditLogParams carries all fields for a single audit log entry.
// Use nil pointers for optional fields that are not relevant to the action.
type AuditLogParams struct {
	CompanyID  string
	UserID     *string
	EmployeeID *string
	Action     string  // e.g. "auth.login", "project.create", "asset.transfer"
	EntityType string  // e.g. "user", "project", "employee"
	EntityID   *string // UUID of the affected record
	OldData    interface{} // serialized to JSON; nil if not applicable
	NewData    interface{} // serialized to JSON; nil if not applicable
	IPAddress  string
	UserAgent  string
}

// CreateAuditLog inserts a record into audit_logs.
// Safe to call in a goroutine — errors are logged, never returned.
func CreateAuditLog(ctx context.Context, db *pgxpool.Pool, p AuditLogParams) {
	oldJSON, err := toJSONBytesOrNil(p.OldData)
	if err != nil {
		log.Printf("audit: failed to marshal old_data: %v", err)
		return
	}
	newJSON, err := toJSONBytesOrNil(p.NewData)
	if err != nil {
		log.Printf("audit: failed to marshal new_data: %v", err)
		return
	}

	var entityIDParam interface{}
	if p.EntityID != nil {
		entityIDParam = *p.EntityID + "::uuid"
	}
	_ = entityIDParam // used below via $8::uuid

	_, err = db.Exec(ctx, `
		INSERT INTO audit_logs (
			company_id, user_id, employee_id,
			action, entity_type, entity_id,
			old_data, new_data,
			ip_address, user_agent
		) VALUES (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			$4, $5,
			$6::uuid,
			$7::jsonb,
			$8::jsonb,
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
		nullableString(p.IPAddress),
		nullableString(p.UserAgent),
	)
	if err != nil {
		log.Printf("audit: failed to insert audit log (action=%s): %v", p.Action, err)
	}
}

func toJSONBytesOrNil(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
