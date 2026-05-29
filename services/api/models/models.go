package models

import "time"

// ── Role constants ────────────────────────────────────────────────────────────

// User roles — only these can log in
const (
	RoleDirektor       = "direktor"
	RoleInzenjer       = "inzenjer"
	RoleAdministracija = "administracija"
	RolePoslovoda      = "poslovoda"
)

// Employee roles — superset of user roles (includes radnik)
const (
	RoleRadnik = "radnik"
)

// Project status values
const (
	ProjectStatusActive   = "active"
	ProjectStatusClosed   = "closed"
	ProjectStatusArchived = "archived"
)

// ── Core types ────────────────────────────────────────────────────────────────

// User is a system user with login credentials.
// PasswordHash is tagged json:"-" so it is never serialized in any response.
type User struct {
	ID            string     `json:"id"`
	CompanyID     string     `json:"company_id"`
	EmployeeID    *string    `json:"employee_id,omitempty"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"`
	Role          string     `json:"role"`
	Active        bool       `json:"active"`
	EmailVerified bool       `json:"email_verified"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Employee is a company employee record. Not all employees have User accounts.
// Radnik role employees typically have no User record.
type Employee struct {
	ID           string    `json:"id"`
	CompanyID    string    `json:"company_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Role         string    `json:"role"`
	SupervisorID *string   `json:"supervisor_id,omitempty"`
	Email        *string   `json:"email,omitempty"`
	Phone        *string   `json:"phone,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Company is the top-level multi-tenant entity. All business data belongs to a company.
type Company struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OIB       *string   `json:"oib,omitempty"`
	Address   *string   `json:"address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ── API response types ────────────────────────────────────────────────────────

// UserResponse is what the API returns for user data. Never includes PasswordHash.
type UserResponse struct {
	ID            string  `json:"id"`
	CompanyID     string  `json:"company_id"`
	EmployeeID    *string `json:"employee_id"`
	Email         string  `json:"email"`
	Role          string  `json:"role"`
	Active        bool    `json:"active"`
	EmailVerified bool    `json:"email_verified"`
}

// EmployeeResponse is the minimal employee data returned in auth responses.
type EmployeeResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}
