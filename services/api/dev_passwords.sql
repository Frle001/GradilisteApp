-- Dev seed: update password hashes
-- Password: password123  |  bcrypt cost: 10
-- DO NOT use in production

UPDATE users SET password_hash = '$2a$10$HHe.fcnqdg1M7rU0plAak.cPurd86.Azf68RJCMPwSTY6FGdXXd8C' WHERE email = 'direktor@example.com';
UPDATE users SET password_hash = '$2a$10$KmBPm5gcsMLYgNIo0N73HOGpwFNXDLpzBiotbz8nQwL3dYQGTNld2' WHERE email = 'inzenjer@example.com';
UPDATE users SET password_hash = '$2a$10$73RxOWA65iE/ATpAdl/u4O7Ho5C.vP9TLTrp8eOk0RO5RnhcAb2iO' WHERE email = 'admin@example.com';
UPDATE users SET password_hash = '$2a$10$vaqulKJgM3f42kEkSVuLwOlJLNt01KZAS0mryCIKsrUhBU0.ni9Im' WHERE email = 'poslovoda@example.com';

-- Done. Seed users can now login with password: password123
