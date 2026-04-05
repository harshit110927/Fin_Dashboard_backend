package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"finance-dashboard/internal/domain"
	"finance-dashboard/pkg/dberr"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func RoleNameToID(name string) (int, error) {
	switch name {
	case "viewer":
		return 1, nil
	case "analyst":
		return 2, nil
	case "admin":
		return 3, nil
	}
	return 0, fmt.Errorf("unknown role: %s", name)
}

func (r *UserRepository) Create(name, email, passwordHash string, roleID int) (*domain.UserResponse, error) {
	var u domain.UserResponse
	err := r.db.QueryRowx(`
		INSERT INTO users (name, email, password_hash, role_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, email,
		    (SELECT name FROM roles WHERE id = role_id) AS role,
		    is_active, created_at`,
		name, email, passwordHash, roleID,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	return &u, dberr.Translate(nil)
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRowx(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id,
		    ro.name AS role_name, u.is_active, u.created_at, u.updated_at, u.deleted_at
		FROM users u
		JOIN roles ro ON ro.id = u.role_id
		WHERE u.email = $1 AND u.deleted_at IS NULL`,
		email,
	).StructScan(&u)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.Translate(nil)
		}
		return nil, dberr.Translate(err)
	}
	return &u, dberr.Translate(nil)
}

func (r *UserRepository) FindByID(id string) (*domain.UserResponse, error) {
	var u domain.UserResponse
	err := r.db.QueryRowx(`
		SELECT u.id, u.name, u.email, ro.name AS role, u.is_active, u.created_at
		FROM users u
		JOIN roles ro ON ro.id = u.role_id
		WHERE u.id = $1 AND u.deleted_at IS NULL`,
		id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.Translate(nil)
		}
		return nil, dberr.Translate(err)
	}
	return &u, dberr.Translate(nil)
}

func (r *UserRepository) List(page, perPage int) ([]domain.UserResponse, int, error) {
	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&total)
	if err != nil {
		return nil, 0, dberr.Translate(err)
	}

	offset := (page - 1) * perPage
	rows, err := r.db.Queryx(`
		SELECT u.id, u.name, u.email, ro.name AS role, u.is_active, u.created_at
		FROM users u
		JOIN roles ro ON ro.id = u.role_id
		WHERE u.deleted_at IS NULL
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2`,
		perPage, offset,
	)
	if err != nil {
		return nil, 0, dberr.Translate(err)
	}
	defer rows.Close()

	var users []domain.UserResponse
	for rows.Next() {
		var u domain.UserResponse
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, 0, dberr.Translate(err)
		}
		users = append(users, u)
	}
	if users == nil {
		users = []domain.UserResponse{}
	}
	return users, total, dberr.Translate(nil)
}

func (r *UserRepository) UpdateRole(id string, roleID int) error {
	_, err := r.db.Exec(`
		UPDATE users SET role_id = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`,
		roleID, id,
	)
	return dberr.Translate(err)
}

func (r *UserRepository) UpdateStatus(id string, isActive bool) error {
	_, err := r.db.Exec(`
		UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`,
		isActive, id,
	)
	return dberr.Translate(err)
}

func (r *UserRepository) Update(id string, name, email *string) error {
	if name == nil && email == nil {
		return nil
	}
	query := `UPDATE users SET updated_at = NOW()`
	args := []interface{}{}
	idx := 1
	if name != nil {
		query += fmt.Sprintf(", name = $%d", idx)
		args = append(args, *name)
		idx++
	}
	if email != nil {
		query += fmt.Sprintf(", email = $%d", idx)
		args = append(args, *email)
		idx++
	}
	query += fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", idx)
	args = append(args, id)
	_, err := r.db.Exec(query, args...)
	return dberr.Translate(err)
}

func (r *UserRepository) SoftDelete(id string) error {
	_, err := r.db.Exec(`
		UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	return dberr.Translate(err)
}
