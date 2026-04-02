package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"finance-dashboard/internal/config"
	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/repository"
	jwtpkg "finance-dashboard/pkg/jwt"
	"finance-dashboard/pkg/password"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	tokenRepo *repository.TokenRepository
}

func NewAuthService(userRepo *repository.UserRepository, tokenRepo *repository.TokenRepository) *AuthService {
	return &AuthService{userRepo: userRepo, tokenRepo: tokenRepo}
}

type TokenPair struct {
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	User         *domain.UserResponse `json:"user"`
}

func (s *AuthService) Register(req *domain.CreateUserRequest) (*domain.UserResponse, error) {
	existing, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already in use")
	}

	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	return s.userRepo.Create(req.Name, req.Email, hash, 1) // always viewer on self-register
}

func (s *AuthService) Login(email, pass string) (*TokenPair, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil || !password.Verify(pass, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}
	if !user.IsActive {
		return nil, errors.New("account is disabled")
	}

	accessToken, err := jwtpkg.GenerateAccessToken(user.ID, user.RoleName, config.C.JWTSecret, config.C.JWTAccessExpiry)
	if err != nil {
		return nil, err
	}
	refreshToken, err := jwtpkg.GenerateRefreshToken(user.ID, config.C.JWTSecret, config.C.JWTRefreshExpiry)
	if err != nil {
		return nil, err
	}

	hash := hashToken(refreshToken)
	expiresAt := time.Now().Add(config.C.JWTRefreshExpiry)
	if err := s.tokenRepo.Store(user.ID, hash, expiresAt); err != nil {
		return nil, err
	}

	resp := &domain.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.RoleName,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}
	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, User: resp}, nil
}

func (s *AuthService) Refresh(refreshToken string) (string, error) {
	hash := hashToken(refreshToken)
	t, err := s.tokenRepo.FindByHash(hash)
	if err != nil {
		return "", err
	}
	if t == nil || t.Revoked || time.Now().After(t.ExpiresAt) {
		return "", errors.New("invalid or expired refresh token")
	}

	claims, err := jwtpkg.ValidateToken(refreshToken, config.C.JWTSecret)
	if err != nil {
		return "", err
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil || user == nil {
		return "", errors.New("user not found")
	}

	return jwtpkg.GenerateAccessToken(user.ID, user.Role, config.C.JWTSecret, config.C.JWTAccessExpiry)
}

func (s *AuthService) Logout(refreshToken string) error {
	hash := hashToken(refreshToken)
	return s.tokenRepo.Revoke(hash)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
