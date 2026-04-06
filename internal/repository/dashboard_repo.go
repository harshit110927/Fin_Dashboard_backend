package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"finance-dashboard/internal/domain"
	"finance-dashboard/pkg/dberr"
)

type DashboardRepository struct {
	db *sqlx.DB
}

func NewDashboardRepository(db *sqlx.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) GetSummary(from, to string) (float64, float64, error) {
	var income, expense float64

	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM financial_records
		WHERE type = 'income' AND status = 'active' AND deleted_at IS NULL AND date >= $1 AND date <= $2`,
		from, to,
	).Scan(&income)
	if err != nil {
		return 0, 0, err
	}

	err = r.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM financial_records
		WHERE type = 'expense' AND status = 'active' AND deleted_at IS NULL AND date >= $1 AND date <= $2`,
		from, to,
	).Scan(&expense)
	if err != nil {
		return 0, 0, err
	}

	return income, expense, dberr.Translate(nil)
}

func (r *DashboardRepository) GetMonthlyTrends() ([]domain.MonthlyTrend, error) {
	rows, err := r.db.Queryx(`
		SELECT TO_CHAR(month, 'YYYY-MM') AS month, type, total
		FROM mvw_monthly_summary
		ORDER BY month DESC`)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	defer rows.Close()

	trendMap := make(map[string]*domain.MonthlyTrend)
	var order []string

	for rows.Next() {
		var month, typ string
		var total float64
		if err := rows.Scan(&month, &typ, &total); err != nil {
			return nil, dberr.Translate(err)
		}
		if _, ok := trendMap[month]; !ok {
			trendMap[month] = &domain.MonthlyTrend{Month: month}
			order = append(order, month)
		}
		switch typ {
		case "income":
			trendMap[month].Income = total
		case "expense":
			trendMap[month].Expenses = total
		}
	}

	result := make([]domain.MonthlyTrend, 0, len(order))
	for _, m := range order {
		t := trendMap[m]
		t.Net = t.Income - t.Expenses
		result = append(result, *t)
	}
	return result, dberr.Translate(nil)
}

func (r *DashboardRepository) GetCategoryTotals() ([]domain.CategoryTotal, error) {
	rows, err := r.db.Queryx(`
		SELECT category_id, category_name, type, total, record_count
		FROM mvw_category_totals
		ORDER BY type, total DESC`)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	defer rows.Close()

	var result []domain.CategoryTotal
	for rows.Next() {
		var ct domain.CategoryTotal
		if err := rows.Scan(&ct.CategoryID, &ct.CategoryName, &ct.Type, &ct.Total, &ct.RecordCount); err != nil {
			return nil, dberr.Translate(err)
		}
		result = append(result, ct)
	}
	return result, dberr.Translate(nil)
}

func (r *DashboardRepository) GetRecentActivity(limit int) ([]domain.FinancialRecord, error) {
	rows, err := r.db.Queryx(`
		SELECT id, amount, type, status, date, description, created_at, updated_at,
		    created_by_id, category_name, category_id, created_by_name
		FROM vw_record_details
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, dberr.Translate(err)
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
			return nil, dberr.Translate(err)
		}
		records = append(records, rec)
	}
	if records == nil {
		records = []domain.FinancialRecord{}
	}
	return records, dberr.Translate(nil)
}

func (r *DashboardRepository) GetCategories() ([]domain.Category, error) {
	rows, err := r.db.Queryx(`
		SELECT id, name, type, is_active FROM categories
		WHERE is_active = true ORDER BY type, name`)
	if err != nil {
		return nil, dberr.Translate(err)
	}
	defer rows.Close()

	var cats []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.IsActive); err != nil {
			return nil, dberr.Translate(err)
		}
		cats = append(cats, c)
	}
	if cats == nil {
		cats = []domain.Category{}
	}
	return cats, dberr.Translate(nil)
}

func (r *DashboardRepository) CreateCategory(name, typ string) (*domain.Category, error) {
	var c domain.Category
	err := r.db.QueryRowx(`
		INSERT INTO categories (name, type) VALUES ($1, $2)
		RETURNING id, name, type, is_active`,
		name, typ,
	).Scan(&c.ID, &c.Name, &c.Type, &c.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.Translate(nil)
		}
		return nil, dberr.Translate(err)
	}
	return &c, dberr.Translate(nil)
}

func (r *DashboardRepository) UpdateCategory(id int, req *domain.UpdateCategoryRequest) (*domain.Category, error) {
	if req.Name == nil && req.Type == nil && req.IsActive == nil {
		return r.FindCategoryByID(id)
	}

	query := "UPDATE categories SET updated_at = NOW()"
	args := []interface{}{}
	idx := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", idx)
		args = append(args, *req.Name)
		idx++
	}
	if req.Type != nil {
		query += fmt.Sprintf(", type = $%d", idx)
		args = append(args, *req.Type)
		idx++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", idx)
		args = append(args, *req.IsActive)
		idx++
	}
	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, name, type, is_active", idx)
	args = append(args, id)

	var c domain.Category
	if err := r.db.QueryRowx(query, args...).Scan(&c.ID, &c.Name, &c.Type, &c.IsActive); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.Translate(nil)
		}
		return nil, dberr.Translate(err)
	}
	return &c, dberr.Translate(nil)
}

func (r *DashboardRepository) FindCategoryByID(id int) (*domain.Category, error) {
	var c domain.Category
	err := r.db.QueryRowx(`SELECT id, name, type, is_active FROM categories WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.Type, &c.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.Translate(nil)
		}
		return nil, dberr.Translate(err)
	}
	return &c, dberr.Translate(nil)
}
