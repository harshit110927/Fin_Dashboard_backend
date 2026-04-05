// Package repository implements data access using raw SQL via sqlx.
package repository

import (
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"

	"finance-dashboard/pkg/dberr"
)

type AuditRepository struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Log(ctx context.Context, entityType, entityID, action, actorID, ip string, oldData, newData interface{}) error {
	var oldJSON, newJSON []byte
	var err error
	if oldData != nil {
		oldJSON, err = json.Marshal(oldData)
		if err != nil {
			return dberr.Translate(err)
		}
	}
	if newData != nil {
		newJSON, err = json.Marshal(newData)
		if err != nil {
			return dberr.Translate(err)
		}
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (entity_type, entity_id, action, actor_id, old_data, new_data, ip_address)
		VALUES ($1, $2::uuid, $3, $4::uuid, $5, $6, $7::inet)`,
		entityType, entityID, action, actorID, oldJSON, newJSON, ip,
	)
	return dberr.Translate(err)
}

// GetByEntity returns all audit log entries for a specific entity ordered most recent first.
func (r *AuditRepository) GetByEntity(ctx context.Context, entityType, entityID string) ([]AuditEntry, error) {
	entries := make([]AuditEntry, 0)
	err := r.db.SelectContext(ctx, &entries,
		`SELECT id, action, actor_id, old_data, new_data,
		        COALESCE(ip_address::text, '') AS ip_address, created_at
		 FROM audit_logs
		 WHERE entity_type = $1 AND entity_id = $2
		 ORDER BY created_at DESC`,
		entityType, entityID,
	)
	return entries, dberr.Translate(err)
}
