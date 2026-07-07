-- Seed data for Phase 2: Test company, users, employees, and projects
-- Safe to re-run: uses INSERT ... ON CONFLICT DO NOTHING

-- Disable foreign key checks temporarily for seed insertion


-- Create test company
INSERT INTO companies (id, name, oib, address)
VALUES (
  '10000000-0000-0000-0000-000000000001'::uuid,
  'Test Construction Company',
  '12345678901',
  'Trg bana Jelačića 1, Zagreb'
)
ON CONFLICT DO NOTHING;

-- Create test employees
INSERT INTO employees (id, company_id, first_name, last_name, role, email, phone, active)
VALUES
  (
    '20000000-0000-0000-0000-000000000001'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Marko',
    'Marković',
    'direktor',
    'direktor@example.com',
    '+385-1-1234-5678',
    true
  ),
  (
    '20000000-0000-0000-0000-000000000002'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Jelena',
    'Jelić',
    'inzenjer',
    'inzenjer@example.com',
    '+385-1-1234-5679',
    true
  ),
  (
    '20000000-0000-0000-0000-000000000003'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Miroslav',
    'Mirić',
    'administracija',
    'admin@example.com',
    '+385-1-1234-5680',
    true
  ),
  (
    '20000000-0000-0000-0000-000000000004'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Dragan',
    'Dragić',
    'poslovoda',
    'poslovoda@example.com',
    '+385-1-1234-5681',
    true
  ),
  (
    '20000000-0000-0000-0000-000000000005'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Ivan',
    'Ivanović',
    'radnik',
    'ivan@example.com',
    '+385-1-1234-5682',
    true
  ),
  (
    '20000000-0000-0000-0000-000000000006'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Petar',
    'Petrović',
    'radnik',
    'petar@example.com',
    '+385-1-1234-5683',
    true
  ),
  (
    '20000000-0000-0000-0000-000000000007'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Ana',
    'Anić',
    'radnik',
    'ana@example.com',
    '+385-1-1234-5684',
    true
  )
ON CONFLICT DO NOTHING;

-- Set supervisor_id: radnik workers are supervised by the poslovoda
UPDATE employees SET supervisor_id = '20000000-0000-0000-0000-000000000004'::uuid
WHERE id IN (
  '20000000-0000-0000-0000-000000000005'::uuid,
  '20000000-0000-0000-0000-000000000006'::uuid,
  '20000000-0000-0000-0000-000000000007'::uuid
);

-- Create test users (direktor, inzenjer, administracija, poslovoda)
-- Password for all: Temp1234!  (bcrypt $2a$10$, cost 10 — valid for any BCRYPT_COST setting)
-- must_change_password=true so user is forced to set a personal password on first login.
INSERT INTO users (id, company_id, employee_id, email, password_hash, role, active, email_verified, must_change_password)
VALUES
  (
    '30000000-0000-0000-0000-000000000001'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000001'::uuid,
    'direktor@example.com',
    '$2a$10$TfnHyv4v6RNT39tNsWvuXO9mSb219/yrBSVSv1bUvkMJvZnoKIxI6',
    'direktor',
    true,
    true,
    true
  ),
  (
    '30000000-0000-0000-0000-000000000002'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000002'::uuid,
    'inzenjer@example.com',
    '$2a$10$TfnHyv4v6RNT39tNsWvuXO9mSb219/yrBSVSv1bUvkMJvZnoKIxI6',
    'inzenjer',
    true,
    true,
    true
  ),
  (
    '30000000-0000-0000-0000-000000000003'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000003'::uuid,
    'admin@example.com',
    '$2a$10$TfnHyv4v6RNT39tNsWvuXO9mSb219/yrBSVSv1bUvkMJvZnoKIxI6',
    'administracija',
    true,
    true,
    true
  ),
  (
    '30000000-0000-0000-0000-000000000004'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000004'::uuid,
    'poslovoda@example.com',
    '$2a$10$TfnHyv4v6RNT39tNsWvuXO9mSb219/yrBSVSv1bUvkMJvZnoKIxI6',
    'poslovoda',
    true,
    true,
    true
  )
ON CONFLICT DO NOTHING;

-- Create test projects
INSERT INTO projects (id, company_id, name, address, description, status, start_date, created_by)
VALUES
  (
    '40000000-0000-0000-0000-000000000001'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Nove poslovne prostorije',
    'Trg bana Jelačića 15, Zagreb',
    'Izgradnja novog poslovnog kompleksa u centru grada',
    'active',
    '2026-03-01'::date,
    '30000000-0000-0000-0000-000000000001'::uuid
  ),
  (
    '40000000-0000-0000-0000-000000000002'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Obnova starog objekta',
    'Ulica A. Šenoe 20, Zagreb',
    'Rekonstrukcija i modernizacija starog stambenog objekta',
    'active',
    '2026-02-15'::date,
    '30000000-0000-0000-0000-000000000001'::uuid
  ),
  (
    '40000000-0000-0000-0000-000000000003'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    'Infrastrukturni radovi',
    'Ul. Mihanovićeva, Zagreb',
    'Radovi na komunalnoj infrastrukturi',
    'active',
    '2026-01-15'::date,
    '30000000-0000-0000-0000-000000000002'::uuid
  )
ON CONFLICT DO NOTHING;

-- Assign poslovoda to projects
INSERT INTO project_assignments (id, company_id, project_id, employee_id, role_on_project, assigned_by)
VALUES
  (
    '50000000-0000-0000-0000-000000000001'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000004'::uuid,
    'poslovoda',
    '30000000-0000-0000-0000-000000000001'::uuid
  ),
  (
    '50000000-0000-0000-0000-000000000002'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000002'::uuid,
    '20000000-0000-0000-0000-000000000004'::uuid,
    'poslovoda',
    '30000000-0000-0000-0000-000000000001'::uuid
  ),
  (
    '50000000-0000-0000-0000-000000000003'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000003'::uuid,
    '20000000-0000-0000-0000-000000000004'::uuid,
    'poslovoda',
    '30000000-0000-0000-0000-000000000002'::uuid
  )
ON CONFLICT DO NOTHING;

-- Assign workers to projects
INSERT INTO project_assignments (id, company_id, project_id, employee_id, role_on_project, assigned_by)
VALUES
  (
    '50000000-0000-0000-0000-000000000010'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000005'::uuid,
    'worker',
    '30000000-0000-0000-0000-000000000001'::uuid
  ),
  (
    '50000000-0000-0000-0000-000000000011'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000006'::uuid,
    'worker',
    '30000000-0000-0000-0000-000000000001'::uuid
  ),
  (
    '50000000-0000-0000-0000-000000000012'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000007'::uuid,
    'worker',
    '30000000-0000-0000-0000-000000000001'::uuid
  )
ON CONFLICT DO NOTHING;

-- Create test project materials
INSERT INTO project_materials (id, company_id, project_id, material_name, material_code, planned_quantity, unit, source)
VALUES
  (
    '60000000-0000-0000-0000-000000000001'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000001'::uuid,
    'Portland cement CEM I 42.5',
    'CEM-001',
    100,
    'vreća 50kg',
    'Lafarge'
  ),
  (
    '60000000-0000-0000-0000-000000000002'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000001'::uuid,
    'Armatura RA 500/560',
    'ARM-001',
    5000,
    'kg',
    'ArcelorMittal'
  ),
  (
    '60000000-0000-0000-0000-000000000003'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '40000000-0000-0000-0000-000000000001'::uuid,
    'Oplata za beton',
    'OPL-001',
    200,
    'komad',
    'Local supplier'
  )
ON CONFLICT DO NOTHING;

-- Create test employee assets
INSERT INTO employee_assets (id, company_id, employee_id, asset_type, name, quantity, unit, serial_number, assigned_by)
VALUES
  (
    '70000000-0000-0000-0000-000000000001'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000004'::uuid,
    'car',
    'Mercedes Sprinter van',
    1,
    'komad',
    'ZG-ABC-123',
    '30000000-0000-0000-0000-000000000001'::uuid
  ),
  (
    '70000000-0000-0000-0000-000000000002'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000005'::uuid,
    'tool',
    'Hammer set',
    3,
    'komad',
    null,
    '30000000-0000-0000-0000-000000000001'::uuid
  ),
  (
    '70000000-0000-0000-0000-000000000003'::uuid,
    '10000000-0000-0000-0000-000000000001'::uuid,
    '20000000-0000-0000-0000-000000000005'::uuid,
    'equipment',
    'Safety harness set',
    5,
    'komad',
    null,
    '30000000-0000-0000-0000-000000000001'::uuid
  )
ON CONFLICT DO NOTHING;

-- Re-enable foreign key constraints

-- Log seed data insertion (optional)
INSERT INTO audit_logs (company_id, action, entity_type, entity_id, new_data, created_at)
VALUES (
  '10000000-0000-0000-0000-000000000001'::uuid,
  'seed',
  'system',
  null,
  jsonb_build_object('seed_type', 'phase2_test_data', 'records_inserted', 'company, employees, users, projects, assignments, materials, assets'),
  CURRENT_TIMESTAMP
)
ON CONFLICT DO NOTHING;
