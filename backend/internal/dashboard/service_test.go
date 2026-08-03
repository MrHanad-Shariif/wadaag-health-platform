package dashboard

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// fixedCounters is a fake implementation of every counter interface this
// package composes, each returning a fixed value — enough to exercise
// Service.GetSummary's role-gating logic (which fields get populated for
// which caller) without a database.
type fixedCounters struct {
	patients, encounters, facilities, users int64
	referralsByStatus                       map[string]int64
	mostActiveDoctors                       []DoctorActivity
	pendingReferrals, pendingConsults        int64
	patientsToday                           int64
}

func (f fixedCounters) CountPatients(context.Context) (int64, error)   { return f.patients, nil }
func (f fixedCounters) CountEncounters(context.Context) (int64, error) { return f.encounters, nil }
func (f fixedCounters) CountFacilities(context.Context) (int64, error) { return f.facilities, nil }
func (f fixedCounters) CountUsers(context.Context) (int64, error)      { return f.users, nil }
func (f fixedCounters) CountByStatus(context.Context) (map[string]int64, error) {
	return f.referralsByStatus, nil
}
func (f fixedCounters) MostActiveDoctors(context.Context, int) ([]DoctorActivity, error) {
	return f.mostActiveDoctors, nil
}
func (f fixedCounters) CountPendingReferralsForProvider(context.Context, uuid.UUID) (int64, error) {
	return f.pendingReferrals, nil
}
func (f fixedCounters) CountPendingConsultsForProvider(context.Context, uuid.UUID) (int64, error) {
	return f.pendingConsults, nil
}
func (f fixedCounters) CountEncountersForProviderToday(context.Context, uuid.UUID) (int64, error) {
	return f.patientsToday, nil
}

// TestGetSummaryDoctorFieldsGating covers which fields Service.GetSummary
// populates for which caller: the admin-facing fields (totals,
// MostActiveDoctors) are always populated, while the physician-specific
// fields (PendingConsultCount, MyReferralsPendingCount,
// PatientsTodayCount) are populated only for a RolePhysician caller with a
// successfully-resolved provider id — every other combination must leave
// them at zero, not silently compute them for a role they don't apply to.
func TestGetSummaryDoctorFieldsGating(t *testing.T) {
	fixed := fixedCounters{
		patients: 10, encounters: 20, facilities: 3, users: 5,
		referralsByStatus: map[string]int64{"routed": 2},
		mostActiveDoctors: []DoctorActivity{{ProviderID: uuid.New(), ReferralCount: 4}},
		pendingReferrals:  7, pendingConsults: 8, patientsToday: 9,
	}
	providerID := uuid.New()

	newService := func() *Service {
		s := NewService(fixed, fixed, fixed, fixed, fixed, fixed, fixed, fixed)
		s.SetConsultPendingCounter(fixed)
		return s
	}

	t.Run("admin fields are always populated", func(t *testing.T) {
		s := newService()
		summary, err := s.GetSummary(context.Background(), uuid.New(), platform.RoleSystemAdmin, nil)
		if err != nil {
			t.Fatalf("GetSummary() error = %v", err)
		}
		if summary.TotalPatients != 10 || summary.TotalEncounters != 20 || summary.TotalFacilities != 3 || summary.TotalUsers != 5 {
			t.Errorf("totals not populated correctly: %+v", summary)
		}
		if len(summary.MostActiveDoctors) != 1 {
			t.Errorf("MostActiveDoctors = %v, want 1 entry", summary.MostActiveDoctors)
		}
	})

	t.Run("non-physician caller gets zero-value doctor-specific fields", func(t *testing.T) {
		s := newService()
		summary, err := s.GetSummary(context.Background(), uuid.New(), platform.RoleSystemAdmin, &providerID)
		if err != nil {
			t.Fatalf("GetSummary() error = %v", err)
		}
		if summary.PendingConsultCount != 0 || summary.MyReferralsPendingCount != 0 || summary.PatientsTodayCount != 0 {
			t.Errorf("expected zero-value doctor-specific fields for system_admin, got %+v", summary)
		}
	})

	t.Run("physician caller with unresolved provider id gets zero-value doctor-specific fields", func(t *testing.T) {
		s := newService()
		summary, err := s.GetSummary(context.Background(), uuid.New(), platform.RolePhysician, nil)
		if err != nil {
			t.Fatalf("GetSummary() error = %v", err)
		}
		if summary.PendingConsultCount != 0 || summary.MyReferralsPendingCount != 0 || summary.PatientsTodayCount != 0 {
			t.Errorf("expected zero-value doctor-specific fields when actorProviderID is nil, got %+v", summary)
		}
	})

	t.Run("physician caller with a resolved provider id gets every doctor-specific field", func(t *testing.T) {
		s := newService()
		summary, err := s.GetSummary(context.Background(), uuid.New(), platform.RolePhysician, &providerID)
		if err != nil {
			t.Fatalf("GetSummary() error = %v", err)
		}
		if summary.PendingConsultCount != 8 || summary.MyReferralsPendingCount != 7 || summary.PatientsTodayCount != 9 {
			t.Errorf("doctor-specific fields not populated correctly: %+v", summary)
		}
	})

	t.Run("physician caller with no consults counter wired leaves PendingConsultCount zero", func(t *testing.T) {
		s := NewService(fixed, fixed, fixed, fixed, fixed, fixed, fixed, fixed)
		summary, err := s.GetSummary(context.Background(), uuid.New(), platform.RolePhysician, &providerID)
		if err != nil {
			t.Fatalf("GetSummary() error = %v", err)
		}
		if summary.PendingConsultCount != 0 {
			t.Errorf("PendingConsultCount = %d, want 0 when the consults dependency was never wired", summary.PendingConsultCount)
		}
		if summary.MyReferralsPendingCount != 7 || summary.PatientsTodayCount != 9 {
			t.Errorf("the other two doctor-specific fields should still populate independently of consults: %+v", summary)
		}
	})
}
