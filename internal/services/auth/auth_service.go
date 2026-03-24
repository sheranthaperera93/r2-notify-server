package authService

import (
	"context"
	"errors"
	"fmt"

	"github.com/sheranthaperera93/r2-notify-server/internal/config"
	"github.com/sheranthaperera93/r2-notify-server/internal/logger"
	tokenRepo "github.com/sheranthaperera93/r2-notify-server/internal/repository/token"
	userRepo "github.com/sheranthaperera93/r2-notify-server/internal/repository/user"
	emailService "github.com/sheranthaperera93/r2-notify-server/internal/services/email"
	"github.com/sheranthaperera93/r2-notify-server/pkg/token"
	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"golang.org/x/crypto/bcrypt"
)

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type AuthService struct {
	userRepo  userRepo.UserRepository
	tokenRepo tokenRepo.TokenRepository
	email     *emailService.EmailService
}

func NewAuthService(
	userRepo userRepo.UserRepository,
	tokenRepo tokenRepo.TokenRepository,
	emailSvc *emailService.EmailService,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		email:     emailSvc,
	}
}

func (s *AuthService) Register(username, email, password string) error {
	cfg := config.LoadConfig()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.userRepo.Create(username, email, string(hash))
	if err != nil {
		return err
	}

	// Create Unkey Identity so API keys can be linked to this user.
	unkeyClient := unkey.New(unkey.WithSecurity(cfg.UnkeyRootKey))
	resp, err := unkeyClient.Identities.CreateIdentity(
		context.Background(),
		components.V2IdentitiesCreateIdentityRequestBody{ExternalID: user.Username},
	)
	if err != nil || resp.V2IdentitiesCreateIdentityResponseBody == nil {
		logger.Log.Warn(logger.LogPayload{
			Component: "AuthService",
			Operation: "Register",
			Message:   "failed to create Unkey identity for user " + user.ID,
			Error:     err,
		})
	} else {
		externalID := resp.V2IdentitiesCreateIdentityResponseBody.Data.IdentityID
		if err := s.userRepo.SetExternalID(user.ID, externalID); err != nil {
			logger.Log.Warn(logger.LogPayload{
				Component: "AuthService",
				Operation: "Register",
				Message:   "failed to persist Unkey identity ID for user " + user.ID,
				Error:     err,
			})
		}
	}

	verifyToken, err := s.tokenRepo.CreateVerifyToken(user.ID)
	if err != nil {
		return fmt.Errorf("failed to create verification token: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/api-keys/verify-email?token=%s", cfg.AppBaseURL, verifyToken)
	if err := s.email.SendVerification(user.Email, verifyURL); err != nil {
		logger.Log.Warn(logger.LogPayload{
			Component: "AuthService",
			Operation: "Register",
			Message:   "failed to send verification email to " + user.Email,
			Error:     err,
		})
	}

	return nil
}

func (s *AuthService) VerifyEmail(verifyToken string) error {
	userID, err := s.tokenRepo.ConsumeVerifyToken(verifyToken)
	if err != nil {
		return err
	}
	return s.userRepo.SetVerified(userID)
}

func (s *AuthService) Login(username, password string) (*AuthTokens, error) {
	cfg := config.LoadConfig()

	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, userRepo.ErrNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.Verified {
		return nil, errors.New("email address not verified")
	}

	accessToken, err := token.GenerateAccessToken(user.ID, user.Username, user.Email, cfg.JWTSecret, cfg.JWTAccessExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenRepo.CreateRefreshToken(user.ID, cfg.JWTRefreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(cfg.JWTAccessExpiry.Seconds()),
	}, nil
}

func (s *AuthService) Refresh(refreshToken string) (*AuthTokens, error) {
	cfg := config.LoadConfig()

	userID, err := s.tokenRepo.ConsumeRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	accessToken, err := token.GenerateAccessToken(user.ID, user.Username, user.Email, cfg.JWTSecret, cfg.JWTAccessExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.tokenRepo.CreateRefreshToken(user.ID, cfg.JWTRefreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(cfg.JWTAccessExpiry.Seconds()),
	}, nil
}

func (s *AuthService) Logout(refreshToken string) error {
	return s.tokenRepo.DeleteRefreshToken(refreshToken)
}

func (s *AuthService) ForgotPassword(email string) error {
	cfg := config.LoadConfig()

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// Always return nil — don't leak whether the email exists.
		return nil
	}

	resetToken, err := s.tokenRepo.CreateResetToken(user.ID)
	if err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", cfg.AppBaseURL, resetToken)
	return s.email.SendPasswordReset(user.Email, resetURL)
}

func (s *AuthService) ResetPassword(resetToken, newPassword string) error {
	userID, err := s.tokenRepo.ConsumeResetToken(resetToken)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(userID, string(hash)); err != nil {
		return err
	}

	return s.tokenRepo.DeleteAllRefreshTokens(userID)
}
