package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"bank-loan-mvp/internal/model"
	"bank-loan-mvp/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo      *repository.Repository
	validate  *validator.Validate
	jwtSecret []byte
}

func NewAuthService(repo *repository.Repository, validate *validator.Validate, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, validate: validate, jwtSecret: []byte(jwtSecret)}
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *AuthService) newAccessToken(user model.User) (string, int64, error) {
	expiresAt := time.Now().UTC().Add(15 * time.Minute)
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": user.Role,
		"exp":  expiresAt.Unix(),
		"iat":  time.Now().UTC().Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, err
	}
	return signed, int64((15 * time.Minute).Seconds()), nil
}

func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest) (model.User, error) {
	if err := s.validate.Struct(req); err != nil {
		return model.User{}, err
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return model.User{}, err
	}
	u, err := s.repo.CreateUser(ctx, req.Email, string(hash), "client", strings.TrimSpace(req.FullName))
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return model.User{}, ErrConflict
		}
		return model.User{}, err
	}
	return u, nil
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (model.AuthTokensResponse, model.User, error) {
	if err := s.validate.Struct(req); err != nil {
		return model.AuthTokensResponse{}, model.User{}, err
	}
	u, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.AuthTokensResponse{}, model.User{}, ErrUnauthorized
		}
		return model.AuthTokensResponse{}, model.User{}, err
	}
	if !u.IsActive {
		return model.AuthTokensResponse{}, model.User{}, ErrForbidden
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		return model.AuthTokensResponse{}, model.User{}, ErrUnauthorized
	}
	access, expiresIn, err := s.newAccessToken(u)
	if err != nil {
		return model.AuthTokensResponse{}, model.User{}, err
	}
	rawRefresh, err := newOpaqueToken()
	if err != nil {
		return model.AuthTokensResponse{}, model.User{}, err
	}
	if err := s.repo.StoreRefreshToken(ctx, u.ID, hashToken(rawRefresh), time.Now().UTC().Add(7*24*time.Hour)); err != nil {
		return model.AuthTokensResponse{}, model.User{}, err
	}
	return model.AuthTokensResponse{AccessToken: access, RefreshToken: rawRefresh, TokenType: "Bearer", ExpiresIn: expiresIn}, u, nil
}

func (s *AuthService) Refresh(ctx context.Context, req model.RefreshRequest) (model.AuthTokensResponse, model.User, error) {
	if err := s.validate.Struct(req); err != nil {
		return model.AuthTokensResponse{}, model.User{}, err
	}
	u, oldTokenID, err := s.repo.GetActiveRefreshToken(ctx, hashToken(req.RefreshToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.AuthTokensResponse{}, model.User{}, ErrUnauthorized
		}
		return model.AuthTokensResponse{}, model.User{}, err
	}
	if err := s.repo.RevokeRefreshTokenByID(ctx, oldTokenID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.AuthTokensResponse{}, model.User{}, ErrUnauthorized
		}
		return model.AuthTokensResponse{}, model.User{}, err
	}
	newRawRefresh, err := newOpaqueToken()
	if err != nil {
		return model.AuthTokensResponse{}, model.User{}, err
	}
	if err := s.repo.StoreRefreshToken(ctx, u.ID, hashToken(newRawRefresh), time.Now().UTC().Add(7*24*time.Hour)); err != nil {
		return model.AuthTokensResponse{}, model.User{}, err
	}
	access, expiresIn, err := s.newAccessToken(u)
	if err != nil {
		return model.AuthTokensResponse{}, model.User{}, err
	}
	return model.AuthTokensResponse{AccessToken: access, RefreshToken: newRawRefresh, TokenType: "Bearer", ExpiresIn: expiresIn}, u, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, req model.LogoutRequest) error {
	if err := s.validate.Struct(req); err != nil {
		return err
	}
	if err := s.repo.RevokeRefreshTokenForUser(ctx, userID, hashToken(req.RefreshToken)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnauthorized
		}
		return err
	}
	return nil
}
