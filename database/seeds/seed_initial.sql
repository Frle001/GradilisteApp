-- Initial seed data for development and testing
-- This file is safe to re-run (uses INSERT ... ON CONFLICT)

-- Disable foreign key constraints temporarily for seed insertion
SET session_replication_role = 'replica';

-- Seed Users
INSERT INTO users (email, first_name, last_name, password_hash, role, active)
VALUES
    ('direktor@example.com', 'Marko', 'Direktor', '$2a$12$placeholder', 'direktor', true),
    ('inzenjer@example.com', 'Jelena', 'Inženjer', '$2a$12$placeholder', 'inzenjer', true),
    ('admin@example.com', 'Miroslav', 'Admin', '$2a$12$placeholder', 'administracija', true),
    ('poslovoda@example.com', 'Dragan', 'Poslovođa', '$2a$12$placeholder', 'poslovoda', true)
ON CONFLICT (email) DO NOTHING;

-- Seed Employees
INSERT INTO employees (first_name, last_name, role, active)
VALUES
    ('Marko', 'Marković', 'direktor', true),
    ('Jelena', 'Jelić', 'inzenjer', true),
    ('Miroslav', 'Mirić', 'administracija', true),
    ('Dragan', 'Dragić', 'poslovoda', true),
    ('Ivan', 'Ivanović', 'radnik', true),
    ('Petar', 'Petrović', 'radnik', true),
    ('Ana', 'Anić', 'radnik', true)
ON CONFLICT DO NOTHING;

-- Seed Projects
INSERT INTO projects (name, description, active)
VALUES
    ('Nove poslovne prostorije', 'Izgradnja novog poslovnog kompleksa u centru grada', true),
    ('Obnova starog objekta', 'Rekonstrukcija i modernizacija starog stambenog objekta', true),
    ('Infrastrukturni radovi', 'Radovi na komunalnoj infrastrukturi', true)
ON CONFLICT DO NOTHING;

-- Seed Project Assignments (Poslovođe na projektima)
INSERT INTO project_assignments (project_id, employee_id, role)
VALUES
    (1, 4, 'poslovoda'),
    (2, 4, 'poslovoda'),
    (3, 4, 'poslovoda')
ON CONFLICT (project_id, employee_id) DO NOTHING;

-- Re-enable foreign key constraints
SET session_replication_role = 'default';
