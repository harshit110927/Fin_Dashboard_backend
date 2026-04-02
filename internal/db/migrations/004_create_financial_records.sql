CREATE TABLE IF NOT EXISTS financial_records(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), amount NUMERIC(15,2) NOT NULL CHECK(amount>0),
    type VARCHAR(10) NOT NULL CHECK(type IN('income','expense')),
    category_id SMALLINT NOT NULL REFERENCES categories(id), date DATE NOT NULL, description TEXT,
    status VARCHAR(10) NOT NULL DEFAULT 'active' CHECK(status IN('active','void')),
    created_by UUID NOT NULL REFERENCES users(id), updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), deleted_at TIMESTAMPTZ);
CREATE INDEX IF NOT EXISTS idx_fr_date ON financial_records(date DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_fr_type ON financial_records(type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_fr_category ON financial_records(category_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_fr_type_date ON financial_records(type,date DESC) WHERE deleted_at IS NULL;
