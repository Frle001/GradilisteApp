-- Initial schema for Gradilište App
-- Creates core tables for users, employees, and projects

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table (for login/auth)
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    password_hash VARCHAR(255),
    role VARCHAR(50) NOT NULL CHECK (role IN ('direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik')),
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Employees table (employee records)
CREATE TABLE IF NOT EXISTS employees (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik')),
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Projects table (gradilišta)
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Junction table: Project assignments (Poslovođa to Projects)
CREATE TABLE IF NOT EXISTS project_assignments (
    id SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    employee_id INTEGER NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'poslovoda',
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, employee_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);
CREATE INDEX IF NOT EXISTS idx_employees_role ON employees(role);
CREATE INDEX IF NOT EXISTS idx_employees_active ON employees(active);
CREATE INDEX IF NOT EXISTS idx_projects_active ON projects(active);
CREATE INDEX IF NOT EXISTS idx_project_assignments_project ON project_assignments(project_id);
CREATE INDEX IF NOT EXISTS idx_project_assignments_employee ON project_assignments(employee_id);
