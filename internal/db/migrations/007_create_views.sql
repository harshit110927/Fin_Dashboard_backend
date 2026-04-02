CREATE OR REPLACE VIEW vw_record_details AS
SELECT fr.id,fr.amount,fr.type,fr.status,fr.date,fr.description,fr.created_at,fr.updated_at,
    fr.created_by AS created_by_id,c.name AS category_name,c.id AS category_id,u.name AS created_by_name
FROM financial_records fr
JOIN categories c ON c.id=fr.category_id JOIN users u ON u.id=fr.created_by
WHERE fr.deleted_at IS NULL;

CREATE MATERIALIZED VIEW IF NOT EXISTS mvw_monthly_summary AS
SELECT DATE_TRUNC('month',date) AS month,type,SUM(amount) AS total,COUNT(*) AS record_count
FROM financial_records WHERE deleted_at IS NULL AND status='active'
GROUP BY DATE_TRUNC('month',date),type ORDER BY month DESC;
CREATE UNIQUE INDEX IF NOT EXISTS idx_mvw_monthly ON mvw_monthly_summary(month,type);

CREATE MATERIALIZED VIEW IF NOT EXISTS mvw_category_totals AS
SELECT c.id AS category_id,c.name AS category_name,c.type,
    COALESCE(SUM(fr.amount),0) AS total,COUNT(fr.id) AS record_count
FROM categories c
LEFT JOIN financial_records fr ON fr.category_id=c.id AND fr.deleted_at IS NULL AND fr.status='active'
GROUP BY c.id,c.name,c.type;
CREATE UNIQUE INDEX IF NOT EXISTS idx_mvw_category ON mvw_category_totals(category_id);
