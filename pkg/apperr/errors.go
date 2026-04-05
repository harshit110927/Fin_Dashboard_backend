// Package apperr defines typed application errors that carry an HTTP status
// code and a machine-readable error code. Services return *AppError values;
// handlers inspect them to produce consistent API error responses without
// any magic strings at the call site.
package apperr

import "net/http"

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *AppError) Error() string { return e.Message }

var (
	ErrNotFound             = &AppError{Code: "NOT_FOUND", Message: "Resource not found", HTTPStatus: http.StatusNotFound}
	ErrUnauthorized         = &AppError{Code: "UNAUTHORIZED", Message: "Authentication required", HTTPStatus: http.StatusUnauthorized}
	ErrForbidden            = &AppError{Code: "INSUFFICIENT_PERMISSIONS", Message: "You do not have permission to perform this action", HTTPStatus: http.StatusForbidden}
	ErrImmutableField       = &AppError{Code: "IMMUTABLE_FIELD", Message: "amount and type cannot be changed after a record is created. Void this record and create a corrected one.", HTTPStatus: http.StatusBadRequest}
	ErrAlreadyVoided        = &AppError{Code: "ALREADY_VOIDED", Message: "This record has already been voided", HTTPStatus: http.StatusBadRequest}
	ErrDuplicateEmail       = &AppError{Code: "DUPLICATE_EMAIL", Message: "This email address is already registered", HTTPStatus: http.StatusBadRequest}
	ErrDuplicateCategory    = &AppError{Code: "DUPLICATE_CATEGORY", Message: "A category with this name already exists", HTTPStatus: http.StatusBadRequest}
	ErrInvalidCredentials   = &AppError{Code: "INVALID_CREDENTIALS", Message: "Email or password is incorrect", HTTPStatus: http.StatusUnauthorized}
	ErrAccountInactive      = &AppError{Code: "ACCOUNT_INACTIVE", Message: "This account has been deactivated", HTTPStatus: http.StatusUnauthorized}
	ErrInvalidReference     = &AppError{Code: "INVALID_REFERENCE", Message: "Referenced resource does not exist", HTTPStatus: http.StatusBadRequest}
	ErrConstraintViolation  = &AppError{Code: "CONSTRAINT_VIOLATION", Message: "Value violates a database constraint", HTTPStatus: http.StatusBadRequest}
	ErrBootstrapUnavailable = &AppError{Code: "BOOTSTRAP_UNAVAILABLE", Message: "Bootstrap is only available when the user table is empty", HTTPStatus: http.StatusForbidden}
	ErrRateLimitExceeded    = &AppError{Code: "RATE_LIMIT_EXCEEDED", Message: "Too many requests. Try again in a moment.", HTTPStatus: http.StatusTooManyRequests}
	ErrInvalidDateRange     = &AppError{Code: "INVALID_DATE_RANGE", Message: "date_from cannot be after date_to", HTTPStatus: http.StatusBadRequest}
	ErrLimitExceeded        = &AppError{Code: "LIMIT_EXCEEDED", Message: "limit cannot exceed 50", HTTPStatus: http.StatusBadRequest}
)
