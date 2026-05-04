
INSERT INTO users (id, email, password_hash, role, full_name) VALUES
('11111111-1111-1111-1111-111111111111', 'admin@bank.local', '$2a$12$PHlc2lXyYSkp2LAUhzzY1egbH4Nbo1L.klK3J0fCcwEKNq9MmvN/C', 'admin', 'Admin User'),
('22222222-2222-2222-2222-222222222222', 'manager@bank.local', '$2a$12$niP9Z2jXQktJCIdOU1.GmumpjrDxZteVe6Ms4z0bBY0upO2hAEUtC', 'manager', 'Manager User'),
('33333333-3333-3333-3333-333333333333', 'alice@example.com', '$2a$12$Lijr467BWd9a9qU3aDucLuQ./nsqvlQXB.1QNvKcLVYOvIkiopYqW', 'client', 'Alice Smith'),
('44444444-4444-4444-4444-444444444444', 'bob@example.com', '$2a$12$bsOWxMi0wrMfXswDrQLARusbE9S7VdRjv4d7wub/7cSEI2Z/yHPGm', 'client', 'Bob Jones');

INSERT INTO loan_applications (id, applicant_id, amount, term_months, purpose, status) VALUES
('55555555-5555-5555-5555-555555555555', '33333333-3333-3333-3333-333333333333', 5000.00, 24, 'Home Renovation', 'pending'),
('66666666-6666-6666-6666-666666666666', '33333333-3333-3333-3333-333333333333', 12000.00, 36, 'Car', 'approved'),
('77777777-7777-7777-7777-777777777777', '44444444-4444-4444-4444-444444444444', 20000.00, 48, 'Business', 'rejected');

INSERT INTO loan_decisions (application_id, manager_id, decision, comment) VALUES
('66666666-6666-6666-6666-666666666666', '22222222-2222-2222-2222-222222222222', 'approved', 'Looks good'),
('77777777-7777-7777-7777-777777777777', '22222222-2222-2222-2222-222222222222', 'rejected', 'Too risky');
