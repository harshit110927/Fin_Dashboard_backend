package domain

import "time"

type FinancialRecord struct {
	ID            string    `db:"id"              json:"id"`
	Amount        float64   `db:"amount"          json:"amount"`
	Type          string    `db:"type"            json:"type"`
	CategoryID    int       `db:"category_id"     json:"category_id"`
	CategoryName  string    `db:"category_name"   json:"category_name"`
	Date          string    `db:"date"            json:"date"`
	Description   *string   `db:"description"     json:"description,omitempty"`
	Status        string    `db:"status"          json:"status"`
	CreatedByID   string    `db:"created_by_id"   json:"created_by_id"`
	CreatedByName string    `db:"created_by_name" json:"created_by_name"`
	CreatedAt     time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"      json:"updated_at"`
}

type CreateRecordRequest struct {
	Amount      float64 `json:"amount"      validate:"required,gt=0"`
	Type        string  `json:"type"        validate:"required,oneof=income expense"`
	CategoryID  int     `json:"category_id" validate:"required,gt=0"`
	Date        string  `json:"date"        validate:"required"`
	Description *string `json:"description"`
}

type UpdateRecordRequest struct {
	Amount      *float64 `json:"amount"`
	CategoryID  *int     `json:"category_id"`
	Date        *string  `json:"date"`
	Description *string  `json:"description"`
	Type        *string  `json:"type"`
}

type VoidRecordRequest struct {
	Reason string `json:"reason" validate:"required,min=10"`
}

type RecordFilter struct {
	Type       string
	Status     string
	DateFrom   string
	DateTo     string
	Sort       string
	CategoryID int
	Page       int
	PerPage    int
}
