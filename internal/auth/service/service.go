package service

import (
	"backend-go/internal/auth/domain"
	"backend-go/internal/platform/security"
	"backend-go/internal/users/service"
	"context"
	"strings"
	"time"
)

type Service struct {
	users       *service.Service
	jwt         *security.JWTManager
	refreshRepo domain.RefreshTokenRepository
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

func NewService(
	users *service.Service,
	jwt *security.JWTManager,
	refreshRepo domain.RefreshTokenRepository,
	accessTTL, refreshTTL time.Duration,
) *Service {
	return &Service{
		users:       users,
		jwt:         jwt,
		refreshRepo: refreshRepo,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (s *Service) Login(ctx context.Context, req *LoginInput) (*LoginResponse, error) {
	// validation
	if req == nil {
		return nil, domain.ErrInvalidInput
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password // password is not trimmed

	if email == "" || password == "" {
		return nil, domain.ErrInvalidInput
	}

	// check for password match
	user, err := s.users.AuthenticateUser(ctx, email, password)
	if err != nil {
		return nil, err
	}

	// generate access token (short-lived)
	accessToken, err := s.jwt.GenerateToken(user.ID, string(user.Role), s.accessTTL)
	if err != nil {
		return nil, err
	}

	// generate refresh token string (long-lived)
	refreshTokenStr, err := security.GenerateSecureToken()
	if err != nil {
		return nil, err
	}

	refreshToken := domain.NewRefreshToken(
		user.ID,
		refreshTokenStr,
		s.refreshTTL,
	)

	// Storing in DB
	err = s.refreshRepo.Create(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil

}

func (s *Service) Refresh(ctx context.Context, refreshTokenStr string) (*LoginResponse, error) {
	// fetch from DB
	token, err := s.refreshRepo.GetByToken(ctx, refreshTokenStr)
	if err != nil {
		return nil, err
	}

	// validate
	if token.IsExpired() {
		_ = s.refreshRepo.Revoke(ctx, token.ID) // revoke the invalid/expired token
		return nil, domain.ErrInvalidRefreshToken
	}
	if token.Revoked {
		return nil, domain.ErrInvalidRefreshToken
	}

	// generate new access token
	// IMPORTANT: fetch user to get role
	user, err := s.users.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, err
	}

	// generate new access token
	accessToken, err := s.jwt.GenerateToken(user.ID, string(user.Role), s.accessTTL)
	if err != nil {
		return nil, err
	}

	// generate new refresh token string (long-lived)
	// rotation
	newRefreshStr, err := security.GenerateSecureToken()
	if err != nil {
		return nil, err
	}

	// create new refresh token
	newRefresh := domain.NewRefreshToken(
		user.ID,
		newRefreshStr,
		s.refreshTTL,
	)

	// revoke old
	if err := s.refreshRepo.Revoke(ctx, token.ID); err != nil {
		return nil, err
	}

	// store new
	if err := s.refreshRepo.Create(ctx, newRefresh); err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshStr,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshTokenStr string) error {
	token, err := s.refreshRepo.GetByToken(ctx, refreshTokenStr)
	if err != nil {
		return err
	}

	if token.Revoked || token.IsExpired() {
		return domain.ErrInvalidRefreshToken
	}

	return s.refreshRepo.Revoke(ctx, token.ID)
}

type SignupInput struct {
	Email    string
	Password string
	Role     string
	Gender   string
}

func (s *Service) Signup(ctx context.Context, req *SignupInput) (string, error) {
	return "", nil
}
