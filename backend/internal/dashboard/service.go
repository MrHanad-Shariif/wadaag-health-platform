package dashboard

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// mostActiveDoctorsLimit caps the admin dashboard's "most active doctors"
// panel — a top-5 leaderboard, not a full ranking of every provider.
const mostActiveDoctorsLimit = 5

type Service struct {
	patients   PatientCounter
	encounters EncounterCounter
	facilities FacilityCounter
	users      UserCounter
	referrals  ReferralCounter

	doctorActivity   DoctorActivityCounter
	referralsPending ReferralPendingCounter
	encountersToday  EncounterTodayCounter

	// consults is optional (nil by default) — set via
	// SetConsultPendingCounter once consults.Service exists in main.go's
	// wiring order (consults is constructed after dashboard). Nil-tolerant:
	// if never set, Summary.PendingConsultCount is simply left at zero.
	consults ConsultPendingCounter
}

func NewService(
	patients PatientCounter, encounters EncounterCounter, facilities FacilityCounter, users UserCounter, referrals ReferralCounter,
	doctorActivity DoctorActivityCounter, referralsPending ReferralPendingCounter, encountersToday EncounterTodayCounter,
) *Service {
	return &Service{
		patients: patients, encounters: encounters, facilities: facilities, users: users, referrals: referrals,
		doctorActivity: doctorActivity, referralsPending: referralsPending, encountersToday: encountersToday,
	}
}

// SetConsultPendingCounter wires the optional consults dependency — see the
// consults field's doc comment for why it isn't a NewService parameter.
func (s *Service) SetConsultPendingCounter(c ConsultPendingCounter) { s.consults = c }

// GetSummary composes the admin overview: the platform-wide counts every
// caller sees, plus a physician-specific slice (pending consults/referrals,
// patients seen today) populated only when actorRole is RolePhysician and
// actorProviderID resolved successfully — every other caller gets the zero
// value for those three fields. See dashboard.Handler.summary for how
// actorProviderID is resolved.
func (s *Service) GetSummary(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, actorProviderID *uuid.UUID) (Summary, error) {
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
	mostActiveDoctors, err := s.doctorActivity.MostActiveDoctors(ctx, mostActiveDoctorsLimit)
	if err != nil {
		return Summary{}, fmt.Errorf("most active doctors: %w", err)
	}

	summary := Summary{
		TotalPatients: patients, TotalEncounters: encounters, TotalFacilities: facilities,
		TotalUsers: users, ReferralsByStatus: referralsByStatus, MostActiveDoctors: mostActiveDoctors,
	}

	if actorRole == platform.RolePhysician && actorProviderID != nil {
		pendingReferrals, err := s.referralsPending.CountPendingReferralsForProvider(ctx, *actorProviderID)
		if err != nil {
			return Summary{}, fmt.Errorf("count pending referrals: %w", err)
		}
		patientsToday, err := s.encountersToday.CountEncountersForProviderToday(ctx, *actorProviderID)
		if err != nil {
			return Summary{}, fmt.Errorf("count patients seen today: %w", err)
		}
		summary.MyReferralsPendingCount = pendingReferrals
		summary.PatientsTodayCount = patientsToday

		if s.consults != nil {
			pendingConsults, err := s.consults.CountPendingConsultsForProvider(ctx, *actorProviderID)
			if err != nil {
				return Summary{}, fmt.Errorf("count pending consults: %w", err)
			}
			summary.PendingConsultCount = pendingConsults
		}
	}

	return summary, nil
}
