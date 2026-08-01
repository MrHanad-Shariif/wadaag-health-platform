package dashboard

import (
	"context"
	"fmt"
)

type Service struct {
	patients   PatientCounter
	encounters EncounterCounter
	facilities FacilityCounter
	users      UserCounter
	referrals  ReferralCounter
}

func NewService(patients PatientCounter, encounters EncounterCounter, facilities FacilityCounter, users UserCounter, referrals ReferralCounter) *Service {
	return &Service{patients: patients, encounters: encounters, facilities: facilities, users: users, referrals: referrals}
}

func (s *Service) GetSummary(ctx context.Context) (Summary, error) {
	patients, err := s.patients.CountPatients(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("count patients: %w", err)
	}
	encounters, err := s.encounters.CountEncounters(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("count encounters: %w", err)
	}
	facilities, err := s.facilities.CountFacilities(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("count facilities: %w", err)
	}
	users, err := s.users.CountUsers(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("count users: %w", err)
	}
	referralsByStatus, err := s.referrals.CountByStatus(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("count referrals: %w", err)
	}

	return Summary{
		TotalPatients: patients, TotalEncounters: encounters, TotalFacilities: facilities,
		TotalUsers: users, ReferralsByStatus: referralsByStatus,
	}, nil
}
