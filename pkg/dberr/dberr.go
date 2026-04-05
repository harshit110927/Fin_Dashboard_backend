// Package dberr translates raw PostgreSQL driver errors into *apperr.AppError values.
package dberr

import (
	"database/sql"
	"errors"

	"finance-dashboard/pkg/apperr"
	"github.com/lib/pq"
)

func Translate(err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return apperr.ErrNotFound
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			switch pqErr.Constraint {
			case "users_email_key":
				return apperr.ErrDuplicateEmail
			case "categories_name_key":
				return apperr.ErrDuplicateCategory
			default:
				return &apperr.AppError{Code: "DUPLICATE_ENTRY", Message: "A record with these values already exists", HTTPStatus: 400}
			}
		case "23503":
			return apperr.ErrInvalidReference
		case "23514":
			return apperr.ErrConstraintViolation
		case "23502":
			return &apperr.AppError{Code: "MISSING_REQUIRED_FIELD", Message: "A required field is missing", HTTPStatus: 400}
		}
	}

	return err
}
