package domain

import "time"

type User struct {
	ID           string     `db:"id"`
	Name         string     `db:"name"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	RoleName     string     `db:"role_name"`
	RoleID       int        `db:"role_id"`
	IsActive     bool       `db:"is_active"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Name     string `json:"name"     validate:"required,min=2,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role"     validate:"required,oneof=viewer analyst admin"`
}

type UpdateUserRequest struct {
	Name  *string `json:"name"  validate:"omitempty,min=2,max=100"`
	Email *string `json:"email" validate:"omitempty,email"`
}

type UpdateRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=viewer analyst admin"`
}

type UpdateStatusRequest struct {
	IsActive bool `json:"is_active"`
}
