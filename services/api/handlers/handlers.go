package handlers

// This package contains HTTP request handlers for all API endpoints.
// Handlers parse requests, validate input, call services, and return responses.
//
// Structure:
// - Each domain (auth, employees, projects, etc.) may have its own file
// - Handlers should be thin - delegate business logic to services
// - Use consistent response formats
