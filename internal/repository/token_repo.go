package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type RefreshToken struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	Revoked   bool      `db:"revoked"`
	CreatedAt time.Time `db:"created_at"`
}

type TokenRepository struct {
	db *sqlx.DB
}

func NewTokenRepository(db *sqlx.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) Store(userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *TokenRepository) FindByHash(hash string) (*RefreshToken, error) {
	var t RefreshToken
	err := r.db.QueryRowx(`
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens WHERE token_hash = $1`,
		hash,
	).StructScan(&t)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *TokenRepository) Revoke(hash string) error {
	_, err := r.db.Exec(`UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`, hash)
	return err
}

func (r *TokenRepository) CleanExpired() error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE expires_at < NOW() OR revoked = true`)
	return err
}
