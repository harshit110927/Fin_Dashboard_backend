CREATE TABLE IF NOT EXISTS audit_logs(
    id BIGSERIAL PRIMARY KEY, entity_type VARCHAR(50) NOT NULL, entity_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL, actor_id UUID NOT NULL REFERENCES users(id),
    old_data JSONB, new_data JSONB, ip_address INET, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_al_entity ON audit_logs(entity_type,entity_id);
CREATE INDEX IF NOT EXISTS idx_al_actor ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_al_created ON audit_logs(created_at DESC);
