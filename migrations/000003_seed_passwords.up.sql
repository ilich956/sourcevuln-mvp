
-- Set known passwords for predefined demo users.
-- Uses bcrypt via pgcrypto: crypt(password, gen_salt('bf', 12))

UPDATE users
SET password_hash = crypt('AdminPass123!', gen_salt('bf', 12))
WHERE email = 'admin@bank.local';

UPDATE users
SET password_hash = crypt('ManagerPass123!', gen_salt('bf', 12))
WHERE email = 'manager@bank.local';

UPDATE users
SET password_hash = crypt('AlicePass123!', gen_salt('bf', 12))
WHERE email = 'alice@example.com';

UPDATE users
SET password_hash = crypt('BobPass123!', gen_salt('bf', 12))
WHERE email = 'bob@example.com';

