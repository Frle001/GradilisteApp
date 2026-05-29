-- Dev seed: update password hashes
-- Password: password123  |  bcrypt cost: 10
-- DO NOT use in production

UPDATE users SET password_hash = '$2a$10$3oN2m1yopEMJAEmLt0QljOV6HHvTUT4CbJHJkjG0bf3LMu4aZsVAq' WHERE email = 'direktor@example.com';
UPDATE users SET password_hash = '$2a$10$U3WNNkoulU.E8u14XSKMh.xI5fla7cfqfmnXT39DYN.ZwCeo8q3Va' WHERE email = 'inzenjer@example.com';
UPDATE users SET password_hash = '$2a$10$YZYIg8oL6l7eK6hyZkjw..k6C.3QbNDcAoAwMV6hd4sjh3RxORyky' WHERE email = 'admin@example.com';
UPDATE users SET password_hash = '$2a$10$sonilnuC5Xw0QZmdWOUaVuaECHFBbC7DKq7YuIXRPH3SAVaGCxdYW' WHERE email = 'poslovoda@example.com';

-- Done. Seed users can now login with password: password123
