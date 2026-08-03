package search

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/consults"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
	"github.com/wadaag/health-platform/backend/internal/referrals"
)

func strPtr(s string) *string { return &s }

func TestWantsFacet(t *testing.T) {
	if !wantsFacet(nil, FacetPatient) {
		t.Error("empty types should want every facet")
	}
	if !wantsFacet([]string{FacetPatient, FacetDoctor}, FacetPatient) {
		t.Error("explicit types should want a facet it lists")
	}
	if wantsFacet([]string{FacetPatient}, FacetDoctor) {
		t.Error("explicit types should not want a facet it omits")
	}
}

func TestCanSearchPatients(t *testing.T) {
	cases := []struct {
		role platform.Role
		want bool
	}{
		{platform.RoleSystemAdmin, true},
		{platform.RolePhysician, true},
		{platform.RoleHospitalAdmin, true},
		{platform.RolePatient, false},
		{platform.RoleLabTech, false},
		{platform.RolePharmacist, false},
		{platform.RoleInsurer, false},
	}
	for _, c := range cases {
		if got := canSearchPatients(c.role); got != c.want {
			t.Errorf("canSearchPatients(%v) = %v, want %v", c.role, got, c.want)
		}
	}
}

func TestCanSearchReferrals(t *testing.T) {
	facilityID := uuid.New()
	if !canSearchReferrals(platform.RoleSystemAdmin, nil) {
		t.Error("system_admin should always be able to search referrals")
	}
	if canSearchReferrals(platform.RolePhysician, nil) {
		t.Error("a non-admin with no facility should not be able to search referrals")
	}
	if !canSearchReferrals(platform.RolePhysician, &facilityID) {
		t.Error("a non-admin with a facility should be able to search referrals")
	}
}

func TestCanSearchConsultations(t *testing.T) {
	providerID := uuid.New()
	if !canSearchConsultations(platform.RoleSystemAdmin, nil) {
		t.Error("system_admin should always be able to search consultations")
	}
	if canSearchConsultations(platform.RolePhysician, nil) {
		t.Error("a non-admin with no provider identity should not be able to search consultations")
	}
	if !canSearchConsultations(platform.RolePhysician, &providerID) {
		t.Error("a non-admin with a provider identity should be able to search consultations")
	}
}

func TestPatientResults(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	nationalID := "N123"
	phone := "555-0100"

	out := patientResults([]records.Patient{
		{ID: id, FullName: "Amina Yusuf", NationalID: &nationalID, Phone: &phone, CreatedAt: now},
	})
	if len(out) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out))
	}
	got := out[0]
	if got.Type != FacetPatient || got.ID != id || got.Title != "Amina Yusuf" || got.Subtitle != nationalID || got.CreatedAt != now {
		t.Errorf("unexpected result: %+v", got)
	}

	// national_id absent falls back to phone.
	out = patientResults([]records.Patient{{ID: id, FullName: "No National ID", Phone: &phone, CreatedAt: now}})
	if out[0].Subtitle != phone {
		t.Errorf("expected fallback to phone, got %q", out[0].Subtitle)
	}

	// neither present yields an empty subtitle, not a crash.
	out = patientResults([]records.Patient{{ID: id, FullName: "Neither", CreatedAt: now}})
	if out[0].Subtitle != "" {
		t.Errorf("expected empty subtitle, got %q", out[0].Subtitle)
	}
}

func TestDoctorResults(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	out := doctorResults([]facilities.ProviderSearchResult{
		{ProviderID: id, UserFullName: strPtr("Dr. Hassan"), Specialty: strPtr("Cardiology"), CreatedAt: now},
	})
	if len(out) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out))
	}
	got := out[0]
	if got.Type != FacetDoctor || got.ID != id || got.Title != "Dr. Hassan" || got.Subtitle != "Cardiology" {
		t.Errorf("unexpected result: %+v", got)
	}

	// missing full_name/specialty shouldn't panic, just empty strings.
	out = doctorResults([]facilities.ProviderSearchResult{{ProviderID: id, CreatedAt: now}})
	if out[0].Title != "" || out[0].Subtitle != "" {
		t.Errorf("expected empty title/subtitle, got %+v", out[0])
	}
}

func TestHospitalResults(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	out := hospitalResults([]facilities.Facility{
		{ID: id, Name: "Banadir Hospital", Region: strPtr("Banadir"), District: strPtr("Hodan"), CreatedAt: now},
	})
	if out[0].Subtitle != "Banadir, Hodan" {
		t.Errorf("expected combined region/district, got %q", out[0].Subtitle)
	}

	out = hospitalResults([]facilities.Facility{{ID: id, Name: "No Region", District: strPtr("Hodan"), CreatedAt: now}})
	if out[0].Subtitle != "Hodan" {
		t.Errorf("expected district-only fallback, got %q", out[0].Subtitle)
	}

	out = hospitalResults([]facilities.Facility{{ID: id, Name: "Neither", CreatedAt: now}})
	if out[0].Subtitle != "" {
		t.Errorf("expected empty subtitle, got %q", out[0].Subtitle)
	}
}

func TestReferralResults(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	out := referralResults([]referrals.Referral{
		{ID: id, Reason: "Suspected fracture", Status: referrals.StatusRouted, CreatedAt: now},
	})
	if out[0].Type != FacetReferral || out[0].Title != "Suspected fracture" || out[0].Subtitle != string(referrals.StatusRouted) {
		t.Errorf("unexpected result: %+v", out[0])
	}
}

func TestConsultationResults(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	out := consultationResults([]consults.Consultation{
		{ID: id, Reason: "Second opinion needed", Status: consults.StatusRequested, CreatedAt: now},
	})
	if out[0].Type != FacetConsultation || out[0].Title != "Second opinion needed" || out[0].Subtitle != string(consults.StatusRequested) {
		t.Errorf("unexpected result: %+v", out[0])
	}
}
