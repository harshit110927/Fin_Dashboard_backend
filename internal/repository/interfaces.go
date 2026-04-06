package repository

import (
	"context"
	"time"

	"finance-dashboard/internal/domain"
)

type UserRepo interface {
	Create(name, email, passwordHash string, roleID int) (*domain.UserResponse, error)
	FindByEmail(email string) (*domain.User, error)
	FindByID(id string) (*domain.UserResponse, error)
	List(page, perPage int) ([]domain.UserResponse, int, error)
	Update(id string, name, email *string) error
	UpdateRole(id string, roleID int) error
	UpdateStatus(id string, isActive bool) error
	SoftDelete(id string) error
}

type RecordRepo interface {
	Create(rec *domain.FinancialRecord) (*domain.FinancialRecord, error)
	FindByID(id string) (*domain.FinancialRecord, error)
	List(filter domain.RecordFilter) ([]domain.FinancialRecord, int, error)
	Update(id string, req *domain.UpdateRecordRequest, updatedBy string) error
	SoftDelete(id string) error
	Void(id string) error
}

type AuditRepo interface {
	Log(ctx context.Context, entityType, entityID, action, actorID, ip string, oldData, newData interface{}) error
	GetByEntity(ctx context.Context, entityType, entityID string) ([]AuditEntry, error)
}

type AuditEntry struct {
	ID        int64       `db:"id" json:"id"`
	Action    string      `db:"action" json:"action"`
	ActorID   string      `db:"actor_id" json:"actor_id"`
	OldData   interface{} `db:"old_data" json:"old_data"`
	NewData   interface{} `db:"new_data" json:"new_data"`
	IPAddress string      `db:"ip_address" json:"ip_address"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
}

type TokenRepo interface {
	Store(userID, tokenHash string, expiresAt time.Time) error
	FindByHash(tokenHash string) (*RefreshToken, error)
	Revoke(tokenHash string) error
	CleanExpired() error
}

type DashboardRepo interface {
	GetSummary(from, to string) (float64, float64, error)
	GetMonthlyTrends() ([]domain.MonthlyTrend, error)
	GetCategoryTotals() ([]domain.CategoryTotal, error)
	GetRecentActivity(limit int) ([]domain.FinancialRecord, error)
	GetCategories() ([]domain.Category, error)
	CreateCategory(name, typ string) (*domain.Category, error)
	UpdateCategory(id int, req *domain.UpdateCategoryRequest) (*domain.Category, error)
}

var (
	_ UserRepo      = (*UserRepository)(nil)
	_ RecordRepo    = (*RecordRepository)(nil)
	_ AuditRepo     = (*AuditRepository)(nil)
	_ TokenRepo     = (*TokenRepository)(nil)
	_ DashboardRepo = (*DashboardRepository)(nil)
)
