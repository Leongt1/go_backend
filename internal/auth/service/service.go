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

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	users             *service.Service
	jwt               *security.JWTManager
	frontendBaseURL   string
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
	frontendBaseURL string,
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
		frontendBaseURL:   frontendBaseURL,
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

	// prune this user's expired tokens while we're here
	if err := s.refreshRepo.DeleteExpiredByUser(ctx, user.ID); err != nil {
		return nil, err
	}

	// generate refresh token string (long-lived); only its hash is stored.
	// each login starts a new rotation family.
	refreshTokenStr, err := security.GenerateSecureToken()
	if err != nil {
		return nil, err
	}

	refreshToken := domain.NewRefreshToken(
		user.ID,
		security.HashToken(refreshTokenStr),
		uuid.New(),
		s.refreshTTL,
	)

	// Storing in DB
	err = s.refreshRepo.Create(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// cap concurrent sessions per user (oldest beyond the cap get revoked)
	if err := s.refreshRepo.RevokeActiveBeyondCap(ctx, user.ID, maxActiveRefreshTokens); err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil

}

// maxActiveRefreshTokens caps concurrent sessions (rotation families) per user.
const maxActiveRefreshTokens = 5

func (s *Service) Refresh(ctx context.Context, refreshTokenStr string) (*LoginOutput, error) {
	// fetch from DB (only hashes are stored)
	token, err := s.refreshRepo.GetByTokenHash(ctx, security.HashToken(refreshTokenStr))
	if err != nil {
		return nil, err
	}

	// reuse detection: a revoked token being replayed means the rotation chain
	// leaked - kill the whole family so a stolen descendant dies too
	if token.Revoked {
		if err := s.refreshRepo.RevokeFamily(ctx, token.FamilyID); err != nil {
			return nil, err
		}
		return nil, domain.ErrInvalidRefreshToken
	}
	if token.IsExpired() {
		if err := s.refreshRepo.Revoke(ctx, token.ID); err != nil {
			return nil, err
		}
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

	// create new refresh token in the same rotation family
	newRefresh := domain.NewRefreshToken(
		user.ID,
		security.HashToken(newRefreshStr),
		token.FamilyID,
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
	token, err := s.refreshRepo.GetByTokenHash(ctx, security.HashToken(refreshTokenStr))
	if err != nil {
		return err
	}

	if token.Revoked || token.IsExpired() {
		return domain.ErrInvalidRefreshToken
	}

	// end the whole session chain, not just the current link
	return s.refreshRepo.RevokeFamily(ctx, token.FamilyID)
}

type SignupInput struct {
	Name        string
	Email       string
	Password    string
	Gender      string
	DateOfBirth *time.Time
}

// validatePassword enforces the minimum password policy shared by signup and reset.
func validatePassword(password string) error {
	if len(password) < 8 {
		return domain.ErrWeakPassword
	}
	return nil
}

func (s *Service) Signup(ctx context.Context, req *SignupInput) error {
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	if name == "" || email == "" || password == "" {
		return domain.ErrInvalidInput
	}

	if err := validatePassword(password); err != nil {
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
		Role:        userDomain.RoleUser, // public signup can never grant elevated roles
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
	passwordResetToken := domain.NewPasswordResetToken(user.ID, hashedToken, s.resetPasswordTTL)
	if err := s.passwordResetRepo.CreatePasswordResetToken(ctx, passwordResetToken); err != nil {
		return err
	}

	// link := fmt.Sprint(`http://localhost:8080/api/v1/auth/reset-password?token=` + resetToken)
	link := fmt.Sprintf("%s/reset-password?token=%s", s.frontendBaseURL, resetToken)

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
	if err := validatePassword(req.Password); err != nil {
		return err
	}

	hashedToken := security.HashToken(req.ResetToken)
	resetToken, err := s.passwordResetRepo.GetPasswordResetTokenByHash(ctx, hashedToken)
	if err != nil {
		return err
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
