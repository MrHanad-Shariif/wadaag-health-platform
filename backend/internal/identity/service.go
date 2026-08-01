package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email/phone or password")

// FacilityResolver lets identity populate a provider's facility claim on
// login without depending on the facilities module's internals — it's
// implemented by facilities.Service and wired in main.go.
type FacilityResolver interface {
	FacilityIDForUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
}

type Service struct {
	repo       *Repository
	tm         *platform.TokenManager
	facilities FacilityResolver
}

func NewService(repo *Repository, tm *platform.TokenManager, facilities FacilityResolver) *Service {
	return &Service{repo: repo, tm: tm, facilities: facilities}
}

type RegisterInput struct {
	Email    *string
	Phone    *string
	Password string
	Role     platform.Role
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {
	if in.Email == nil && in.Phone == nil {
		return User{}, fmt.Errorf("email or phone is required")
	}
	if len(in.Password) < 8 {
		return User{}, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	return s.repo.CreateUser(ctx, in.Email, in.Phone, string(hash), in.Role)
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *Service) Login(ctx context.Context, identifier, password string) (User, TokenPair, error) {
	user, err := s.repo.FindByEmailOrPhone(ctx, identifier)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, TokenPair{}, ErrInvalidCredentials
		}
		return User{}, TokenPair{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return User{}, TokenPair{}, ErrInvalidCredentials
	}

	facilityID, err := s.facilities.FacilityIDForUser(ctx, user.ID)
	if err != nil {
		return User{}, TokenPair{}, fmt.Errorf("resolve facility: %w", err)
	}

	tokens, err := s.issueTokens(user, facilityID)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	return user, tokens, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) issueTokens(user User, facilityID *uuid.UUID) (TokenPair, error) {
	access, err := s.tm.IssueAccessToken(user.ID, user.Role, facilityID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}

	refresh, err := s.tm.IssueRefreshToken(user.ID, user.Role, facilityID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue refresh token: %w", err)
	}

	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
