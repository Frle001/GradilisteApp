package models

import "time"

// User roles
const (
	RoleDirektor      = "direktor"
	RoleInzenjer      = "inzenjer"
	RoleAdministracija = "administracija"
	RolePoslovoda     = "poslovoda"
	RoleRadnik        = "radnik"
)

// User represents a system user
type User struct {
	ID        int       `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Role      string    `db:"role" json:"role"`
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Employee represents an employee record
type Employee struct {
	ID        int       `db:"id" json:"id"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Role      string    `db:"role" json:"role"`
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Project represents a construction project
type Project struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
