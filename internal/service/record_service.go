package service

import (
	"context"

	"github.com/google/uuid"

	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/repository"
	"finance-dashboard/pkg/apperr"
)

type RecordService struct {
	repo      repository.RecordRepo
	auditRepo repository.AuditRepo
}

func NewRecordService(repo repository.RecordRepo, auditRepo repository.AuditRepo) *RecordService {
	return &RecordService{repo: repo, auditRepo: auditRepo}
}

func (s *RecordService) Create(req *domain.CreateRecordRequest, actorID, ip string) (*domain.FinancialRecord, error) {
	rec := &domain.FinancialRecord{
		ID:          uuid.New().String(),
		Amount:      req.Amount,
		Type:        req.Type,
		CategoryID:  req.CategoryID,
		Date:        req.Date,
		Description: req.Description,
		CreatedByID: actorID,
	}
	created, err := s.repo.Create(rec)
	if err != nil {
		return nil, err
	}
	_ = s.auditRepo.Log(context.Background(), "financial_record", created.ID, "CREATE", actorID, ip, nil, created)
	return created, nil
}

func (s *RecordService) GetByID(id string) (*domain.FinancialRecord, error) {
	return s.repo.FindByID(id)
}

func (s *RecordService) List(filter domain.RecordFilter) ([]domain.FinancialRecord, int, error) {
	if filter.DateFrom != "" && filter.DateTo != "" && filter.DateFrom > filter.DateTo {
		return nil, 0, apperr.ErrInvalidDateRange
	}
	if filter.PerPage <= 0 || filter.PerPage > 100 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	return s.repo.List(filter)
}

func (s *RecordService) Update(id string, req *domain.UpdateRecordRequest, actorID, ip string) (*domain.FinancialRecord, error) {

	old, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, nil
	}

	if err := s.repo.Update(id, req, actorID); err != nil {
		return nil, err
	}

	updated, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	_ = s.auditRepo.Log(context.Background(), "financial_record", id, "UPDATE", actorID, ip, old, updated)
	return updated, nil
}

func (s *RecordService) Delete(id, actorID, ip string) error {
	old, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if old == nil {
		return nil
	}

	if err := s.repo.SoftDelete(id); err != nil {
		return err
	}
	_ = s.auditRepo.Log(context.Background(), "financial_record", id, "DELETE", actorID, ip, old, nil)
	return nil
}

func (s *RecordService) Void(id, reason, actorID, ip string) (*domain.FinancialRecord, error) {
	rec, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, apperr.ErrNotFound
	}
	if rec.Status == "void" {
		return nil, apperr.ErrAlreadyVoided
	}

	if err := s.repo.Void(id); err != nil {
		return nil, err
	}
	_ = s.auditRepo.Log(context.Background(), "financial_record", id, "VOID", actorID, ip, rec, map[string]string{"reason": reason})

	return s.repo.FindByID(id)
}
