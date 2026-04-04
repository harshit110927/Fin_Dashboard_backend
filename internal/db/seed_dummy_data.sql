-- Dummy users with known passwords:
-- admin@fin.local   / Admin@12345
-- analyst@fin.local / Analyst@12345
-- viewer@fin.local  / Viewer@12345
-- This file is idempotent and safe to run multiple times.

INSERT INTO users (id, name, email, password_hash, role_id, is_active)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'Admin User', 'admin@fin.local',   '$2a$12$HFYeSMMouDNZybzaIAlvqu4ruE1E6Bat.U4am1/f4jD4VWVJKDAU2', 3, true),
  ('22222222-2222-2222-2222-222222222222', 'Analyst User', 'analyst@fin.local', '$2a$12$Ds.Ebn7n4OUJlrresGyR.O2nLcVMyR.mmgz87LSuAAdz6Z/928PpO', 2, true),
  ('33333333-3333-3333-3333-333333333333', 'Viewer User', 'viewer@fin.local',  '$2a$12$4Fhvw6t2u2fjUZ4LHiVa1u2ci42JSyD8PTQWVba8EUoIytV2QfvBW', 1, true)
ON CONFLICT (email) DO NOTHING;

INSERT INTO financial_records (id, amount, type, category_id, date, description, status, created_by, updated_by)
VALUES
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1', 5000.00, 'income', 1, CURRENT_DATE - INTERVAL '10 day', 'Monthly salary', 'active', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2', 1200.00, 'income', 2, CURRENT_DATE - INTERVAL '6 day', 'Freelance design', 'active', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3', 1500.00, 'expense', 4, CURRENT_DATE - INTERVAL '5 day', 'Office rent', 'active', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4', 300.00,  'expense', 6, CURRENT_DATE - INTERVAL '3 day', 'Utilities bill', 'active', '22222222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa5', 250.00,  'expense', 8, CURRENT_DATE - INTERVAL '2 day', 'Client travel', 'void',   '22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa6', 820.00,  'income', 3, CURRENT_DATE - INTERVAL '35 day', 'Dividend payout', 'active', '22222222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa7', 560.00,  'expense', 9, CURRENT_DATE - INTERVAL '33 day', 'Annual software subscription', 'active', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa8', 2200.00, 'income', 1, CURRENT_DATE - INTERVAL '65 day', 'Salary (previous cycle)', 'active', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa9', 900.00,  'expense', 7, CURRENT_DATE - INTERVAL '62 day', 'Payroll adjustment', 'active', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111')
ON CONFLICT (id) DO NOTHING;

REFRESH MATERIALIZED VIEW CONCURRENTLY mvw_monthly_summary;
REFRESH MATERIALIZED VIEW CONCURRENTLY mvw_category_totals;
