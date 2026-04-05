package service_test

import (
	"context"
	"testing"

	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/repository"
	"finance-dashboard/internal/service"
	"finance-dashboard/pkg/apperr"
)

type mockRecordRepo struct {
	findByIDFn     func(id string) (*domain.FinancialRecord, error)
	updateFn       func(id string, req *domain.UpdateRecordRequest, updatedBy string) error
	voidFn         func(id string) error
	listFn         func(filter domain.RecordFilter) ([]domain.FinancialRecord, int, error)
	voidCallCount  int
	listCallCount  int
	lastListFilter domain.RecordFilter
}

func (m *mockRecordRepo) Create(rec *domain.FinancialRecord) (*domain.FinancialRecord, error) {
	return rec, nil
}

func (m *mockRecordRepo) FindByID(id string) (*domain.FinancialRecord, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, nil
}

func (m *mockRecordRepo) List(filter domain.RecordFilter) ([]domain.FinancialRecord, int, error) {
	m.listCallCount++
	m.lastListFilter = filter
	if m.listFn != nil {
		return m.listFn(filter)
	}
	return []domain.FinancialRecord{}, 0, nil
}

func (m *mockRecordRepo) Update(id string, req *domain.UpdateRecordRequest, updatedBy string) error {
	if m.updateFn != nil {
		return m.updateFn(id, req, updatedBy)
	}
	return nil
}

func (m *mockRecordRepo) SoftDelete(id string) error { return nil }

func (m *mockRecordRepo) Void(id string) error {
	m.voidCallCount++
	if m.voidFn != nil {
		return m.voidFn(id)
	}
	return nil
}

type mockAuditRepo struct{}

func (m *mockAuditRepo) Log(ctx context.Context, entityType, entityID, action, actorID, ip string, oldData, newData interface{}) error {
	return nil
}

func (m *mockAuditRepo) GetByEntity(ctx context.Context, entityType, entityID string) ([]repository.AuditEntry, error) {
	return []repository.AuditEntry{}, nil
}

type mockDashboardRepo struct{}

func (m *mockDashboardRepo) GetSummary(from, to string) (float64, float64, error) {
	return 0, 0, nil
}

func (m *mockDashboardRepo) GetMonthlyTrends() ([]domain.MonthlyTrend, error) {
	return []domain.MonthlyTrend{}, nil
}

func (m *mockDashboardRepo) GetCategoryTotals() ([]domain.CategoryTotal, error) {
	return []domain.CategoryTotal{}, nil
}

func (m *mockDashboardRepo) GetRecentActivity(limit int) ([]domain.FinancialRecord, error) {
	return []domain.FinancialRecord{}, nil
}

func (m *mockDashboardRepo) GetCategories() ([]domain.Category, error) {
	return []domain.Category{}, nil
}

func (m *mockDashboardRepo) CreateCategory(name, typ string) (*domain.Category, error) {
	return &domain.Category{}, nil
}

func (m *mockDashboardRepo) UpdateCategory(id int, req *domain.UpdateCategoryRequest) (*domain.Category, error) {
	return &domain.Category{}, nil
}

func TestRecordService_Void_AlreadyVoided(t *testing.T) {
	t.Run("returns ALREADY_VOIDED and does not call repo.Void", func(t *testing.T) {
		repo := &mockRecordRepo{
			findByIDFn: func(id string) (*domain.FinancialRecord, error) {
				return &domain.FinancialRecord{ID: id, Status: "void"}, nil
			},
		}
		svc := service.NewRecordService(repo, &mockAuditRepo{})

		_, err := svc.Void("rec-1", "already voided", "actor-1", "127.0.0.1")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		appErr, ok := err.(*apperr.AppError)
		if !ok {
			t.Fatalf("expected *apperr.AppError, got %T", err)
		}
		if appErr.Code != "ALREADY_VOIDED" {
			t.Fatalf("expected code ALREADY_VOIDED, got %s", appErr.Code)
		}
		if repo.voidCallCount != 0 {
			t.Fatalf("expected repo.Void not to be called, got %d", repo.voidCallCount)
		}
	})
}

func TestRecordService_List_InvalidDateRange(t *testing.T) {
	t.Run("returns INVALID_DATE_RANGE and does not call repo.List", func(t *testing.T) {
		repo := &mockRecordRepo{}
		svc := service.NewRecordService(repo, &mockAuditRepo{})

		_, _, err := svc.List(domain.RecordFilter{DateFrom: "2026-04-30", DateTo: "2026-04-01"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		appErr, ok := err.(*apperr.AppError)
		if !ok {
			t.Fatalf("expected *apperr.AppError, got %T", err)
		}
		if appErr.Code != "INVALID_DATE_RANGE" {
			t.Fatalf("expected code INVALID_DATE_RANGE, got %s", appErr.Code)
		}
		if repo.listCallCount != 0 {
			t.Fatalf("expected repo.List not to be called, got %d", repo.listCallCount)
		}
	})
}

func TestRecordService_List_ClampsPerPage(t *testing.T) {
	t.Run("clamps per_page > 100 to default 20", func(t *testing.T) {
		repo := &mockRecordRepo{}
		svc := service.NewRecordService(repo, &mockAuditRepo{})

		_, _, err := svc.List(domain.RecordFilter{PerPage: 999, Page: 1})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if repo.listCallCount != 1 {
			t.Fatalf("expected repo.List to be called once, got %d", repo.listCallCount)
		}
		if repo.lastListFilter.PerPage != 20 {
			t.Fatalf("expected PerPage to be clamped to 20, got %d", repo.lastListFilter.PerPage)
		}
	})
}

func TestRecordService_GetRecentActivity_LimitExceeded(t *testing.T) {
	t.Run("dashboard service returns LIMIT_EXCEEDED for limit > 50", func(t *testing.T) {
		dashSvc := service.NewDashboardService(&mockDashboardRepo{})

		_, err := dashSvc.GetRecentActivity(100)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		appErr, ok := err.(*apperr.AppError)
		if !ok {
			t.Fatalf("expected *apperr.AppError, got %T", err)
		}
		if appErr.Code != "LIMIT_EXCEEDED" {
			t.Fatalf("expected code LIMIT_EXCEEDED, got %s", appErr.Code)
		}
	})
}
