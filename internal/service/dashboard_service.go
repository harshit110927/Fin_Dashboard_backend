package service

import (
	"time"

	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/repository"
	"finance-dashboard/pkg/apperr"
)

type DashboardService struct {
	repo repository.DashboardRepo
}

func NewDashboardService(repo repository.DashboardRepo) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) GetSummary(from, to string) (*domain.DashboardSummary, error) {
	if from == "" || to == "" {
		now := time.Now()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		to = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}

	income, expense, err := s.repo.GetSummary(from, to)
	if err != nil {
		return nil, err
	}

	summary := &domain.DashboardSummary{
		TotalIncome:   income,
		TotalExpenses: expense,
		NetBalance:    income - expense,
	}
	// net_balance is derived in the service layer, not the DB, so it is
	// always exactly income - expenses with no floating point accumulation.
	summary.NetBalance = summary.TotalIncome - summary.TotalExpenses
	summary.Period.From = from
	summary.Period.To = to
	return summary, nil
}

func (s *DashboardService) GetTrends() ([]domain.MonthlyTrend, error) {
	return s.repo.GetMonthlyTrends()
}

func (s *DashboardService) GetCategoryBreakdown() (*domain.CategoryBreakdown, error) {
	totals, err := s.repo.GetCategoryTotals()
	if err != nil {
		return nil, err
	}

	breakdown := &domain.CategoryBreakdown{
		Income:   []domain.CategoryTotal{},
		Expenses: []domain.CategoryTotal{},
	}
	for _, t := range totals {
		switch t.Type {
		case "income":
			breakdown.Income = append(breakdown.Income, t)
		case "expense":
			breakdown.Expenses = append(breakdown.Expenses, t)
		}
	}
	return breakdown, nil
}

func (s *DashboardService) GetRecentActivity(limit int) ([]domain.FinancialRecord, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 50 {
		return nil, apperr.ErrLimitExceeded
	}
	return s.repo.GetRecentActivity(limit)
}

func (s *DashboardService) GetCategories() ([]domain.Category, error) {
	return s.repo.GetCategories()
}

func (s *DashboardService) CreateCategory(req *domain.CreateCategoryRequest) (*domain.Category, error) {
	return s.repo.CreateCategory(req.Name, req.Type)
}

func (s *DashboardService) UpdateCategory(id int, req *domain.UpdateCategoryRequest) (*domain.Category, error) {
	return s.repo.UpdateCategory(id, req)
}
