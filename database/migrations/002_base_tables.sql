-- Phase 2 Migration 2: Base tables (companies, users, employees)
-- Multi-company setup with user/employee separation

-- Companies table
CREATE TABLE IF NOT EXISTS companies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  oib TEXT,
  address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE companies IS 'Multi-tenant company/organization records';
COMMENT ON COLUMN companies.oib IS 'Croatian business identification number';

-- Employees table
-- Important: Employees are separate from Users. Not all employees have login accounts.
CREATE TABLE IF NOT EXISTS employees (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  role TEXT NOT NULL,
  supervisor_id UUID REFERENCES employees(id) ON DELETE SET NULL,
  email TEXT,
  phone TEXT,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (role IN ('direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik'))
);

COMMENT ON TABLE employees IS 'Employee records for everyone at the company (radnik does not login)';
COMMENT ON COLUMN employees.role IS 'Employee role: direktor, inzenjer, administracija, poslovoda, radnik';
COMMENT ON COLUMN employees.supervisor_id IS 'Manager/supervisor; poslovoda supervises radnik';

CREATE INDEX idx_employees_company_id ON employees(company_id);
CREATE INDEX idx_employees_role ON employees(role);
CREATE INDEX idx_employees_active ON employees(active);
CREATE INDEX idx_employees_supervisor_id ON employees(supervisor_id);
CREATE UNIQUE INDEX idx_employees_company_email ON employees(company_id, email) WHERE email IS NOT NULL;

-- Trigger for employees.updated_at
CREATE TRIGGER trg_employees_updated_at
  BEFORE UPDATE ON employees
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Users table
-- Only direktor, inzenjer, administracija, poslovoda can login
-- Maps to an employee record (optional - some users might not be employees yet)
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  employee_id UUID REFERENCES employees(id) ON DELETE SET NULL,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  email_verified BOOLEAN NOT NULL DEFAULT false,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (role IN ('direktor', 'inzenjer', 'administracija', 'poslovoda'))
);

COMMENT ON TABLE users IS 'System users who can login. Must have role one of: direktor, inzenjer, administracija, poslovoda';
COMMENT ON COLUMN users.employee_id IS 'Optional FK to employees table. User might not be an employee (contractor, admin user, etc.)';
COMMENT ON COLUMN users.email_verified IS 'Email verification status for password reset and login';

CREATE INDEX idx_users_company_id ON users(company_id);
CREATE INDEX idx_users_employee_id ON users(employee_id);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(active);
CREATE INDEX idx_users_email ON users(email);

-- Trigger for users.updated_at
CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Trigger for companies.updated_at
CREATE TRIGGER trg_companies_updated_at
  BEFORE UPDATE ON companies
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
