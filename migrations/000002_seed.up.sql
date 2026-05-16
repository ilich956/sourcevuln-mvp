
INSERT INTO users (id, email, password_hash, role, full_name) VALUES
('11111111-1111-1111-1111-111111111111', 'admin@bank.local', '$2a$12$HeNPmoMPwirMSH3xmTwgKufvy7k3mqT0SSt1C84sgZJq93lTNINM2', 'admin', 'Admin User'),
('22222222-2222-2222-2222-222222222222', 'manager@bank.local', '$2a$12$3R2KmY2SEn.hF2KYCLqjSeEYqltJOo85aFnwYyRW48qP3pfM9/qkC', 'manager', 'Manager User'),
('33333333-3333-3333-3333-333333333333', 'alice@example.com', '$2a$12$A5p5CL0HpFsP2ARZTEY77.m2PZ5fBkCsXuTNeJm0BTjls1H9ideDa', 'client', 'Alice Smith'),
('44444444-4444-4444-4444-444444444444', 'bob@example.com', '$2a$12$wtFhrj94sr5s7XddO78qBe/0/O.BDcX/1KxpUYJxYcyN9UAtaPlQy', 'client', 'Bob Jones');

INSERT INTO loan_applications (id, applicant_id, amount, term_months, purpose, status) VALUES
('55555555-5555-5555-5555-555555555555', '33333333-3333-3333-3333-333333333333', 5000.00, 24, 'Home Renovation', 'pending'),
('66666666-6666-6666-6666-666666666666', '33333333-3333-3333-3333-333333333333', 12000.00, 36, 'Car', 'approved'),
('77777777-7777-7777-7777-777777777777', '44444444-4444-4444-4444-444444444444', 20000.00, 48, 'Business', 'rejected');

INSERT INTO loan_decisions (id, application_id, manager_id, decision, comment) VALUES
('88888888-8888-8888-8888-888888888888', '66666666-6666-6666-6666-666666666666', '22222222-2222-2222-2222-222222222222', 'approved', 'Looks good'),
('99999999-9999-9999-9999-999999999999', '77777777-7777-7777-7777-777777777777', '22222222-2222-2222-2222-222222222222', 'rejected', 'Too risky');
