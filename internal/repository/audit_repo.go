package repository

import (
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"
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
			return err
		}
	}
	if newData != nil {
		newJSON, err = json.Marshal(newData)
		if err != nil {
			return err
		}
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (entity_type, entity_id, action, actor_id, old_data, new_data, ip_address)
		VALUES ($1, $2::uuid, $3, $4::uuid, $5, $6, $7::inet)`,
		entityType, entityID, action, actorID, oldJSON, newJSON, ip,
	)
	return err
}
