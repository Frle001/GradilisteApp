// genhash generates bcrypt hashes for development seed users and prints SQL UPDATE statements.
//
// Usage (from services/api/):
//
//	go run ./cmd/genhash/ | psql -h localhost -U gradiliste -d gradiliste
//	go run ./cmd/genhash/ > /tmp/update_passwords.sql
//
// Default password: password123
// Default bcrypt cost: 10 (fast for development)
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "password123"
	if p := os.Getenv("SEED_PASSWORD"); p != "" {
		password = p
	}

	cost := 10
	if c, err := strconv.Atoi(os.Getenv("BCRYPT_COST")); err == nil && c > 0 {
		cost = c
	}

	seedUsers := []struct {
		email string
	}{
		{"direktor@example.com"},
		{"inzenjer@example.com"},
		{"admin@example.com"},
		{"poslovoda@example.com"},
	}

	fmt.Println("-- Dev seed: update password hashes")
	fmt.Printf("-- Password: %s  |  bcrypt cost: %d\n", password, cost)
	fmt.Println("-- DO NOT use in production")
	fmt.Println()

	for _, u := range seedUsers {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
		if err != nil {
			log.Fatalf("failed to hash password for %s: %v", u.email, err)
		}
		fmt.Printf("UPDATE users SET password_hash = '%s' WHERE email = '%s';\n", hash, u.email)
	}

	fmt.Println()
	fmt.Printf("-- Done. Seed users can now login with password: %s\n", password)
}
