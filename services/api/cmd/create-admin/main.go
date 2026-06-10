// create-admin bootstraps the first company and admin user in a fresh database.
// Run it once per environment after applying migrations.
//
//	DATABASE_URL=postgres://... go run ./cmd/create-admin
//
// The created user will be forced to change their password on first login
// (must_change_password = true).
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load .env if present; ignore errors (env vars may be set directly)
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Fall back to individual DB_* vars
		host := getenv("DB_HOST", "localhost")
		port := getenv("DB_PORT", "5432")
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASSWORD")
		name := os.Getenv("DB_NAME")
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, name)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("cannot connect to database: %v\n\nSet DATABASE_URL or DB_HOST/DB_USER/DB_PASSWORD/DB_NAME", err)
	}
	defer conn.Close(ctx)

	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	fmt.Println("=== Gradilište — create first admin user ===")
	fmt.Println("This will create a new company and an admin user with must_change_password=true.")
	fmt.Println()

	r := bufio.NewReader(os.Stdin)

	companyName := prompt(r, "Company name (e.g. Gradnja d.o.o.)")
	if companyName == "" {
		log.Fatal("company name is required")
	}

	firstName := prompt(r, "Admin first name")
	if firstName == "" {
		log.Fatal("first name is required")
	}

	lastName := prompt(r, "Admin last name")
	if lastName == "" {
		log.Fatal("last name is required")
	}

	email := prompt(r, "Admin email")
	if !strings.Contains(email, "@") {
		log.Fatal("invalid email address")
	}

	fmt.Println("Role options: direktor | inzenjer | administracija | poslovoda")
	role := prompt(r, "Admin role [direktor]")
	if role == "" {
		role = "direktor"
	}
	validRoles := map[string]bool{"direktor": true, "inzenjer": true, "administracija": true, "poslovoda": true}
	if !validRoles[role] {
		log.Fatalf("invalid role %q", role)
	}

	password := promptPassword(r, "Temporary password (min 12 chars)")
	if len(password) < 12 {
		log.Fatal("password must be at least 12 characters")
	}

	confirm := promptPassword(r, "Confirm password")
	if password != confirm {
		log.Fatal("passwords do not match")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatalf("bcrypt error: %v", err)
	}

	fmt.Printf("\nAbout to create:\n  Company:  %s\n  Employee: %s %s (%s)\n  Email:    %s\n\n", companyName, firstName, lastName, role, email)
	confirm2 := prompt(r, "Confirm? [y/N]")
	if strings.ToLower(confirm2) != "y" {
		fmt.Println("Aborted.")
		os.Exit(0)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var companyID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO companies (name) VALUES ($1) RETURNING id::text`,
		companyName,
	).Scan(&companyID); err != nil {
		log.Fatalf("insert company: %v", err)
	}

	var employeeID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO employees (company_id, first_name, last_name, role, email, active)
		 VALUES ($1, $2, $3, $4, $5, true) RETURNING id::text`,
		companyID, firstName, lastName, role, email,
	).Scan(&employeeID); err != nil {
		log.Fatalf("insert employee: %v", err)
	}

	var userID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO users (company_id, employee_id, email, password_hash, role, active, must_change_password)
		 VALUES ($1, $2, $3, $4, $5, true, true) RETURNING id::text`,
		companyID, employeeID, email, string(hash), role,
	).Scan(&userID); err != nil {
		log.Fatalf("insert user: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Admin user created successfully.")
	fmt.Printf("  Company ID:  %s\n", companyID)
	fmt.Printf("  Employee ID: %s\n", employeeID)
	fmt.Printf("  User ID:     %s\n", userID)
	fmt.Println()
	fmt.Println("The user must change their password on first login.")
}

func prompt(r *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

// promptPassword reads a password from stdin without terminal echo.
// Falls back to plain-text input if the terminal package is unavailable.
func promptPassword(r *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
