package repositories

// This package contains data access logic using pgx.
// Each repository handles a specific entity or domain.
//
// Structure:
// - Each entity (User, Employee, Project, etc.) gets a repository
// - Repositories use prepared statements and pgx directly (sqlc-ready)
// - No business logic in repositories
// - Methods return data models, not business models
//
// Future:
// - Can migrate to sqlc-generated code (sqlc.yaml + queries/)
