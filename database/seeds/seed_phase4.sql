-- seed_phase4.sql
-- Phase 4 login test users
-- Password for all seeded users: Temp1234!
-- Radnik is created as employee only, no login user.

ALTER TABLE users
ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT false;

-- Create demo employees under the first existing company.
-- This avoids hardcoding company_id as integer because companies.id is UUID.

WITH demo_company AS (
  SELECT id AS company_id
  FROM companies
  ORDER BY created_at NULLS LAST, id
  LIMIT 1
)
INSERT INTO employees (
  company_id,
  first_name,
  last_name,
  role,
  active
)
SELECT company_id, 'Demo', 'Direktor', 'direktor', true FROM demo_company
UNION ALL
SELECT company_id, 'Demo', 'Inzenjer', 'inzenjer', true FROM demo_company
UNION ALL
SELECT company_id, 'Demo', 'Administracija', 'administracija', true FROM demo_company
UNION ALL
SELECT company_id, 'Demo', 'Poslovoda', 'poslovoda', true FROM demo_company
UNION ALL
SELECT company_id, 'Demo', 'Radnik', 'radnik', true FROM demo_company
ON CONFLICT DO NOTHING;

-- Login users.
-- Password: Temp1234!
-- Bcrypt hash uses $2a$ prefix for Go compatibility.

INSERT INTO users (
  company_id,
  employee_id,
  email,
  password_hash,
  role,
  active,
  must_change_password
)
SELECT
  e.company_id,
  e.id,
  'direktor@gradiliste.test',
  '$2a$10$TfnHyv4v6RNT39tNsWvuXO9mSb219/yrBSVSv1bUvkMJvZnoKIxI6',
  'direktor',
  true,
  true
FROM employees e
WHERE e.first_name = 'Demo'
  AND e.last_name = 'Direktor'
  AND e.role = 'direktor'
ORDER BY e.created_at DESC
LIMIT 1
ON CONFLICT (email) DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  role = EXCLUDED.role,
  active = true,
  must_change_password = true,
  employee_id = EXCLUDED.employee_id;

INSERT INTO users (
  company_id,
  employee_id,
  email,
  password_hash,
  role,
  active,
  must_change_password
)
SELECT
  e.company_id,
  e.id,
  'inzenjer@gradiliste.test',
  '$2a$10$TfnHyv4v6RNT39tNsWvuXO9mSb219/yrBSVSv1bUvkMJvZnoKIxI6',
  'inzenjer',
  true,
  true
FROM employees e
WHERE e.first_name = 'Demo'
  AND e.last_name = 'Inzenjer'
  AND e.role = 'inzenjer'
ORDER BY e.created_at DESC
LIMIT 1
ON CONFLICT (email) DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  role = EXCLUDED.role,
  active = true,
  must_change_password = true,
  employee_id = EXCLUDED.employee_id;

INSERT INTO users (
  company_id,
  employee_id,
  email,
  password_hash,
  role,
  active,
  must_change_password
)
SELECT
  e.company_id,
  e.id,
  'administracija@gradiliste.test',
  '$2a$10$TfnHyv4v6RNT39tNsWvuXO9mSb219/yrBSVSv1bUvkMJvZnoKIxI6',
  'administracija',
  true,
  true
FROM employees e
WHERE e.first_name = 'Demo'
  AND e.last_name = 'Administracija'
  AND e.role = 'administracija'
ORDER BY e.created_at DESC
LIMIT 1
ON CONFLICT (email) DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  role = EXCLUDED.role,
  active = true,
  must_change_password = true,
  employee_id = EXCLUDED.employee_id;

INSERT INTO users (
  company_id,
  employee_id,
  email,
  password_hash,
  role,
  active,
  must_change_password
)
SELECT
  e.company_id,
  e.id,
  'poslovoda@gradiliste.test',
  '$2a$10$TfnHyv4v6RNT39tNsWvuXO9mSb219/yrBSVSv1bUvkMJvZnoKIxI6',
  'poslovoda',
  true,
  true
FROM employees e
WHERE e.first_name = 'Demo'
  AND e.last_name = 'Poslovoda'
  AND e.role = 'poslovoda'
ORDER BY e.created_at DESC
LIMIT 1
ON CONFLICT (email) DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  role = EXCLUDED.role,
  active = true,
  must_change_password = true,
  employee_id = EXCLUDED.employee_id;