package service

import (
	"context"
	"errors"
	"strings"

	"bank-loan-mvp/internal/model"
	"bank-loan-mvp/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type LoanService struct {
	repo     *repository.Repository
	validate *validator.Validate
}

func NewLoanService(repo *repository.Repository, validate *validator.Validate) *LoanService {
	return &LoanService{repo: repo, validate: validate}
}

func (s *LoanService) Create(ctx context.Context, userID uuid.UUID, req model.CreateLoanRequest) (model.LoanApplication, error) {
	if err := s.validate.Struct(req); err != nil {
		return model.LoanApplication{}, err
	}
	return s.repo.CreateLoanApplication(ctx, userID, req.Amount, req.TermMonths, strings.TrimSpace(req.Purpose))
}

func (s *LoanService) ListOwn(ctx context.Context, userID uuid.UUID, page, perPage int) ([]model.LoanApplication, int, error) {
	offset := (page - 1) * perPage
	return s.repo.ListOwnLoanApplications(ctx, userID, perPage, offset)
}

func (s *LoanService) GetOwn(ctx context.Context, loanID, userID uuid.UUID) (model.LoanApplication, error) {
	loan, err := s.repo.GetOwnLoanApplicationByID(ctx, loanID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.LoanApplication{}, ErrNotFound
		}
		return model.LoanApplication{}, err
	}
	return loan, nil
}

func (s *LoanService) ListAll(ctx context.Context, status *string, page, perPage int) ([]model.LoanApplication, int, error) {
	offset := (page - 1) * perPage
	return s.repo.ListAllLoanApplications(ctx, status, perPage, offset)
}

func (s *LoanService) GetAny(ctx context.Context, loanID uuid.UUID) (model.LoanApplication, error) {
	loan, err := s.repo.GetLoanApplicationByID(ctx, loanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.LoanApplication{}, ErrNotFound
		}
		return model.LoanApplication{}, err
	}
	return loan, nil
}

func (s *LoanService) Cancel(ctx context.Context, loanID, userID uuid.UUID) error {
	if err := s.repo.CancelLoanApplication(ctx, loanID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *LoanService) Decide(ctx context.Context, loanID, managerID uuid.UUID, req model.DecideLoanRequest) error {
	if err := s.validate.Struct(req); err != nil {
		return err
	}
	if err := s.repo.CreateLoanDecisionAndUpdateStatus(ctx, loanID, managerID, req.Decision, strings.TrimSpace(req.Comment)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
