package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"finance-dashboard/internal/domain"
)

type RecordRepository struct {
	db *sqlx.DB
}

func NewRecordRepository(db *sqlx.DB) *RecordRepository {
	return &RecordRepository{db: db}
}

func (r *RecordRepository) refreshViews() {
	go func() {
		if _, err := r.db.Exec(`REFRESH MATERIALIZED VIEW CONCURRENTLY mvw_monthly_summary`); err != nil {
			log.Printf("refreshViews: failed to refresh mvw_monthly_summary: %v", err)
		}
		if _, err := r.db.Exec(`REFRESH MATERIALIZED VIEW CONCURRENTLY mvw_category_totals`); err != nil {
			log.Printf("refreshViews: failed to refresh mvw_category_totals: %v", err)
		}
	}()
}

func (r *RecordRepository) Create(rec *domain.FinancialRecord) (*domain.FinancialRecord, error) {
	_, err := r.db.Exec(`
		INSERT INTO financial_records (id, amount, type, category_id, date, description, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rec.ID, rec.Amount, rec.Type, rec.CategoryID, rec.Date, rec.Description, rec.CreatedByID,
	)
	if err != nil {
		return nil, err
	}
	r.refreshViews()
	return r.FindByID(rec.ID)
}

func (r *RecordRepository) FindByID(id string) (*domain.FinancialRecord, error) {
	var rec domain.FinancialRecord
	err := r.db.QueryRowx(`
		SELECT id, amount, type, status, date, description, created_at, updated_at,
		    created_by_id, category_name, category_id, created_by_name
		FROM vw_record_details WHERE id = $1`, id,
	).Scan(
		&rec.ID, &rec.Amount, &rec.Type, &rec.Status, &rec.Date, &rec.Description,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.CreatedByID, &rec.CategoryName,
		&rec.CategoryID, &rec.CreatedByName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *RecordRepository) List(filter domain.RecordFilter) ([]domain.FinancialRecord, int, error) {
	conditions := []string{}
	args := []interface{}{}
	idx := 1

	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", idx))
		args = append(args, filter.Type)
		idx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}
	if filter.CategoryID > 0 {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", idx))
		args = append(args, filter.CategoryID)
		idx++
	}
	if filter.DateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("date >= $%d", idx))
		args = append(args, filter.DateFrom)
		idx++
	}
	if filter.DateTo != "" {
		conditions = append(conditions, fmt.Sprintf("date <= $%d", idx))
		args = append(args, filter.DateTo)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE "
		for i, c := range conditions {
			if i > 0 {
				where += " AND "
			}
			where += c
		}
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM vw_record_details %s`, where)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "date DESC"
	switch filter.Sort {
	case "date_asc":
		orderBy = "date ASC"
	case "amount_desc":
		orderBy = "amount DESC"
	case "amount_asc":
		orderBy = "amount ASC"
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	query := fmt.Sprintf(`
		SELECT id, amount, type, status, date, description, created_at, updated_at,
		    created_by_id, category_name, category_id, created_by_name
		FROM vw_record_details %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		where, orderBy, idx, idx+1,
	)
	args = append(args, perPage, offset)

	rows, err := r.db.Queryx(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []domain.FinancialRecord
	for rows.Next() {
		var rec domain.FinancialRecord
		if err := rows.Scan(
			&rec.ID, &rec.Amount, &rec.Type, &rec.Status, &rec.Date, &rec.Description,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.CreatedByID, &rec.CategoryName,
			&rec.CategoryID, &rec.CreatedByName,
		); err != nil {
			return nil, 0, err
		}
		records = append(records, rec)
	}
	if records == nil {
		records = []domain.FinancialRecord{}
	}
	return records, total, nil
}

func (r *RecordRepository) Update(id string, req *domain.UpdateRecordRequest, updatedBy string) error {
	if req.CategoryID == nil && req.Date == nil && req.Description == nil {
		return nil
	}
	query := `UPDATE financial_records SET updated_at = NOW(), updated_by = $1`
	args := []interface{}{updatedBy}
	idx := 2

	if req.CategoryID != nil {
		query += fmt.Sprintf(", category_id = $%d", idx)
		args = append(args, *req.CategoryID)
		idx++
	}
	if req.Date != nil {
		query += fmt.Sprintf(", date = $%d", idx)
		args = append(args, *req.Date)
		idx++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", idx)
		args = append(args, *req.Description)
		idx++
	}
	query += fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", idx)
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}
	r.refreshViews()
	return nil
}

func (r *RecordRepository) SoftDelete(id string) error {
	_, err := r.db.Exec(`
		UPDATE financial_records SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return err
	}
	r.refreshViews()
	return nil
}

func (r *RecordRepository) Void(id string) error {
	_, err := r.db.Exec(`
		UPDATE financial_records SET status = 'void', updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return err
	}
	r.refreshViews()
	return nil
}
