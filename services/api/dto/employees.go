package dto

import "time"

// ── Employee responses ────────────────────────────────────────────────────────

type EmployeeListItem struct {
	ID           string    `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Role         string    `json:"role"`
	Email        *string   `json:"email"`
	Phone        *string   `json:"phone"`
	Active       bool      `json:"active"`
	SupervisorID *string   `json:"supervisor_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type SupervisorRef struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

type EmployeeDetail struct {
	ID           string         `json:"id"`
	CompanyID    string         `json:"company_id"`
	FirstName    string         `json:"first_name"`
	LastName     string         `json:"last_name"`
	Role         string         `json:"role"`
	Email        *string        `json:"email"`
	Phone        *string        `json:"phone"`
	Active       bool           `json:"active"`
	SupervisorID *string        `json:"supervisor_id"`
	Supervisor   *SupervisorRef `json:"supervisor"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ── Employee requests ─────────────────────────────────────────────────────────

type CreateEmployeeRequest struct {
	FirstName    string  `json:"first_name" binding:"required"`
	LastName     string  `json:"last_name" binding:"required"`
	Role         string  `json:"role" binding:"required"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	SupervisorID *string `json:"supervisor_id"`
}

type UpdateEmployeeRequest struct {
	FirstName    string  `json:"first_name" binding:"required"`
	LastName     string  `json:"last_name" binding:"required"`
	Role         string  `json:"role" binding:"required"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	SupervisorID *string `json:"supervisor_id"`
}

// ── Asset responses ───────────────────────────────────────────────────────────

type AssetListItem struct {
	ID           string    `json:"id"`
	EmployeeID   string    `json:"employee_id"`
	AssetType    string    `json:"asset_type"`
	Name         string    `json:"name"`
	Quantity     float64   `json:"quantity"`
	Unit         *string   `json:"unit"`
	SerialNumber *string   `json:"serial_number"`
	Notes        *string   `json:"notes"`
	Active       bool      `json:"active"`
	AssignedAt   time.Time `json:"assigned_at"`
}

// ── Asset requests ────────────────────────────────────────────────────────────

type CreateAssetRequest struct {
	AssetType    string  `json:"asset_type" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	Quantity     float64 `json:"quantity" binding:"required"`
	Unit         *string `json:"unit"`
	SerialNumber *string `json:"serial_number"`
	Notes        *string `json:"notes"`
}

type UpdateAssetRequest struct {
	AssetType    string  `json:"asset_type" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	Quantity     float64 `json:"quantity" binding:"required"`
	Unit         *string `json:"unit"`
	SerialNumber *string `json:"serial_number"`
	Notes        *string `json:"notes"`
}
