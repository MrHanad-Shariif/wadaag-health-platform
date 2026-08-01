package facilities

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateFacilityInput struct {
	Name     string
	Type     Type
	Region   *string
	District *string
	Phone    *string
	Address  *string
}

func (s *Service) CreateFacility(ctx context.Context, in CreateFacilityInput) (Facility, error) {
	return s.repo.CreateFacility(ctx, in.Name, in.Type, in.Region, in.District, in.Phone, in.Address)
}

func (s *Service) ListFacilities(ctx context.Context) ([]Facility, error) {
	return s.repo.ListFacilities(ctx)
}

func (s *Service) CountFacilities(ctx context.Context) (int64, error) {
	return s.repo.CountFacilities(ctx)
}

func (s *Service) GetFacility(ctx context.Context, id uuid.UUID) (Facility, error) {
	return s.repo.FindFacilityByID(ctx, id)
}

type CreateProviderInput struct {
	UserID        uuid.UUID
	FacilityID    uuid.UUID
	Specialty     *string
	LicenseNumber *string
}

func (s *Service) CreateProvider(ctx context.Context, in CreateProviderInput) (Provider, error) {
	if _, err := s.repo.FindFacilityByID(ctx, in.FacilityID); err != nil {
		return Provider{}, err
	}
	return s.repo.CreateProvider(ctx, in.UserID, in.FacilityID, in.Specialty, in.LicenseNumber)
}

// FacilityIDForUser implements identity.FacilityResolver, letting identity
// populate the facility claim on a provider's JWT without depending on this
// package directly.
func (s *Service) FacilityIDForUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	provider, err := s.repo.FindProviderByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider.FacilityID, nil
}

// ResolveProviderID implements records.ProviderResolver — maps a user_id
// to the providers.id their encounters get recorded under.
func (s *Service) ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	provider, err := s.repo.FindProviderByUserID(ctx, userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	return provider.ID, nil
}
