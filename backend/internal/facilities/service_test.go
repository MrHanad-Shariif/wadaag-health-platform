package facilities

import (
	"fmt"
	"reflect"
	"testing"
)

// strPtr is a small test helper since Go has no address-of-literal syntax.
func strPtr(s string) *string { return &s }

// TestMergeFacilityUpdate covers the partial-update (PATCH) merge logic for
// a facility's logo/working-hours: fields omitted from the request (nil)
// must fall back to whatever is already on the row, not be wiped to empty.
// Mirrors identity.TestMergeProfileUpdate.
func TestMergeFacilityUpdate(t *testing.T) {
	existing := Facility{
		LogoURL:      strPtr("https://example.com/old-logo.png"),
		WorkingHours: []byte(`{"mon":{"open":"08:00","close":"17:00"}}`),
	}

	tests := []struct {
		name             string
		current          Facility
		in               UpdateFacilityInput
		wantLogoURL      *string
		wantWorkingHours []byte
	}{
		{
			name:             "empty input leaves every existing field unchanged",
			current:          existing,
			in:               UpdateFacilityInput{},
			wantLogoURL:      strPtr("https://example.com/old-logo.png"),
			wantWorkingHours: []byte(`{"mon":{"open":"08:00","close":"17:00"}}`),
		},
		{
			name:             "supplying only logo_url leaves working_hours untouched",
			current:          existing,
			in:               UpdateFacilityInput{LogoURL: strPtr("https://example.com/new-logo.png")},
			wantLogoURL:      strPtr("https://example.com/new-logo.png"),
			wantWorkingHours: []byte(`{"mon":{"open":"08:00","close":"17:00"}}`),
		},
		{
			name: "supplying only working_hours leaves logo_url untouched",
			current: existing,
			in: UpdateFacilityInput{
				WorkingHours: func() *[]byte {
					wh := []byte(`{"tue":{"open":"09:00","close":"18:00"}}`)
					return &wh
				}(),
			},
			wantLogoURL:      strPtr("https://example.com/old-logo.png"),
			wantWorkingHours: []byte(`{"tue":{"open":"09:00","close":"18:00"}}`),
		},
		{
			name:             "no existing values (zero-value current) with a partial update only sets the supplied field",
			current:          Facility{},
			in:               UpdateFacilityInput{LogoURL: strPtr("https://example.com/logo.png")},
			wantLogoURL:      strPtr("https://example.com/logo.png"),
			wantWorkingHours: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logoURL, workingHours := mergeFacilityUpdate(tt.current, tt.in)

			if !reflect.DeepEqual(logoURL, tt.wantLogoURL) {
				t.Errorf("logoURL = %v, want %v", derefOrNil(logoURL), derefOrNil(tt.wantLogoURL))
			}
			if !reflect.DeepEqual(workingHours, tt.wantWorkingHours) {
				t.Errorf("workingHours = %s, want %s", workingHours, tt.wantWorkingHours)
			}
		})
	}
}

func derefOrNil(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func intPtr(i int) *int { return &i }

// TestMergeProviderProfileUpdate covers the partial-update (PATCH) merge
// logic for a provider editing their own profile: fields omitted from the
// request (nil) must fall back to whatever is already on the row, not be
// wiped to empty. Mirrors TestMergeFacilityUpdate /
// identity.TestMergeProfileUpdate.
func TestMergeProviderProfileUpdate(t *testing.T) {
	existing := Provider{
		Qualifications:     []string{"MBBS"},
		YearsExperience:    intPtr(5),
		ConsultationFee:    strPtr("50.00"),
		VerificationStatus: "verified",
		Languages:          []string{"so", "en"},
		Certificates:       []byte(`[{"name":"Board Certification"}]`),
		AreasOfExpertise:   []string{"cardiology"},
	}

	tests := []struct {
		name                 string
		current              Provider
		in                   UpdateProviderProfileInput
		wantQualifications   []string
		wantYearsExperience  *int
		wantConsultationFee  *string
		wantLanguages        []string
		wantCertificates     []byte
		wantAreasOfExpertise []string
	}{
		{
			name:                 "empty input leaves every existing field unchanged",
			current:              existing,
			in:                   UpdateProviderProfileInput{},
			wantQualifications:   []string{"MBBS"},
			wantYearsExperience:  intPtr(5),
			wantConsultationFee:  strPtr("50.00"),
			wantLanguages:        []string{"so", "en"},
			wantCertificates:     []byte(`[{"name":"Board Certification"}]`),
			wantAreasOfExpertise: []string{"cardiology"},
		},
		{
			name:    "supplying only years_experience leaves other fields untouched",
			current: existing,
			in: UpdateProviderProfileInput{
				YearsExperience: intPtr(7),
			},
			wantQualifications:   []string{"MBBS"},
			wantYearsExperience:  intPtr(7),
			wantConsultationFee:  strPtr("50.00"),
			wantLanguages:        []string{"so", "en"},
			wantCertificates:     []byte(`[{"name":"Board Certification"}]`),
			wantAreasOfExpertise: []string{"cardiology"},
		},
		{
			name:    "supplying qualifications and consultation_fee updates only those",
			current: existing,
			in: UpdateProviderProfileInput{
				Qualifications:  &[]string{"MBBS", "MD"},
				ConsultationFee: strPtr("75.00"),
			},
			wantQualifications:   []string{"MBBS", "MD"},
			wantYearsExperience:  intPtr(5),
			wantConsultationFee:  strPtr("75.00"),
			wantLanguages:        []string{"so", "en"},
			wantCertificates:     []byte(`[{"name":"Board Certification"}]`),
			wantAreasOfExpertise: []string{"cardiology"},
		},
		{
			name:    "no existing values (zero-value current) with a partial update only sets the supplied field",
			current: Provider{},
			in: UpdateProviderProfileInput{
				Languages: &[]string{"en"},
			},
			wantQualifications:   nil,
			wantYearsExperience:  nil,
			wantConsultationFee:  nil,
			wantLanguages:        []string{"en"},
			wantCertificates:     nil,
			wantAreasOfExpertise: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qualifications, yearsExperience, consultationFee, languages, certificates, areasOfExpertise := mergeProviderProfileUpdate(tt.current, tt.in)

			if !reflect.DeepEqual(qualifications, tt.wantQualifications) {
				t.Errorf("qualifications = %v, want %v", qualifications, tt.wantQualifications)
			}
			if !reflect.DeepEqual(yearsExperience, tt.wantYearsExperience) {
				t.Errorf("yearsExperience = %v, want %v", derefIntOrNil(yearsExperience), derefIntOrNil(tt.wantYearsExperience))
			}
			if !reflect.DeepEqual(consultationFee, tt.wantConsultationFee) {
				t.Errorf("consultationFee = %v, want %v", derefOrNil(consultationFee), derefOrNil(tt.wantConsultationFee))
			}
			if !reflect.DeepEqual(languages, tt.wantLanguages) {
				t.Errorf("languages = %v, want %v", languages, tt.wantLanguages)
			}
			if !reflect.DeepEqual(certificates, tt.wantCertificates) {
				t.Errorf("certificates = %s, want %s", certificates, tt.wantCertificates)
			}
			if !reflect.DeepEqual(areasOfExpertise, tt.wantAreasOfExpertise) {
				t.Errorf("areasOfExpertise = %v, want %v", areasOfExpertise, tt.wantAreasOfExpertise)
			}
		})
	}
}

func derefIntOrNil(i *int) string {
	if i == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d", *i)
}

// TestUpdateProviderVerificationStatusValidation covers the input
// validation UpdateProviderVerificationStatus performs before ever reaching
// the repository — invalid statuses must be rejected with
// ErrInvalidVerificationStatus rather than falling through to a raw DB
// constraint violation.
func TestUpdateProviderVerificationStatusValidation(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{"pending", true},
		{"verified", true},
		{"rejected", true},
		{"approved", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := validVerificationStatuses[tt.status]; got != tt.valid {
				t.Errorf("validVerificationStatuses[%q] = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}
