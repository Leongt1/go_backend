package service

import (
	"backend-go/internal/auth/domain"
	"backend-go/internal/categories"
	categoryDomain "backend-go/internal/categories/domain"
	"backend-go/internal/platform/email"
	platformErrors "backend-go/internal/platform/errors"
	"backend-go/internal/platform/security"
	userDomain "backend-go/internal/users/domain"
	"backend-go/internal/users/service"
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	users             *service.Service
	jwt               *security.JWTManager
	refreshRepo       domain.RefreshTokenRepository
	passwordResetRepo domain.PasswordResetRepository

	categoryRepo categoryDomain.CategoryRepository

	emailProvider email.Provider

	accessTTL        time.Duration
	refreshTTL       time.Duration
	resetPasswordTTL time.Duration
}

func NewService(
	users *service.Service,
	jwt *security.JWTManager,
	refreshRepo domain.RefreshTokenRepository,
	passwordResetRepo domain.PasswordResetRepository,
	categoryRepo categoryDomain.CategoryRepository,
	emailProvider email.Provider,
	accessTTL,
	refreshTTL,
	resetPasswordTTL time.Duration,
) *Service {
	return &Service{
		users:             users,
		jwt:               jwt,
		refreshRepo:       refreshRepo,
		passwordResetRepo: passwordResetRepo,
		categoryRepo:      categoryRepo,
		emailProvider:     emailProvider,
		accessTTL:         accessTTL,
		refreshTTL:        refreshTTL,
		resetPasswordTTL:  resetPasswordTTL,
	}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (s *Service) Login(ctx context.Context, req *LoginInput) (*LoginOutput, error) {
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

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil

}

func (s *Service) Refresh(ctx context.Context, refreshTokenStr string) (*LoginOutput, error) {
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

	return &LoginOutput{
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
	Name        string
	Email       string
	Password    string
	Role        string
	Gender      string
	DateOfBirth *time.Time
}

func (s *Service) Signup(ctx context.Context, req *SignupInput) error {
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	if name == "" || email == "" || password == "" {
		return domain.ErrInvalidInput
	}

	role, err := userDomain.ParseRole(req.Role)
	if err != nil {
		return err
	}

	gender, err := userDomain.ParseGender(req.Gender)
	if err != nil {
		return err
	}

	createUser := &service.CreateInput{
		Name:        name,
		Email:       email,
		Password:    password,
		Role:        role,
		Gender:      gender,
		DateOfBirth: req.DateOfBirth,
	}

	if err = s.users.CreateUser(ctx, createUser); err != nil {
		return err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return err
	}

	// seed categories
	if err := categories.SeedCategoriesForUser(ctx, user.ID, s.categoryRepo); err != nil {
		return err
	}

	return nil
}

type ForgotPasswordInput struct {
	Email string
}

func (s *Service) ForgotPassword(ctx context.Context, req *ForgotPasswordInput) error {
	if req.Email == "" {
		return nil // user not found
	}

	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	if err := s.passwordResetRepo.DeletePasswordResetTokensByUserID(ctx, user.ID); err != nil {
		return err
	}

	resetToken, err := security.GenerateSecureToken()
	if err != nil {
		return err
	}

	hashedToken := security.HashToken(resetToken)
	if passwordResetToken := domain.NewPasswordResetToken(user.ID, hashedToken, s.resetPasswordTTL); passwordResetToken != nil {
		if err := s.passwordResetRepo.CreatePasswordResetToken(ctx, passwordResetToken); err != nil {
			return err
		}
	}

	link := fmt.Sprint(`http://localhost:8080/api/v1/auth/reset-password?token=` + resetToken)

	return s.emailProvider.Send(ctx, email.SendEmailInput{
		To:      user.Email,
		Subject: "Reset your password",
		HTML:    email.PasswordResetHTML(link),
	})
}

type PasswordResetInput struct {
	ResetToken string
	Password   string
}

func (s *Service) PasswordReset(ctx context.Context, req *PasswordResetInput) error {
	hashedToken := security.HashToken(req.ResetToken)
	resetToken, err := s.passwordResetRepo.GetPasswordResetTokenByHash(ctx, hashedToken)
	if err != nil {
		return err
	}

	if resetToken == nil {
		return domain.ErrInvalidPasswordResetToken
	}

	hasedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return platformErrors.ErrHashPassword
	}

	if err := s.users.UpdatePassword(ctx, resetToken.UserID, string(hasedPassword)); err != nil {
		return err
	}

	return s.passwordResetRepo.MarkPasswordResetTokenUsed(ctx, resetToken.ID)
}
