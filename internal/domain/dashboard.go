package domain

type DashboardSummary struct {
	TotalIncome   float64 `json:"total_income"`
	TotalExpenses float64 `json:"total_expenses"`
	NetBalance    float64 `json:"net_balance"`
	Period        struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"period"`
}

type MonthlyTrend struct {
	Month    string  `json:"month"`
	Income   float64 `json:"income"`
	Expenses float64 `json:"expenses"`
	Net      float64 `json:"net"`
}

type CategoryTotal struct {
	CategoryID   int     `db:"category_id"   json:"category_id"`
	CategoryName string  `db:"category_name" json:"category_name"`
	Type         string  `db:"type"          json:"type"`
	Total        float64 `db:"total"         json:"total"`
	RecordCount  int     `db:"record_count"  json:"record_count"`
}

type CategoryBreakdown struct {
	Income   []CategoryTotal `json:"income"`
	Expenses []CategoryTotal `json:"expenses"`
}

type Category struct {
	ID       int    `db:"id"        json:"id"`
	Name     string `db:"name"      json:"name"`
	Type     string `db:"type"      json:"type"`
	IsActive bool   `db:"is_active" json:"is_active"`
}

type CreateCategoryRequest struct {
	Name string `json:"name" validate:"required,min=2,max=80"`
	Type string `json:"type" validate:"required,oneof=income expense"`
}

type UpdateCategoryRequest struct {
	Name     *string `json:"name" validate:"omitempty,min=2,max=80"`
	Type     *string `json:"type" validate:"omitempty,oneof=income expense"`
	IsActive *bool   `json:"is_active"`
}
