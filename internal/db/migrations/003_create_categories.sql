CREATE TABLE IF NOT EXISTS categories(
    id SMALLSERIAL PRIMARY KEY, name VARCHAR(80) UNIQUE NOT NULL,
    type VARCHAR(10) NOT NULL CHECK(type IN('income','expense')), is_active BOOLEAN NOT NULL DEFAULT true);
INSERT INTO categories(name,type) VALUES('Salary','income'),('Freelance','income'),('Investment','income'),
('Rent','expense'),('Marketing','expense'),('Utilities','expense'),('Salaries','expense'),
('Travel','expense'),('Software','expense') ON CONFLICT DO NOTHING;
