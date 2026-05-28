package services

// This package contains business logic for the application.
// Services handle domain-specific operations and orchestrate repository calls.
//
// Structure:
// - Each domain (auth, employees, projects, etc.) gets a service
// - Services are called by handlers
// - Services delegate data access to repositories
// - No HTTP concerns in services
