-- Migration 011: Add 'radnik' to the users.role check constraint
-- Previously, the constraint only allowed direktor/inzenjer/administracija/poslovoda.
-- Radnik users can now have login accounts.

ALTER TABLE users DROP CONSTRAINT users_role_check;

ALTER TABLE users
  ADD CONSTRAINT users_role_check
  CHECK (role = ANY (ARRAY['direktor'::text, 'inzenjer'::text, 'administracija'::text, 'poslovoda'::text, 'radnik'::text]));
