// Package dashboard composes read-only counts from the other modules into
// one summary for the admin overview — it owns no data of its own.
package dashboard

import (
	"context"

	"github.com/google/uuid"
)

type Summary struct {
	TotalPatients     int64
	TotalEncounters   int64
	TotalFacilities   int64
	TotalUsers        int64
	ReferralsByStatus map[string]int64

	// MostActiveDoctors is always populated (any caller who can reach
	// GET /dashboard/summary sees it) — it's the admin overview's "most
	// active doctors" panel, ranking providers by how many referrals
	// they've received. See Service.GetSummary.
	MostActiveDoctors []DoctorActivity

	// PendingConsultCount, MyReferralsPendingCount, and PatientsTodayCount
	// are the physician-specific view: populated only when the caller's
	// role is platform.RolePhysician (left at zero value for every other
	// role, including system_admin) — see Service.GetSummary.
	PendingConsultCount     int64
	MyReferralsPendingCount int64
	PatientsTodayCount      int64
}

// DoctorActivity is one row of the admin dashboard's "most active doctors"
// panel — how many referrals a provider has received, plus their display
// name.
type DoctorActivity struct {
	ProviderID    uuid.UUID
	FullName      *string
	ReferralCount int64
}

type PatientCounter interface {
	CountPatients(ctx context.Context) (int64, error)
}

type EncounterCounter interface {
	CountEncounters(ctx context.Context) (int64, error)
}

type FacilityCounter interface {
	CountFacilities(ctx context.Context) (int64, error)
}

type UserCounter interface {
	CountUsers(ctx context.Context) (int64, error)
}

type ReferralCounter interface {
	CountByStatus(ctx context.Context) (map[string]int64, error)
}

// DoctorActivityCounter powers MostActiveDoctors — satisfied by
// *referrals.Service via a small adapter defined in main.go (referrals
// returns its own ProviderReferralActivity type rather than importing this
// package, see that type's doc comment).
type DoctorActivityCounter interface {
	MostActiveDoctors(ctx context.Context, limit int) ([]DoctorActivity, error)
}

// ReferralPendingCounter powers MyReferralsPendingCount — satisfied by
// *referrals.Service.
type ReferralPendingCounter interface {
	CountPendingReferralsForProvider(ctx context.Context, providerID uuid.UUID) (int64, error)
}

// ConsultPendingCounter powers PendingConsultCount — satisfied by
// *consults.Service. Optional (wired via Service.SetConsultPendingCounter
// once consults.Service exists, since it's constructed after dashboard in
// main.go's dependency order).
type ConsultPendingCounter interface {
	CountPendingConsultsForProvider(ctx context.Context, providerID uuid.UUID) (int64, error)
}

// EncounterTodayCounter powers PatientsTodayCount — satisfied by
// *records.Service.
type EncounterTodayCounter interface {
	CountEncountersForProviderToday(ctx context.Context, providerID uuid.UUID) (int64, error)
}
