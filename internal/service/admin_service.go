package service

import (
	"context"
	"errors"

	"bank-loan-mvp/internal/model"
	"bank-loan-mvp/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AdminService struct {
	repo *repository.Repository
}

func NewAdminService(repo *repository.Repository) *AdminService {
	return &AdminService{repo: repo}
}

func (s *AdminService) ListUsers(ctx context.Context, page, perPage int) ([]model.User, int, error) {
	offset := (page - 1) * perPage
	return s.repo.ListUsers(ctx, perPage, offset)
}

func (s *AdminService) SetUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error {
	if err := s.repo.UpdateUserStatus(ctx, userID, isActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *AdminService) SetUserRole(ctx context.Context, userID uuid.UUID, role string) error {
	if err := s.repo.UpdateUserRole(ctx, userID, role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *AdminService) GetStats(ctx context.Context) (model.LoanStats, error) {
	return s.repo.GetLoanStats(ctx)
}
